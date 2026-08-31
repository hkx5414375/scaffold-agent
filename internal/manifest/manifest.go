// Package manifest records the files owned by Scaffold Agent capability packs.
package manifest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
)

var (
	resourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	semverPattern       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

const (
	// APIVersionV1Alpha1 is the first ownership manifest contract.
	APIVersionV1Alpha1 = "scaffold-agent.io/manifest/v1alpha1"
	// Path is reserved for the Engine-owned project manifest.
	Path = ".scaffold-agent/manifest.json"
	// Owner identifies metadata written by the Engine itself.
	Owner = "scaffold-agent-engine"
)

// Document describes the exact generated state currently owned by Scaffold Agent.
type Document struct {
	APIVersion     string            `json:"api_version"`
	BlueprintHash  string            `json:"blueprint_hash"`
	CapabilityLock map[string]string `json:"capability_lock"`
	Files          map[string]File   `json:"files"`
}

// File is the last applied hash and capability owner of one generated file.
type File struct {
	Owner string `json:"owner"`
	Hash  string `json:"hash"`
}

// Loaded includes the decoded document and the exact bytes used as a transaction precondition.
type Loaded struct {
	Document Document
	Content  []byte
	Hash     string
	Exists   bool
}

// Empty returns a valid manifest with initialized maps.
func Empty() Document {
	return Document{
		APIVersion:     APIVersionV1Alpha1,
		CapabilityLock: map[string]string{},
		Files:          map[string]File{},
	}
}

// Load reads and strictly validates the ownership manifest. A missing manifest is not an error.
func Load(root projectfs.Root) (Loaded, error) {
	target, err := root.Resolve(Path)
	if err != nil {
		return Loaded{}, err
	}
	content, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return Loaded{Document: Empty()}, nil
	}
	if err != nil {
		return Loaded{}, fmt.Errorf("read ownership manifest: %w", err)
	}
	value, err := Decode(content, root)
	if err != nil {
		return Loaded{}, fmt.Errorf("decode ownership manifest: %w", err)
	}
	return Loaded{
		Document: value,
		Content:  append([]byte(nil), content...),
		Hash:     projectfs.HashBytes(content),
		Exists:   true,
	}, nil
}

// Decode strictly decodes and validates one manifest document.
func Decode(content []byte, root projectfs.Root) (Document, error) {
	if err := rejectDuplicateObjectKeys(content); err != nil {
		return Document{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var value Document
	if err := decoder.Decode(&value); err != nil {
		return Document{}, err
	}
	if err := requireEOF(decoder.Decode(new(any))); err != nil {
		return Document{}, err
	}
	if err := Validate(value, root); err != nil {
		return Document{}, err
	}
	return value, nil
}

// Validate checks that all ownership claims are portable and complete.
func Validate(value Document, root projectfs.Root) error {
	if value.APIVersion != APIVersionV1Alpha1 {
		return fmt.Errorf("api_version must be %q", APIVersionV1Alpha1)
	}
	if !isSHA256(value.BlueprintHash) {
		return errors.New("blueprint_hash must be a lowercase SHA-256 hash")
	}
	if value.CapabilityLock == nil {
		return errors.New("capability_lock must be an object")
	}
	if value.Files == nil {
		return errors.New("files must be an object")
	}
	for name, version := range value.CapabilityLock {
		if !resourceNamePattern.MatchString(name) {
			return fmt.Errorf("capability_lock name %q is invalid", name)
		}
		if !semverPattern.MatchString(version) {
			return fmt.Errorf("capability_lock version %q for %q is not semantic versioning", version, name)
		}
	}
	for relativePath, file := range value.Files {
		if relativePath == Path || relativePath == ".scaffold-agent" || strings.HasPrefix(relativePath, ".scaffold-agent/") {
			return fmt.Errorf("file claim %q targets reserved Scaffold Agent metadata", relativePath)
		}
		if _, err := root.Resolve(relativePath); err != nil {
			return fmt.Errorf("file claim %q is invalid: %w", relativePath, err)
		}
		if !resourceNamePattern.MatchString(file.Owner) {
			return fmt.Errorf("file claim %q has invalid owner %q", relativePath, file.Owner)
		}
		if !isSHA256(file.Hash) {
			return fmt.Errorf("file claim %q has invalid SHA-256 hash", relativePath)
		}
	}
	return nil
}

// Encode produces deterministic, human-readable JSON for transaction staging.
func Encode(value Document, root projectfs.Root) ([]byte, error) {
	if err := Validate(value, root); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode ownership manifest: %w", err)
	}
	return append(content, '\n'), nil
}

// Clone returns a deep copy safe for building the next desired state.
func Clone(value Document) Document {
	cloned := Document{
		APIVersion:     value.APIVersion,
		BlueprintHash:  value.BlueprintHash,
		CapabilityLock: make(map[string]string, len(value.CapabilityLock)),
		Files:          make(map[string]File, len(value.Files)),
	}
	for name, version := range value.CapabilityLock {
		cloned.CapabilityLock[name] = version
	}
	for relativePath, file := range value.Files {
		cloned.Files[relativePath] = file
	}
	return cloned
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}

func requireEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON documents are not allowed")
	}
	return err
}

func rejectDuplicateObjectKeys(input []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	return requireEOF(decoder.Decode(new(any)))
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	_, err = decoder.Token()
	return err
}

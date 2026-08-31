// Package resultstore keeps large structured results out of an Agent's immediate context.
package resultstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/canonicaljson"
	"github.com/hkx5414375/scaffold-agent/internal/paging"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
	"github.com/hkx5414375/scaffold-agent/internal/projectmeta"
	"github.com/hkx5414375/scaffold-agent/internal/result"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	apiVersion   = "scaffold-agent.io/stored-result/v1alpha1"
	maxFileSize  = 32 << 20
	defaultLimit = 20
	maximumLimit = 100
)

var resultIDPattern = regexp.MustCompile(`^result_[a-f0-9]{64}$`)

// Record is a complete result before paging.
type Record struct {
	Status      result.Status
	Summary     string
	Diagnostics []spec.Diagnostic
	Metadata    map[string]any
	Items       []any
}

// PageData is the compact data returned inside a result envelope.
type PageData struct {
	Metadata map[string]any    `json:"metadata,omitempty"`
	Items    []json.RawMessage `json:"items,omitempty"`
}

type document struct {
	APIVersion  string            `json:"api_version"`
	ID          string            `json:"id"`
	Status      result.Status     `json:"status"`
	Summary     string            `json:"summary"`
	Diagnostics []spec.Diagnostic `json:"diagnostics,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Items       []json.RawMessage `json:"items,omitempty"`
}

type identity struct {
	APIVersion  string            `json:"api_version"`
	Status      result.Status     `json:"status"`
	Summary     string            `json:"summary"`
	Diagnostics []spec.Diagnostic `json:"diagnostics,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Items       []json.RawMessage `json:"items,omitempty"`
}

// Save stores a deterministic result and returns its content address.
func Save(rootPath string, record Record) (string, error) {
	if err := validateStatus(record.Status); err != nil {
		return "", err
	}
	if strings.TrimSpace(record.Summary) == "" {
		return "", errors.New("result summary is required")
	}
	items := make([]json.RawMessage, 0, len(record.Items))
	for index, item := range record.Items {
		content, err := json.Marshal(item)
		if err != nil {
			return "", fmt.Errorf("encode result item %d: %w", index, err)
		}
		items = append(items, content)
	}
	value := document{
		APIVersion:  apiVersion,
		Status:      record.Status,
		Summary:     record.Summary,
		Diagnostics: append([]spec.Diagnostic(nil), record.Diagnostics...),
		Metadata:    cloneMetadata(record.Metadata),
		Items:       items,
	}
	id, err := computeID(value)
	if err != nil {
		return "", err
	}
	value.ID = id
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode stored result: %w", err)
	}
	content = append(content, '\n')
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return "", err
	}
	target, err := resultPath(root, id)
	if err != nil {
		return "", err
	}
	if err := projectmeta.WriteImmutable(target, content); err != nil {
		return "", fmt.Errorf("save result %q: %w", id, err)
	}
	return id, nil
}

// Page loads one bounded result page using an opaque cursor.
func Page(rootPath, resultID, cursor string, limit int) (result.Envelope, error) {
	if !resultIDPattern.MatchString(resultID) {
		return result.Envelope{}, fmt.Errorf("invalid result ID %q", resultID)
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maximumLimit {
		return result.Envelope{}, fmt.Errorf("limit must be between 1 and %d", maximumLimit)
	}
	value, err := load(rootPath, resultID)
	if err != nil {
		return result.Envelope{}, err
	}
	offset, err := paging.Decode(cursor, resultID)
	if err != nil {
		return result.Envelope{}, fmt.Errorf("result %w", err)
	}
	start, end, err := paging.Bounds(len(value.Items), offset, limit, defaultLimit, maximumLimit)
	if err != nil {
		return result.Envelope{}, fmt.Errorf("result %w", err)
	}
	pageItems := append([]json.RawMessage(nil), value.Items[start:end]...)
	envelope := result.Envelope{
		APIVersion:  result.APIVersionV1Alpha1,
		Status:      value.Status,
		Summary:     value.Summary,
		ResultID:    value.ID,
		Diagnostics: append([]spec.Diagnostic(nil), value.Diagnostics...),
		Data:        PageData{Metadata: cloneMetadata(value.Metadata), Items: pageItems},
	}
	if end < len(value.Items) {
		envelope.HasMore = true
		envelope.NextCursor, err = paging.Encode(resultID, end)
		if err != nil {
			return result.Envelope{}, err
		}
	}
	envelope.EstimatedTokens = estimateTokens(envelope)
	return envelope, nil
}

func load(rootPath, resultID string) (document, error) {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return document{}, err
	}
	target, err := resultPath(root, resultID)
	if err != nil {
		return document{}, err
	}
	content, err := projectmeta.ReadRegularFile(target, maxFileSize)
	if err != nil {
		return document{}, fmt.Errorf("read result %q: %w", resultID, err)
	}
	var value document
	if err := projectmeta.DecodeStrict(content, &value); err != nil {
		return document{}, fmt.Errorf("decode result %q: %w", resultID, err)
	}
	if value.APIVersion != apiVersion {
		return document{}, fmt.Errorf("result %q has unsupported api_version %q", resultID, value.APIVersion)
	}
	if value.ID != resultID {
		return document{}, fmt.Errorf("result file ID %q does not match stored ID %q", resultID, value.ID)
	}
	if err := validateStatus(value.Status); err != nil {
		return document{}, err
	}
	computedID, err := computeID(value)
	if err != nil {
		return document{}, err
	}
	if computedID != resultID {
		return document{}, fmt.Errorf("stored result hash mismatch: got %q, computed %q", resultID, computedID)
	}
	return value, nil
}

func computeID(value document) (string, error) {
	hash, err := canonicaljson.Hash(identity{
		APIVersion:  value.APIVersion,
		Status:      value.Status,
		Summary:     value.Summary,
		Diagnostics: value.Diagnostics,
		Metadata:    value.Metadata,
		Items:       value.Items,
	})
	if err != nil {
		return "", fmt.Errorf("hash stored result: %w", err)
	}
	return "result_" + hash, nil
}

func resultPath(root projectfs.Root, resultID string) (string, error) {
	return root.Resolve(".scaffold-agent/results/" + resultID + ".json")
}

func validateStatus(value result.Status) error {
	switch value {
	case result.StatusOK, result.StatusWarning, result.StatusError:
		return nil
	default:
		return fmt.Errorf("invalid result status %q", value)
	}
}

func estimateTokens(value result.Envelope) int {
	content, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return (len(content) + 3) / 4
}

func cloneMetadata(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

// Package change compiles desired managed outputs into immutable change plans.
package change

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/canonicaljson"
	"github.com/hkx5414375/scaffold-agent/internal/manifest"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
)

// Output is one desired managed file or deletion.
type Output struct {
	Path    string
	Owner   string
	Content []byte
	Delete  bool
}

// Artifact keeps generated content separate from the compact public plan.
type Artifact struct {
	Plan    plan.Plan
	Content map[string][]byte
}

type fileState struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Hash   string `json:"hash,omitempty"`
}

// Build creates a deterministic plan without changing the filesystem.
func Build(rootPath string, action plan.Action, blueprintHash string, capabilityLock map[string]string, outputs []Output) (Artifact, error) {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return Artifact{}, err
	}
	currentManifest, err := manifest.Load(root)
	if err != nil {
		return Artifact{}, err
	}
	nextManifest := manifest.Clone(currentManifest.Document)
	nextManifest.BlueprintHash = blueprintHash
	nextManifest.CapabilityLock = cloneLock(capabilityLock)
	sortedOutputs := append([]Output(nil), outputs...)
	sort.Slice(sortedOutputs, func(i, j int) bool { return sortedOutputs[i].Path < sortedOutputs[j].Path })
	seen := make(map[string]struct{}, len(sortedOutputs))
	content := make(map[string][]byte)
	var changes []plan.Change
	var states []fileState
	for _, output := range sortedOutputs {
		if _, exists := seen[output.Path]; exists {
			return Artifact{}, fmt.Errorf("duplicate desired output path %q", output.Path)
		}
		seen[output.Path] = struct{}{}
		if output.Path == ".scaffold-agent" || strings.HasPrefix(output.Path, ".scaffold-agent/") {
			return Artifact{}, fmt.Errorf("output %q targets reserved Scaffold Agent metadata", output.Path)
		}
		if output.Owner == "" {
			return Artifact{}, fmt.Errorf("output %q has no owner", output.Path)
		}
		target, err := root.Resolve(output.Path)
		if err != nil {
			return Artifact{}, err
		}
		beforeHash, exists, err := currentFileHash(target)
		if err != nil {
			return Artifact{}, fmt.Errorf("inspect output %q: %w", output.Path, err)
		}
		states = append(states, fileState{Path: output.Path, Exists: exists, Hash: beforeHash})
		managedFile, managed := currentManifest.Document.Files[output.Path]
		if output.Delete {
			if managed && managedFile.Owner != output.Owner {
				return Artifact{}, ownerMismatchError(output.Path, output.Owner, managedFile.Owner)
			}
			if exists {
				if !managed {
					return Artifact{}, fmt.Errorf("refuse to delete unmanaged file %q", output.Path)
				}
				if managedFile.Hash != beforeHash {
					return Artifact{}, modifiedManagedFileError(output.Path, managedFile.Hash, beforeHash)
				}
				changes = append(changes, plan.Change{Operation: plan.OperationDelete, Path: output.Path, Owner: output.Owner, BeforeHash: beforeHash})
			}
			delete(nextManifest.Files, output.Path)
			continue
		}
		afterHash := projectfs.HashBytes(output.Content)
		if exists {
			if !managed {
				return Artifact{}, fmt.Errorf("refuse to overwrite unmanaged file %q", output.Path)
			}
			if managedFile.Owner != output.Owner {
				return Artifact{}, ownerMismatchError(output.Path, output.Owner, managedFile.Owner)
			}
			if managedFile.Hash != beforeHash {
				return Artifact{}, modifiedManagedFileError(output.Path, managedFile.Hash, beforeHash)
			}
		} else if managed {
			if managedFile.Owner != output.Owner {
				return Artifact{}, ownerMismatchError(output.Path, output.Owner, managedFile.Owner)
			}
			if action != plan.ActionRepair {
				return Artifact{}, fmt.Errorf("managed file %q is missing; rebuild with action %q to recreate it", output.Path, plan.ActionRepair)
			}
		}
		nextManifest.Files[output.Path] = manifest.File{Owner: output.Owner, Hash: afterHash}
		if exists && beforeHash == afterHash {
			continue
		}
		operation := plan.OperationCreate
		if exists {
			operation = plan.OperationModify
		}
		changes = append(changes, plan.Change{Operation: operation, Path: output.Path, Owner: output.Owner, BeforeHash: beforeHash, AfterHash: afterHash})
		content[output.Path] = append([]byte(nil), output.Content...)
	}
	manifestContent, err := manifest.Encode(nextManifest, root)
	if err != nil {
		return Artifact{}, fmt.Errorf("build ownership manifest: %w", err)
	}
	states = append(states, fileState{Path: manifest.Path, Exists: currentManifest.Exists, Hash: currentManifest.Hash})
	manifestAfterHash := projectfs.HashBytes(manifestContent)
	if !currentManifest.Exists || currentManifest.Hash != manifestAfterHash || len(changes) > 0 {
		operation := plan.OperationCreate
		if currentManifest.Exists {
			operation = plan.OperationModify
		}
		changes = append(changes, plan.Change{
			Operation:  operation,
			Path:       manifest.Path,
			Owner:      manifest.Owner,
			BeforeHash: currentManifest.Hash,
			AfterHash:  manifestAfterHash,
		})
		content[manifest.Path] = manifestContent
	}
	projectHash, err := canonicaljson.Hash(states)
	if err != nil {
		return Artifact{}, fmt.Errorf("hash project preconditions: %w", err)
	}
	value := plan.Plan{
		APIVersion:     plan.APIVersionV1Alpha1,
		Action:         action,
		ProjectRoot:    root.Path(),
		BlueprintHash:  blueprintHash,
		ProjectHash:    projectHash,
		CapabilityLock: cloneLock(capabilityLock),
		Changes:        changes,
	}
	value, err = plan.WithComputedID(value)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Plan: value, Content: content}, nil
}

func currentFileHash(filePath string) (string, bool, error) {
	hash, err := projectfs.HashFile(filePath)
	if err == nil {
		return hash, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	return "", false, err
}

func cloneLock(lock map[string]string) map[string]string {
	cloned := make(map[string]string, len(lock))
	for name, version := range lock {
		cloned[name] = version
	}
	return cloned
}

func ownerMismatchError(relativePath, desiredOwner, currentOwner string) error {
	return fmt.Errorf("refuse ownership change for %q: requested owner %q, manifest owner %q", relativePath, desiredOwner, currentOwner)
}

func modifiedManagedFileError(relativePath, expectedHash, currentHash string) error {
	return fmt.Errorf("refuse to overwrite managed file %q because it was changed outside Scaffold Agent: manifest hash %q, current hash %q", relativePath, expectedHash, currentHash)
}

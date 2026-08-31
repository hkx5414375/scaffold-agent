// Package artifactstore persists immutable change artifacts between Agent calls.
package artifactstore

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
	"github.com/hkx5414375/scaffold-agent/internal/projectmeta"
)

const (
	apiVersion  = "scaffold-agent.io/artifact/v1alpha1"
	maxFileSize = 64 << 20
)

var planIDPattern = regexp.MustCompile(`^plan_[a-f0-9]{64}$`)

type document struct {
	APIVersion string            `json:"api_version"`
	Plan       json.RawMessage   `json:"plan"`
	Content    map[string][]byte `json:"content"`
}

// Save atomically stores an artifact under its content-addressed Plan ID.
func Save(rootPath string, artifact change.Artifact) error {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return err
	}
	artifact.Plan.ProjectRoot = root.Path()
	if err := change.ValidateArtifact(artifact); err != nil {
		return fmt.Errorf("validate artifact: %w", err)
	}
	if !planIDPattern.MatchString(artifact.Plan.ID) {
		return fmt.Errorf("invalid plan ID %q", artifact.Plan.ID)
	}
	planContent, err := json.Marshal(artifact.Plan)
	if err != nil {
		return fmt.Errorf("encode artifact plan: %w", err)
	}
	content, err := json.MarshalIndent(document{
		APIVersion: apiVersion,
		Plan:       planContent,
		Content:    artifact.Content,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode artifact: %w", err)
	}
	content = append(content, '\n')
	target, err := artifactPath(root, artifact.Plan.ID)
	if err != nil {
		return err
	}
	if err := projectmeta.WriteImmutable(target, content); err != nil {
		return fmt.Errorf("save artifact %q: %w", artifact.Plan.ID, err)
	}
	return nil
}

// Load reads an artifact, rebinds it to the requested project root, and verifies all hashes.
func Load(rootPath, planID string) (change.Artifact, error) {
	if !planIDPattern.MatchString(planID) {
		return change.Artifact{}, fmt.Errorf("invalid plan ID %q", planID)
	}
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return change.Artifact{}, err
	}
	target, err := artifactPath(root, planID)
	if err != nil {
		return change.Artifact{}, err
	}
	content, err := projectmeta.ReadRegularFile(target, maxFileSize)
	if err != nil {
		return change.Artifact{}, fmt.Errorf("read artifact %q: %w", planID, err)
	}
	var stored document
	if err := projectmeta.DecodeStrict(content, &stored); err != nil {
		return change.Artifact{}, fmt.Errorf("decode artifact %q: %w", planID, err)
	}
	if stored.APIVersion != apiVersion {
		return change.Artifact{}, fmt.Errorf("artifact %q has unsupported api_version %q", planID, stored.APIVersion)
	}
	var artifact change.Artifact
	if err := projectmeta.DecodeStrict(stored.Plan, &artifact.Plan); err != nil {
		return change.Artifact{}, fmt.Errorf("decode artifact plan %q: %w", planID, err)
	}
	artifact.Plan.ProjectRoot = root.Path()
	artifact.Content = stored.Content
	if artifact.Plan.ID != planID {
		return change.Artifact{}, fmt.Errorf("artifact file ID %q does not match plan ID %q", planID, artifact.Plan.ID)
	}
	if err := change.ValidateArtifact(artifact); err != nil {
		return change.Artifact{}, fmt.Errorf("validate artifact %q: %w", planID, err)
	}
	return artifact, nil
}

func artifactPath(root projectfs.Root, planID string) (string, error) {
	return root.Resolve(".scaffold-agent/plans/" + planID + ".json")
}

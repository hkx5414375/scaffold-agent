package change

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
)

// Receipt summarizes an applied or rolled-back transaction.
type Receipt struct {
	PlanID         string `json:"plan_id"`
	ChangedFiles   int    `json:"changed_files"`
	AlreadyApplied bool   `json:"already_applied,omitempty"`
	RolledBack     bool   `json:"rolled_back,omitempty"`
}

// Apply verifies every precondition and applies an artifact transactionally.
func Apply(artifact Artifact) (Receipt, error) {
	computedID, err := plan.ComputeID(artifact.Plan)
	if err != nil {
		return Receipt{}, err
	}
	if computedID != artifact.Plan.ID {
		return Receipt{}, fmt.Errorf("plan ID mismatch: got %q, computed %q", artifact.Plan.ID, computedID)
	}
	root, err := projectfs.Open(artifact.Plan.ProjectRoot)
	if err != nil {
		return Receipt{}, err
	}
	if err := os.MkdirAll(root.Path(), 0o755); err != nil {
		return Receipt{}, fmt.Errorf("create project root: %w", err)
	}
	backupRoot, err := root.Resolve(".scaffold-agent/backups/" + artifact.Plan.ID)
	if err != nil {
		return Receipt{}, err
	}
	if value, _, journalErr := loadLatestJournal(backupRoot); journalErr == nil {
		if value.Status == journalApplied {
			if verifyErr := verifyPostconditions(root, artifact.Plan.Changes); verifyErr != nil {
				return Receipt{}, fmt.Errorf("plan was applied but postconditions changed: %w", verifyErr)
			}
			return Receipt{PlanID: artifact.Plan.ID, ChangedFiles: len(artifact.Plan.Changes), AlreadyApplied: true}, nil
		}
		return Receipt{}, fmt.Errorf("plan %q already has transaction status %q; recover or rebuild the plan", artifact.Plan.ID, value.Status)
	} else if !errors.Is(journalErr, os.ErrNotExist) {
		return Receipt{}, fmt.Errorf("inspect existing transaction: %w", journalErr)
	}
	if len(artifact.Plan.Changes) == 0 {
		return Receipt{PlanID: artifact.Plan.ID}, nil
	}
	if err := verifyArtifact(artifact); err != nil {
		return Receipt{}, err
	}
	if err := verifyPreconditions(root, artifact.Plan.Changes); err != nil {
		return Receipt{}, err
	}
	stageRoot, err := root.Resolve(".scaffold-agent/tmp/" + artifact.Plan.ID)
	if err != nil {
		return Receipt{}, err
	}
	if _, err := os.Lstat(stageRoot); err == nil {
		return Receipt{}, fmt.Errorf("staging directory already exists for plan %q", artifact.Plan.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Receipt{}, fmt.Errorf("inspect staging directory: %w", err)
	}
	if err := stageArtifact(root, stageRoot, artifact); err != nil {
		return Receipt{}, err
	}
	defer os.RemoveAll(stageRoot)

	value := journal{Version: journalVersion, PlanID: artifact.Plan.ID, Status: journalPrepared, Entries: journalEntries(artifact.Plan.Changes)}
	if err := writeJournalSnapshot(backupRoot, 0, value); err != nil {
		return Receipt{}, err
	}
	value.Status = journalApplying
	if err := writeJournalSnapshot(backupRoot, 1, value); err != nil {
		return Receipt{}, err
	}
	for index, change := range artifact.Plan.Changes {
		if err := applyChange(root, stageRoot, backupRoot, change); err != nil {
			rollbackErr := rollbackApplied(root, backupRoot, value.Entries[:value.AppliedCount])
			if rollbackErr != nil {
				return Receipt{}, fmt.Errorf("apply %q: %v; automatic rollback also failed: %w", change.Path, err, rollbackErr)
			}
			return Receipt{}, fmt.Errorf("apply %q: %w", change.Path, err)
		}
		value.AppliedCount = index + 1
		if err := writeJournalSnapshot(backupRoot, index+2, value); err != nil {
			rollbackErr := rollbackApplied(root, backupRoot, value.Entries[:value.AppliedCount])
			if rollbackErr != nil {
				return Receipt{}, fmt.Errorf("record applied change: %v; automatic rollback also failed: %w", err, rollbackErr)
			}
			return Receipt{}, fmt.Errorf("record applied change: %w", err)
		}
	}
	value.Status = journalApplied
	if err := writeJournalSnapshot(backupRoot, len(artifact.Plan.Changes)+2, value); err != nil {
		return Receipt{}, fmt.Errorf("record applied transaction: %w", err)
	}
	return Receipt{PlanID: artifact.Plan.ID, ChangedFiles: len(artifact.Plan.Changes)}, nil
}

// Rollback restores files from one successfully applied transaction.
func Rollback(rootPath, planID string) (Receipt, error) {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return Receipt{}, err
	}
	backupRoot, err := root.Resolve(".scaffold-agent/backups/" + planID)
	if err != nil {
		return Receipt{}, err
	}
	value, sequence, err := loadLatestJournal(backupRoot)
	if err != nil {
		return Receipt{}, fmt.Errorf("load rollback journal: %w", err)
	}
	if value.PlanID != planID {
		return Receipt{}, fmt.Errorf("journal plan ID %q does not match %q", value.PlanID, planID)
	}
	if value.Version != journalVersion {
		return Receipt{}, fmt.Errorf("unsupported journal version %d", value.Version)
	}
	if value.Status == journalRolledBack {
		return Receipt{PlanID: planID, ChangedFiles: value.AppliedCount, RolledBack: true}, nil
	}
	if value.Status != journalApplied {
		return Receipt{}, fmt.Errorf("transaction status %q is not safe for explicit rollback", value.Status)
	}
	changes := make([]plan.Change, 0, len(value.Entries))
	for _, entry := range value.Entries {
		changes = append(changes, plan.Change{
			Operation:  plan.Operation(entry.Operation),
			Path:       entry.Path,
			BeforeHash: entry.BeforeHash,
			AfterHash:  entry.AfterHash,
		})
	}
	if err := verifyPostconditions(root, changes); err != nil {
		return Receipt{}, fmt.Errorf("refuse rollback because applied files changed: %w", err)
	}
	if err := rollbackApplied(root, backupRoot, value.Entries); err != nil {
		return Receipt{}, err
	}
	value.Status = journalRolledBack
	if err := writeJournalSnapshot(backupRoot, sequence+1, value); err != nil {
		return Receipt{}, err
	}
	return Receipt{PlanID: planID, ChangedFiles: len(value.Entries), RolledBack: true}, nil
}

// Recover safely restores a transaction interrupted before it reached applied status.
func Recover(rootPath, planID string) (Receipt, error) {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return Receipt{}, err
	}
	backupRoot, err := root.Resolve(".scaffold-agent/backups/" + planID)
	if err != nil {
		return Receipt{}, err
	}
	value, sequence, err := loadLatestJournal(backupRoot)
	if err != nil {
		return Receipt{}, fmt.Errorf("load recovery journal: %w", err)
	}
	if value.PlanID != planID || value.Version != journalVersion {
		return Receipt{}, errors.New("recovery journal identity or version is invalid")
	}
	if value.Status == journalApplied {
		return Receipt{}, errors.New("transaction is fully applied; use rollback instead of recovery")
	}
	if value.Status == journalRolledBack {
		return Receipt{PlanID: planID, ChangedFiles: value.AppliedCount, RolledBack: true}, nil
	}
	for index := len(value.Entries) - 1; index >= 0; index-- {
		if err := recoverEntry(root, backupRoot, value.Entries[index]); err != nil {
			return Receipt{}, fmt.Errorf("recover %q: %w", value.Entries[index].Path, err)
		}
	}
	stageRoot, err := root.Resolve(".scaffold-agent/tmp/" + planID)
	if err != nil {
		return Receipt{}, err
	}
	if err := os.RemoveAll(stageRoot); err != nil {
		return Receipt{}, fmt.Errorf("remove recovered staging directory: %w", err)
	}
	value.Status = journalRolledBack
	if err := writeJournalSnapshot(backupRoot, sequence+1, value); err != nil {
		return Receipt{}, err
	}
	return Receipt{PlanID: planID, ChangedFiles: len(value.Entries), RolledBack: true}, nil
}

func verifyArtifact(artifact Artifact) error {
	expectedContent := make(map[string]struct{})
	for _, change := range artifact.Plan.Changes {
		switch change.Operation {
		case plan.OperationCreate, plan.OperationModify:
			content, exists := artifact.Content[change.Path]
			if !exists {
				return fmt.Errorf("artifact content missing for %q", change.Path)
			}
			if hash := projectfs.HashBytes(content); hash != change.AfterHash {
				return fmt.Errorf("artifact content hash mismatch for %q", change.Path)
			}
			expectedContent[change.Path] = struct{}{}
		case plan.OperationDelete:
			if _, exists := artifact.Content[change.Path]; exists {
				return fmt.Errorf("delete change %q unexpectedly contains content", change.Path)
			}
		default:
			return fmt.Errorf("unsupported change operation %q", change.Operation)
		}
	}
	for path := range artifact.Content {
		if _, exists := expectedContent[path]; !exists {
			return fmt.Errorf("artifact contains undeclared content for %q", path)
		}
	}
	return nil
}

func verifyPreconditions(root projectfs.Root, changes []plan.Change) error {
	for _, change := range changes {
		target, err := root.Resolve(change.Path)
		if err != nil {
			return err
		}
		currentHash, exists, err := currentFileHash(target)
		if err != nil {
			return err
		}
		if change.Operation == plan.OperationCreate {
			if exists {
				return fmt.Errorf("precondition failed for %q: file now exists", change.Path)
			}
			continue
		}
		if !exists || currentHash != change.BeforeHash {
			return fmt.Errorf("precondition failed for %q: expected hash %q, got %q", change.Path, change.BeforeHash, currentHash)
		}
	}
	return nil
}

func verifyPostconditions(root projectfs.Root, changes []plan.Change) error {
	for _, change := range changes {
		target, err := root.Resolve(change.Path)
		if err != nil {
			return err
		}
		currentHash, exists, err := currentFileHash(target)
		if err != nil {
			return err
		}
		if change.Operation == plan.OperationDelete {
			if exists {
				return fmt.Errorf("postcondition failed for %q: deleted file exists", change.Path)
			}
			continue
		}
		if !exists || currentHash != change.AfterHash {
			return fmt.Errorf("postcondition failed for %q: expected hash %q, got %q", change.Path, change.AfterHash, currentHash)
		}
	}
	return nil
}

func stageArtifact(root projectfs.Root, stageRoot string, artifact Artifact) error {
	for path, content := range artifact.Content {
		if _, err := root.Resolve(path); err != nil {
			return err
		}
		target := filepath.Join(stageRoot, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("create staging directory for %q: %w", path, err)
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("create staged file %q: %w", path, err)
		}
		if _, err := file.Write(content); err != nil {
			file.Close()
			return fmt.Errorf("write staged file %q: %w", path, err)
		}
		if err := file.Sync(); err != nil {
			file.Close()
			return fmt.Errorf("sync staged file %q: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close staged file %q: %w", path, err)
		}
	}
	return nil
}

func journalEntries(changes []plan.Change) []journalEntry {
	entries := make([]journalEntry, 0, len(changes))
	for _, change := range changes {
		entry := journalEntry{
			Path:       change.Path,
			Operation:  string(change.Operation),
			BeforeHash: change.BeforeHash,
			AfterHash:  change.AfterHash,
		}
		if change.Operation != plan.OperationCreate {
			entry.BackupPath = filepath.ToSlash(filepath.Join("files", filepath.FromSlash(change.Path)))
		}
		entries = append(entries, entry)
	}
	return entries
}

func applyChange(root projectfs.Root, stageRoot, backupRoot string, change plan.Change) error {
	target, err := root.Resolve(change.Path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	staged := filepath.Join(stageRoot, filepath.FromSlash(change.Path))
	backup := filepath.Join(backupRoot, "files", filepath.FromSlash(change.Path))
	switch change.Operation {
	case plan.OperationCreate:
		if _, err := os.Lstat(target); err == nil {
			return errors.New("target appeared after precondition validation")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return os.Rename(staged, target)
	case plan.OperationModify:
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return err
		}
		if err := os.Rename(target, backup); err != nil {
			return err
		}
		if err := os.Rename(staged, target); err != nil {
			_ = os.Rename(backup, target)
			return err
		}
		return nil
	case plan.OperationDelete:
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return err
		}
		return os.Rename(target, backup)
	default:
		return fmt.Errorf("unsupported operation %q", change.Operation)
	}
}

func rollbackApplied(root projectfs.Root, backupRoot string, entries []journalEntry) error {
	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		target, err := root.Resolve(entry.Path)
		if err != nil {
			return err
		}
		switch plan.Operation(entry.Operation) {
		case plan.OperationCreate:
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove created file %q: %w", entry.Path, err)
			}
			removeEmptyParents(root.Path(), filepath.Dir(target))
		case plan.OperationModify:
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove modified file %q: %w", entry.Path, err)
			}
			if err := restoreBackup(backupRoot, target, entry); err != nil {
				return err
			}
		case plan.OperationDelete:
			if err := restoreBackup(backupRoot, target, entry); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported rollback operation %q", entry.Operation)
		}
	}
	return nil
}

func removeEmptyParents(rootPath, directory string) {
	for directory != rootPath && projectfsContains(rootPath, directory) {
		if err := os.Remove(directory); err != nil {
			return
		}
		directory = filepath.Dir(directory)
	}
}

func projectfsContains(rootPath, targetPath string) bool {
	relative, err := filepath.Rel(rootPath, targetPath)
	return err == nil && relative != ".." && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func restoreBackup(backupRoot, target string, entry journalEntry) error {
	backup, err := backupTarget(backupRoot, entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Rename(backup, target); err != nil {
		return fmt.Errorf("restore backup for %q: %w", entry.Path, err)
	}
	return nil
}

func backupTarget(backupRoot string, entry journalEntry) (string, error) {
	expectedRelative := filepath.ToSlash(filepath.Join("files", filepath.FromSlash(entry.Path)))
	if entry.BackupPath != expectedRelative {
		return "", fmt.Errorf("journal backup path for %q is invalid", entry.Path)
	}
	return filepath.Join(backupRoot, filepath.FromSlash(expectedRelative)), nil
}

func recoverEntry(root projectfs.Root, backupRoot string, entry journalEntry) error {
	target, err := root.Resolve(entry.Path)
	if err != nil {
		return err
	}
	targetHash, targetExists, err := currentFileHash(target)
	if err != nil {
		return err
	}
	switch plan.Operation(entry.Operation) {
	case plan.OperationCreate:
		if !targetExists {
			return nil
		}
		if targetHash != entry.AfterHash {
			return errors.New("created target contains an unexpected user change")
		}
		if err := os.Remove(target); err != nil {
			return err
		}
		removeEmptyParents(root.Path(), filepath.Dir(target))
		return nil
	case plan.OperationModify, plan.OperationDelete:
		backup, err := backupTarget(backupRoot, entry)
		if err != nil {
			return err
		}
		backupHash, backupExists, err := currentFileHash(backup)
		if err != nil {
			return err
		}
		if !backupExists {
			if targetExists && targetHash == entry.BeforeHash {
				return nil
			}
			return errors.New("original file is unavailable and target does not match the precondition")
		}
		if backupHash != entry.BeforeHash {
			return errors.New("backup hash does not match the precondition")
		}
		if targetExists {
			if plan.Operation(entry.Operation) != plan.OperationModify || targetHash != entry.AfterHash {
				return errors.New("target contains an unexpected user change")
			}
			if err := os.Remove(target); err != nil {
				return err
			}
		}
		return restoreBackup(backupRoot, target, entry)
	default:
		return fmt.Errorf("unsupported recovery operation %q", entry.Operation)
	}
}

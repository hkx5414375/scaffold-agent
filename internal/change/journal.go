package change

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const journalVersion = 1

type journalStatus string

const (
	journalPrepared   journalStatus = "prepared"
	journalApplying   journalStatus = "applying"
	journalApplied    journalStatus = "applied"
	journalRolledBack journalStatus = "rolled_back"
)

type journal struct {
	Version      int            `json:"version"`
	PlanID       string         `json:"plan_id"`
	Status       journalStatus  `json:"status"`
	AppliedCount int            `json:"applied_count"`
	Entries      []journalEntry `json:"entries"`
}

type journalEntry struct {
	Path       string `json:"path"`
	Operation  string `json:"operation"`
	BeforeHash string `json:"before_hash,omitempty"`
	AfterHash  string `json:"after_hash,omitempty"`
	BackupPath string `json:"backup_path,omitempty"`
}

func writeJournalSnapshot(backupRoot string, sequence int, value journal) error {
	if err := os.MkdirAll(backupRoot, 0o700); err != nil {
		return fmt.Errorf("create backup root: %w", err)
	}
	target := filepath.Join(backupRoot, fmt.Sprintf("journal-%06d.json", sequence))
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create journal snapshot: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encodeErr := encoder.Encode(value)
	syncErr := file.Sync()
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("encode journal snapshot: %w", encodeErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync journal snapshot: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close journal snapshot: %w", closeErr)
	}
	return nil
}

func loadLatestJournal(backupRoot string) (journal, int, error) {
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return journal{}, 0, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) == len("journal-000000.json") && entry.Name()[:8] == "journal-" {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		content, readErr := os.ReadFile(filepath.Join(backupRoot, name))
		if readErr != nil {
			continue
		}
		var value journal
		if decodeErr := json.Unmarshal(content, &value); decodeErr != nil {
			continue
		}
		var sequence int
		if _, scanErr := fmt.Sscanf(name, "journal-%06d.json", &sequence); scanErr != nil {
			continue
		}
		return value, sequence, nil
	}
	return journal{}, 0, errors.New("no valid journal snapshot found")
}

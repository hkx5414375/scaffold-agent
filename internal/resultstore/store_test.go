package resultstore

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/result"
)

func TestSaveAndPageResult(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	id, err := Save(root, Record{
		Status:   result.StatusOK,
		Summary:  "three findings",
		Metadata: map[string]any{"kind": "test"},
		Items:    []any{map[string]any{"index": 1}, map[string]any{"index": 2}, map[string]any{"index": 3}},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	first, err := Page(root, id, "", 2)
	if err != nil {
		t.Fatalf("Page(first) error = %v", err)
	}
	if !first.HasMore || first.NextCursor == "" {
		t.Fatalf("Page(first) pagination = has_more:%v cursor:%q", first.HasMore, first.NextCursor)
	}
	firstData, ok := first.Data.(PageData)
	if !ok || len(firstData.Items) != 2 {
		t.Fatalf("Page(first) data = %#v, want two items", first.Data)
	}
	second, err := Page(root, id, first.NextCursor, 2)
	if err != nil {
		t.Fatalf("Page(second) error = %v", err)
	}
	secondData := second.Data.(PageData)
	if second.HasMore || len(secondData.Items) != 1 {
		t.Fatalf("Page(second) = has_more:%v items:%d", second.HasMore, len(secondData.Items))
	}
	var item map[string]int
	if err := json.Unmarshal(secondData.Items[0], &item); err != nil || item["index"] != 3 {
		t.Fatalf("Page(second) item = %s, error = %v", secondData.Items[0], err)
	}
}

func TestSaveIsContentAddressedAndIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	record := Record{Status: result.StatusOK, Summary: "same", Items: []any{"value"}}
	first, err := Save(root, record)
	if err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}
	second, err := Save(root, record)
	if err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}
	if first != second {
		t.Fatalf("result IDs differ: %q != %q", first, second)
	}
}

func TestPageRejectsCursorForAnotherResult(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	firstID, err := Save(root, Record{Status: result.StatusOK, Summary: "first", Items: []any{1, 2}})
	if err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}
	secondID, err := Save(root, Record{Status: result.StatusOK, Summary: "second", Items: []any{1, 2}})
	if err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}
	firstPage, err := Page(root, firstID, "", 1)
	if err != nil {
		t.Fatalf("Page(first) error = %v", err)
	}
	if _, err := Page(root, secondID, firstPage.NextCursor, 1); err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("Page(mismatched cursor) error = %v, want ownership error", err)
	}
}

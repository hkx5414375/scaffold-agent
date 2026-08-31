package canonicaljson

import (
	"errors"
	"testing"
)

func TestMarshalSortsObjectKeys(t *testing.T) {
	t.Parallel()

	value := map[string]any{"z": 2, "a": map[string]any{"b": true, "a": "first"}}
	got, err := Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"a":{"a":"first","b":true},"z":2}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestHashIgnoresMapInsertionOrder(t *testing.T) {
	t.Parallel()

	first, err := Hash(map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("Hash(first) error = %v", err)
	}
	second, err := Hash(map[string]any{"b": 2, "a": 1})
	if err != nil {
		t.Fatalf("Hash(second) error = %v", err)
	}
	if first != second {
		t.Fatalf("hashes differ: %s != %s", first, second)
	}
}

func TestMarshalRejectsFloatNumbers(t *testing.T) {
	t.Parallel()

	_, err := Marshal(map[string]any{"amount": 1.25})
	if !errors.Is(err, errNonIntegerNumber) {
		t.Fatalf("Marshal() error = %v, want errNonIntegerNumber", err)
	}
}

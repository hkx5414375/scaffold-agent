package paging

import "testing"

func TestCursorRoundTripAndSubjectBinding(t *testing.T) {
	t.Parallel()

	value, err := Encode("result_one", 12)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	offset, err := Decode(value, "result_one")
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if offset != 12 {
		t.Fatalf("Decode() offset = %d, want 12", offset)
	}
	if _, err := Decode(value, "result_two"); err == nil {
		t.Fatal("Decode(other subject) error = nil, want error")
	}
}

func TestBoundsAppliesLimits(t *testing.T) {
	t.Parallel()

	start, end, err := Bounds(25, 10, 0, 20, 100)
	if err != nil {
		t.Fatalf("Bounds() error = %v", err)
	}
	if start != 10 || end != 25 {
		t.Fatalf("Bounds() = %d:%d, want 10:25", start, end)
	}
	if _, _, err := Bounds(25, 0, 101, 20, 100); err == nil {
		t.Fatal("Bounds(too large) error = nil, want error")
	}
}

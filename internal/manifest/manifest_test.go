package manifest

import (
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
)

func TestEncodeDecodeRoundTripIsDeterministic(t *testing.T) {
	t.Parallel()

	root, err := projectfs.Open(t.TempDir())
	if err != nil {
		t.Fatalf("projectfs.Open() error = %v", err)
	}
	value := Empty()
	value.BlueprintHash = strings.Repeat("c", 64)
	value.CapabilityLock["rbac"] = "1.0.0"
	value.CapabilityLock["auth"] = "1.0.0"
	value.Files["z.txt"] = File{Owner: "rbac", Hash: strings.Repeat("b", 64)}
	value.Files["a.txt"] = File{Owner: "auth", Hash: strings.Repeat("a", 64)}

	first, err := Encode(value, root)
	if err != nil {
		t.Fatalf("Encode(first) error = %v", err)
	}
	decoded, err := Decode(first, root)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	second, err := Encode(decoded, root)
	if err != nil {
		t.Fatalf("Encode(second) error = %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("manifest encoding is not deterministic:\n%s\n%s", first, second)
	}
}

func TestDecodeRejectsUnknownAndDuplicateFields(t *testing.T) {
	t.Parallel()

	root, err := projectfs.Open(t.TempDir())
	if err != nil {
		t.Fatalf("projectfs.Open() error = %v", err)
	}
	unknown := `{"api_version":"scaffold-agent.io/manifest/v1alpha1","capability_lock":{},"files":{},"extra":true}`
	if _, err := Decode([]byte(unknown), root); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Decode(unknown) error = %v, want unknown field error", err)
	}
	duplicate := `{"api_version":"scaffold-agent.io/manifest/v1alpha1","api_version":"duplicate","capability_lock":{},"files":{}}`
	if _, err := Decode([]byte(duplicate), root); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("Decode(duplicate) error = %v, want duplicate key error", err)
	}
}

func TestValidateRejectsReservedOrEscapingClaims(t *testing.T) {
	t.Parallel()

	root, err := projectfs.Open(t.TempDir())
	if err != nil {
		t.Fatalf("projectfs.Open() error = %v", err)
	}
	for _, relativePath := range []string{".scaffold-agent/other.json", "../outside"} {
		value := Empty()
		value.Files[relativePath] = File{Owner: "test", Hash: strings.Repeat("a", 64)}
		if err := Validate(value, root); err == nil {
			t.Fatalf("Validate(%q) error = nil, want error", relativePath)
		}
	}
}

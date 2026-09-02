package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/manifest"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

func TestV1ReleaseSchemaSnapshots(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"capability-pack.schema.json": "f2fbe874b6dc94fd65887cdc20e4e5ea494400b22a012ac894f7d755605aaf7d",
		"manifest.schema.json":        "ec6e1aa9b9f3262438b51163d563d61cd29f14ba021e6a5e91320c5a50519f89",
		"plan.schema.json":            "6260c54864b549ee02853a1e193023a26419d3e8e826f0d8dcdcfb27b0de3821",
		"project.schema.json":         "0dcbde21075782cb1421b4ab4e53df154f70ed5b2e2e0bfe8adfbf4dd48e2aef",
		"result.schema.json":          "5484dd89346e59c8e24b5a5c597bc37bdb8c401a3a81c7cd48b20d3c6c17a55b",
	}
	for name, wantHash := range want {
		content, err := Read("v1alpha1", name)
		if err != nil {
			t.Fatalf("Read(%q) error = %v", name, err)
		}
		hash := sha256.Sum256(content)
		if got := hex.EncodeToString(hash[:]); got != wantHash {
			t.Errorf("%s hash = %s, frozen v1.0 hash = %s", name, got, wantHash)
		}
	}
}

func TestV1ReleaseWireIdentifiers(t *testing.T) {
	t.Parallel()

	if spec.APIVersionV1Alpha1 != "scaffold-agent.io/v1alpha1" {
		t.Errorf("Blueprint API version = %q", spec.APIVersionV1Alpha1)
	}
	if plan.APIVersionV1Alpha1 != "scaffold-agent.io/plan/v1alpha1" {
		t.Errorf("Plan API version = %q", plan.APIVersionV1Alpha1)
	}
	if result.APIVersionV1Alpha1 != "scaffold-agent.io/result/v1alpha1" {
		t.Errorf("Result API version = %q", result.APIVersionV1Alpha1)
	}
	if manifest.APIVersionV1Alpha1 != "scaffold-agent.io/manifest/v1alpha1" {
		t.Errorf("Manifest API version = %q", manifest.APIVersionV1Alpha1)
	}
}

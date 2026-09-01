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
		"capability-pack.schema.json": "3170177c7911dc1766f2c8a6a7d17316b7526a0398c08f30a3130d4637042258",
		"manifest.schema.json":        "ec6e1aa9b9f3262438b51163d563d61cd29f14ba021e6a5e91320c5a50519f89",
		"plan.schema.json":            "f5631c86e1da56cfcfcd22c7cbce40d9ee3b24ebb0cca9dde752dfabad0c1eaa",
		"project.schema.json":         "0dcbde21075782cb1421b4ab4e53df154f70ed5b2e2e0bfe8adfbf4dd48e2aef",
		"result.schema.json":          "93b5d0465c8688e469b3ddffaa77c7081c0cf93cf414ea80541c65e0e374442a",
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

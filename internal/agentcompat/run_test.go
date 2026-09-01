package agentcompat

import (
	"context"
	"testing"
)

func TestRunConformsEveryAgentProfile(t *testing.T) {
	t.Parallel()

	report := Run(context.Background(), "test")
	if report.Status != "ok" {
		t.Fatalf("Run() status = %q, profiles = %#v", report.Status, report.Profiles)
	}
	if len(report.Profiles) != 6 {
		t.Fatalf("Run() profiles = %d, want 6", len(report.Profiles))
	}
	for _, profile := range report.Profiles {
		if profile.Status != "ok" || len(profile.Checks) != 12 {
			t.Errorf("profile %s = %#v", profile.Profile.ID, profile)
		}
		if profile.EstimatedContextTokens <= 0 || profile.EstimatedContextTokens > maximumContextTokens {
			t.Errorf("profile %s estimated tokens = %d", profile.Profile.ID, profile.EstimatedContextTokens)
		}
	}
}

func TestProfilesHaveUniqueStableIdentifiers(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for _, profile := range Profiles() {
		if _, exists := seen[profile.ID]; exists {
			t.Fatalf("duplicate profile ID %q", profile.ID)
		}
		seen[profile.ID] = struct{}{}
	}
}

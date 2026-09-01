package tokenbench

import (
	"context"
	"testing"
)

func TestFullSuiteTokenBudgets(t *testing.T) {
	t.Parallel()

	report := Run(context.Background(), "test")
	if report.Status != "ok" {
		t.Fatalf("Run() status = %q, scenarios = %#v", report.Status, report.Scenarios)
	}
	if len(report.Scenarios) != len(backends) {
		t.Fatalf("Run() scenarios = %d, want %d", len(report.Scenarios), len(backends))
	}
	for _, scenario := range report.Scenarios {
		if scenario.Status != "ok" || scenario.GeneratedFiles == 0 {
			t.Errorf("scenario %s = %#v", scenario.ID, scenario)
		}
		if scenario.ReductionPercent < minimumReductionPercent {
			t.Errorf("scenario %s reduction = %.2f%%", scenario.ID, scenario.ReductionPercent)
		}
		if scenario.EngineWorkflowEstimatedTokens > maximumWorkflowTokenBudget {
			t.Errorf("scenario %s workflow tokens = %d", scenario.ID, scenario.EngineWorkflowEstimatedTokens)
		}
	}
}

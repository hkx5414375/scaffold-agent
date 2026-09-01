// Package tokenbench measures bounded Engine context against generated source.
package tokenbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/artifactstore"
	"github.com/hkx5414375/scaffold-agent/internal/engine"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
)

const (
	reportAPIVersion           = "scaffold-agent.io/token-benchmark/v1alpha1"
	estimator                  = "ceil(UTF-8 bytes / 4); comparative estimate, not provider billing"
	minimumReductionPercent    = 90.0
	maximumWorkflowTokenBudget = 8_000
)

var backends = []string{"go", "java", "python"}

// Report contains reproducible, provider-neutral token estimates.
type Report struct {
	APIVersion string        `json:"api_version"`
	Status     string        `json:"status"`
	Estimator  string        `json:"estimator"`
	Scenarios  []Measurement `json:"scenarios"`
}

// Measurement compares reading every generated file once with the bounded
// Engine workflow needed to create and verify the same project.
type Measurement struct {
	ID                            string  `json:"id"`
	Backend                       string  `json:"backend"`
	Database                      string  `json:"database"`
	GeneratedFiles                int     `json:"generated_files"`
	GeneratedBytes                int     `json:"generated_bytes"`
	SourceContextEstimatedTokens  int     `json:"source_context_estimated_tokens"`
	EngineWorkflowEstimatedTokens int     `json:"engine_workflow_estimated_tokens"`
	SavedEstimatedTokens          int     `json:"saved_estimated_tokens"`
	ReductionPercent              float64 `json:"reduction_percent"`
	MinimumReductionPercent       float64 `json:"minimum_reduction_percent"`
	MaximumWorkflowTokenBudget    int     `json:"maximum_workflow_token_budget"`
	Status                        string  `json:"status"`
	Error                         string  `json:"error,omitempty"`
}

// Run measures a full reusable capability suite for every backend. It invokes
// no model, network service, package manager, compiler, or database.
func Run(ctx context.Context, version string) Report {
	report := Report{APIVersion: reportAPIVersion, Status: "ok", Estimator: estimator}
	for _, backend := range backends {
		measurement := measure(ctx, version, backend)
		report.Scenarios = append(report.Scenarios, measurement)
		if measurement.Status != "ok" {
			report.Status = "error"
		}
	}
	return report
}

func measure(ctx context.Context, version, backend string) Measurement {
	measurement := Measurement{
		ID:                         "full-suite-" + backend + "-postgresql",
		Backend:                    backend,
		Database:                   "postgresql",
		MinimumReductionPercent:    minimumReductionPercent,
		MaximumWorkflowTokenBudget: maximumWorkflowTokenBudget,
		Status:                     "error",
	}
	root, err := os.MkdirTemp("", "scaffold-agent-token-benchmark-")
	if err != nil {
		measurement.Error = "create isolated project: " + err.Error()
		return measurement
	}
	defer func() { _ = os.RemoveAll(root) }()
	blueprint := []byte(fmt.Sprintf(fullSuiteBlueprint, backend, backend))
	if err := os.WriteFile(filepath.Join(root, "scaffold.yaml"), blueprint, 0o600); err != nil {
		measurement.Error = "write benchmark Blueprint: " + err.Error()
		return measurement
	}

	application := engine.New(version)
	workflow := application.Query(ctx, engine.QueryInput{Topic: "workflow"})
	planned := application.Plan(ctx, engine.PlanInput{
		ProjectRoot:   root,
		BlueprintPath: "scaffold.yaml",
		Action:        plan.ActionCreate,
	})
	if workflow.Status != result.StatusOK || planned.Status != result.StatusOK {
		measurement.Error = "query or plan failed: " + planned.Summary
		return measurement
	}
	planData, err := envelopeData(planned)
	if err != nil {
		measurement.Error = err.Error()
		return measurement
	}
	planID, err := requiredString(planData, "plan_id")
	if err != nil {
		measurement.Error = err.Error()
		return measurement
	}
	artifact, err := artifactstore.Load(root, planID)
	if err != nil {
		measurement.Error = "load generated content: " + err.Error()
		return measurement
	}
	measurement.GeneratedFiles = len(artifact.Content)
	for _, content := range artifact.Content {
		measurement.GeneratedBytes += len(content)
	}

	previewed := application.Preview(ctx, engine.PreviewInput{
		ProjectRoot: root,
		PlanID:      planID,
		Limit:       20,
	})
	if previewed.Status != result.StatusOK {
		measurement.Error = "preview failed: " + previewed.Summary
		return measurement
	}
	previewData, err := envelopeData(previewed)
	if err != nil {
		measurement.Error = err.Error()
		return measurement
	}
	applyToken, err := requiredString(previewData, "apply_token")
	if err != nil {
		measurement.Error = err.Error()
		return measurement
	}
	applied := application.Apply(ctx, engine.ApplyInput{
		ProjectRoot: root,
		PlanID:      planID,
		ApplyToken:  applyToken,
	})
	if applied.Status != result.StatusOK {
		measurement.Error = "apply failed: " + applied.Summary
		return measurement
	}
	verified := application.Verify(ctx, engine.VerifyInput{ProjectRoot: root, Limit: 20})
	if verified.Status != result.StatusOK {
		measurement.Error = "verify failed: " + verified.Summary
		return measurement
	}

	measurement.SourceContextEstimatedTokens = estimateBytes(len(blueprint) + measurement.GeneratedBytes)
	measurement.EngineWorkflowEstimatedTokens = estimateBytes(len(blueprint))
	for _, envelope := range []result.Envelope{workflow, planned, previewed, applied, verified} {
		measurement.EngineWorkflowEstimatedTokens += estimateValue(envelope)
	}
	measurement.SavedEstimatedTokens = measurement.SourceContextEstimatedTokens - measurement.EngineWorkflowEstimatedTokens
	if measurement.SourceContextEstimatedTokens > 0 {
		measurement.ReductionPercent = roundTwoPlaces(
			100 * float64(measurement.SavedEstimatedTokens) / float64(measurement.SourceContextEstimatedTokens),
		)
	}
	if measurement.ReductionPercent < minimumReductionPercent {
		measurement.Error = fmt.Sprintf(
			"estimated reduction %.2f%% is below %.2f%%",
			measurement.ReductionPercent,
			minimumReductionPercent,
		)
		return measurement
	}
	if measurement.EngineWorkflowEstimatedTokens > maximumWorkflowTokenBudget {
		measurement.Error = fmt.Sprintf(
			"workflow estimate %d exceeds budget %d",
			measurement.EngineWorkflowEstimatedTokens,
			maximumWorkflowTokenBudget,
		)
		return measurement
	}
	measurement.Status = "ok"
	return measurement
}

func envelopeData(envelope result.Envelope) (map[string]any, error) {
	content, err := json.Marshal(envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("encode envelope data: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("decode envelope data: %w", err)
	}
	return data, nil
}

func requiredString(value map[string]any, name string) (string, error) {
	field, ok := value[name].(string)
	if !ok || strings.TrimSpace(field) == "" {
		return "", fmt.Errorf("result field %s is missing", name)
	}
	return field, nil
}

func estimateValue(value any) int {
	content, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return estimateBytes(len(content))
}

func estimateBytes(size int) int {
	return (size + 3) / 4
}

func roundTwoPlaces(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

const fullSuiteBlueprint = `api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: token-benchmark-%s
spec:
  stack:
    backend: %s
    admin_ui: element-plus
    storefront: nuxt
  database:
    engine: postgresql
  auth:
    modes: [session, token]
  capabilities:
    - {name: organization-tenancy, version: 0.3.0}
    - {name: background-jobs, version: 0.1.0}
    - {name: notifications, version: 0.1.0}
    - {name: file-assets, version: 0.1.0}
    - {name: application-cache, version: 0.1.0}
    - {name: job-administration, version: 0.1.0}
    - {name: observability, version: 0.1.0}
    - {name: csv-import-export, version: 0.1.0}
    - {name: approval-workflows, version: 0.1.0}
    - {name: commerce-catalog, version: 0.1.0}
    - {name: customer-accounts, version: 0.1.0}
    - {name: crm-core, version: 0.1.0}
    - {name: erp-inventory, version: 0.1.0}
  modules:
    - name: tasks
      entities:
        - name: task
          fields:
            - {name: title, type: string, required: true, unique: true}
            - {name: description, type: text}
            - {name: completed, type: bool, required: true}
            - {name: priority, type: int64, required: true}
            - {name: due_at, type: datetime}
      workflows:
        - name: approval
          states: [pending, approved, rejected, cancelled]
      permissions:
        - {code: "tasks:task:create"}
        - {code: "tasks:task:read"}
        - {code: "tasks:task:update"}
        - {code: "tasks:task:delete"}
`

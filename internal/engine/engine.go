// Package engine exposes model-neutral application services shared by CLI and MCP.
package engine

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/artifactstore"
	"github.com/hkx5414375/scaffold-agent/internal/canonicaljson"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	gogen "github.com/hkx5414375/scaffold-agent/internal/generator/golang"
	"github.com/hkx5414375/scaffold-agent/internal/manifest"
	"github.com/hkx5414375/scaffold-agent/internal/paging"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
	"github.com/hkx5414375/scaffold-agent/internal/projectmeta"
	"github.com/hkx5414375/scaffold-agent/internal/result"
	"github.com/hkx5414375/scaffold-agent/internal/resultstore"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
	publicschema "github.com/hkx5414375/scaffold-agent/schema"
)

const (
	defaultPageLimit = 20
	maximumPageLimit = 100
	maxBlueprintSize = 4 << 20
)

// MCPProtocolVersions are accepted by the STDIO adapter in newest-first order.
var MCPProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// Engine coordinates the stable application use cases without depending on a transport.
type Engine struct {
	version    string
	generators generator.Registry
}

// New returns an Engine with build-version reporting.
func New(version string) *Engine {
	return NewWithRegistry(version, generator.NewRegistry(gogen.New()))
}

// NewWithRegistry returns an Engine with an explicit language-adapter registry.
func NewWithRegistry(version string, registry generator.Registry) *Engine {
	return &Engine{version: version, generators: registry}
}

// Query returns compact support, workflow, or current project facts.
func (engine *Engine) Query(ctx context.Context, input QueryInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	switch input.Topic {
	case "support":
		return success("Engine support matrix", supportData{
			EngineVersion:       engine.version,
			MCPProtocolVersions: append([]string(nil), MCPProtocolVersions...),
			Implemented: []string{
				"blueprint-validation",
				"capability-resolution",
				"filesystem-transactions",
				"artifact-storage",
				"result-pagination",
				"mcp-stdio",
				"go-base-generator",
				"go-postgresql-identity",
				"go-postgresql-crud",
			},
			ContractTargets: map[string][]string{
				"backends":   {"go", "java", "python"},
				"databases":  {"postgresql", "mysql"},
				"admin_ui":   {"element-plus"},
				"storefront": {"nuxt"},
			},
		})
	case "workflow":
		return success("Required Agent workflow", workflowData{
			Steps: []string{"scaffold_query", "scaffold_plan", "scaffold_preview", "scaffold_apply", "scaffold_verify"},
			Rule:  "Never apply a plan without using the apply_token returned by preview.",
		})
	case "project":
		return engine.queryProject(input.ProjectRoot)
	default:
		return failure("query.topic.invalid", "topic", "topic must be support, workflow, or project")
	}
}

// Validate strictly decodes, schema-checks, and semantically validates one Blueprint.
func (engine *Engine) Validate(ctx context.Context, input ValidateInput) result.Envelope {
	project, hash, diagnostics := engine.validateProject(ctx, input)
	return validationEnvelope(project, hash, diagnostics)
}

func validationEnvelope(project spec.Project, hash string, diagnostics []spec.Diagnostic) result.Envelope {
	data := validationData{
		BlueprintHash:   hash,
		ProjectName:     project.Metadata.Name,
		Backend:         project.Spec.Stack.Backend,
		Database:        project.Spec.Database.Engine,
		CapabilityCount: len(project.Spec.Capabilities),
		ModuleCount:     len(project.Spec.Modules),
	}
	if spec.HasErrors(diagnostics) {
		return finalize(result.Envelope{
			APIVersion:  result.APIVersionV1Alpha1,
			Status:      result.StatusError,
			Summary:     "Blueprint validation failed",
			Diagnostics: diagnostics,
			Data:        data,
		})
	}
	return finalize(result.Envelope{
		APIVersion: result.APIVersionV1Alpha1,
		Status:     result.StatusOK,
		Summary:    "Blueprint is valid",
		Data:       data,
	})
}

// Plan validates the request before dispatching to a language adapter.
func (engine *Engine) Plan(ctx context.Context, input PlanInput) result.Envelope {
	if !validAction(input.Action) {
		return failure("plan.action.invalid", "action", "action must be create, modify, extend, reduce, repair, or upgrade")
	}
	project, blueprintHash, diagnostics := engine.validateProject(ctx, ValidateInput{ProjectRoot: input.ProjectRoot, BlueprintPath: input.BlueprintPath})
	if spec.HasErrors(diagnostics) {
		return validationEnvelope(project, blueprintHash, diagnostics)
	}
	adapter, exists := engine.generators.Get(project.Spec.Stack.Backend)
	if !exists {
		return failure("generator.adapter.unavailable", "spec.stack.backend", "the selected language generator is not installed in this build yet")
	}
	generated, err := adapter.Generate(ctx, project)
	if err != nil {
		return failure("generator.generate.failed", "blueprint_path", safeMessage(err))
	}
	artifact, err := change.Build(input.ProjectRoot, input.Action, blueprintHash, generated.CapabilityLock, generated.Outputs)
	if err != nil {
		return failure("plan.build.failed", "project_root", safeMessage(err))
	}
	if err := artifactstore.Save(input.ProjectRoot, artifact); err != nil {
		return failure("plan.save.failed", "project_root", safeMessage(err))
	}
	return success(fmt.Sprintf("Stored plan with %d filesystem changes", len(artifact.Plan.Changes)), planData{
		PlanID:         artifact.Plan.ID,
		Action:         artifact.Plan.Action,
		Backend:        project.Spec.Stack.Backend,
		BlueprintHash:  artifact.Plan.BlueprintHash,
		ProjectHash:    artifact.Plan.ProjectHash,
		CapabilityLock: cloneLock(artifact.Plan.CapabilityLock),
		ChangeCount:    len(artifact.Plan.Changes),
	})
}

// Preview returns a bounded change list and a token required by Apply.
func (engine *Engine) Preview(ctx context.Context, input PreviewInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	artifact, err := artifactstore.Load(input.ProjectRoot, input.PlanID)
	if err != nil {
		return failure("plan.load.failed", "plan_id", safeMessage(err))
	}
	offset, err := paging.Decode(input.Cursor, input.PlanID)
	if err != nil {
		return failure("plan.cursor.invalid", "cursor", err.Error())
	}
	start, end, err := paging.Bounds(len(artifact.Plan.Changes), offset, input.Limit, defaultPageLimit, maximumPageLimit)
	if err != nil {
		return failure("plan.page.invalid", "limit", err.Error())
	}
	token, err := applyToken(artifact.Plan)
	if err != nil {
		return failure("plan.preview.failed", "plan_id", safeMessage(err))
	}
	envelope := result.Envelope{
		APIVersion: result.APIVersionV1Alpha1,
		Status:     result.StatusOK,
		Summary:    fmt.Sprintf("Plan contains %d filesystem changes", len(artifact.Plan.Changes)),
		Data: previewData{
			PlanID:         artifact.Plan.ID,
			Action:         artifact.Plan.Action,
			BlueprintHash:  artifact.Plan.BlueprintHash,
			ProjectHash:    artifact.Plan.ProjectHash,
			CapabilityLock: cloneLock(artifact.Plan.CapabilityLock),
			ApplyToken:     token,
			TotalChanges:   len(artifact.Plan.Changes),
			Changes:        append([]plan.Change(nil), artifact.Plan.Changes[start:end]...),
		},
	}
	if end < len(artifact.Plan.Changes) {
		envelope.HasMore = true
		envelope.NextCursor, err = paging.Encode(input.PlanID, end)
		if err != nil {
			return failure("plan.cursor.failed", "cursor", safeMessage(err))
		}
	}
	return finalize(envelope)
}

// Apply verifies the preview token and applies one stored artifact transactionally.
func (engine *Engine) Apply(ctx context.Context, input ApplyInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	artifact, err := artifactstore.Load(input.ProjectRoot, input.PlanID)
	if err != nil {
		return failure("plan.load.failed", "plan_id", safeMessage(err))
	}
	wantToken, err := applyToken(artifact.Plan)
	if err != nil {
		return failure("plan.apply_token.failed", "apply_token", safeMessage(err))
	}
	if subtle.ConstantTimeCompare([]byte(input.ApplyToken), []byte(wantToken)) != 1 {
		return failure("plan.apply_token.invalid", "apply_token", "apply_token does not match the current immutable plan; preview it again")
	}
	receipt, err := change.Apply(artifact)
	if err != nil {
		return failure("plan.apply.failed", "plan_id", safeMessage(err))
	}
	return success("Plan applied successfully", receipt)
}

// Rollback restores one fully applied transaction after verifying its postconditions.
func (engine *Engine) Rollback(ctx context.Context, input TransactionInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	receipt, err := change.Rollback(input.ProjectRoot, input.PlanID)
	if err != nil {
		return failure("plan.rollback.failed", "plan_id", safeMessage(err))
	}
	return success("Plan rolled back successfully", receipt)
}

// Recover restores a transaction interrupted before it reached applied status.
func (engine *Engine) Recover(ctx context.Context, input TransactionInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	receipt, err := change.Recover(input.ProjectRoot, input.PlanID)
	if err != nil {
		return failure("plan.recovery.failed", "plan_id", safeMessage(err))
	}
	return success("Interrupted plan recovered successfully", receipt)
}

// Verify hashes every managed file and stores pageable findings.
func (engine *Engine) Verify(ctx context.Context, input VerifyInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	root, err := projectfs.Open(input.ProjectRoot)
	if err != nil {
		return failure("project.root.invalid", "project_root", safeMessage(err))
	}
	loaded, err := manifest.Load(root)
	if err != nil {
		return failure("manifest.load.failed", manifest.Path, safeMessage(err))
	}
	if !loaded.Exists {
		return failure("manifest.missing", manifest.Path, "project has no Scaffold Agent ownership manifest")
	}
	paths := make([]string, 0, len(loaded.Document.Files))
	for relativePath := range loaded.Document.Files {
		paths = append(paths, relativePath)
	}
	sort.Strings(paths)
	findings := make([]any, 0)
	for _, relativePath := range paths {
		if err := ctx.Err(); err != nil {
			return failure("request.cancelled", "", err.Error())
		}
		managedFile := loaded.Document.Files[relativePath]
		target, resolveErr := root.Resolve(relativePath)
		if resolveErr != nil {
			findings = append(findings, verificationFinding{Path: relativePath, Owner: managedFile.Owner, Problem: "invalid_path", ExpectedHash: managedFile.Hash})
			continue
		}
		currentHash, hashErr := projectfs.HashFile(target)
		switch {
		case errors.Is(hashErr, os.ErrNotExist):
			findings = append(findings, verificationFinding{Path: relativePath, Owner: managedFile.Owner, Problem: "missing", ExpectedHash: managedFile.Hash})
		case hashErr != nil:
			findings = append(findings, verificationFinding{Path: relativePath, Owner: managedFile.Owner, Problem: "unreadable", ExpectedHash: managedFile.Hash})
		case currentHash != managedFile.Hash:
			findings = append(findings, verificationFinding{Path: relativePath, Owner: managedFile.Owner, Problem: "hash_mismatch", ExpectedHash: managedFile.Hash, CurrentHash: currentHash})
		}
	}
	status := result.StatusOK
	summary := fmt.Sprintf("Verified %d managed files with no findings", len(paths))
	if len(findings) > 0 {
		status = result.StatusError
		summary = fmt.Sprintf("Verified %d managed files and found %d problems", len(paths), len(findings))
	}
	resultID, err := resultstore.Save(root.Path(), resultstore.Record{
		Status:   status,
		Summary:  summary,
		Metadata: map[string]any{"checked_files": len(paths), "finding_count": len(findings)},
		Items:    findings,
	})
	if err != nil {
		return failure("result.save.failed", "project_root", safeMessage(err))
	}
	return engine.Result(ctx, ResultInput{ProjectRoot: root.Path(), ResultID: resultID, Limit: input.Limit})
}

// Result returns one stored-result page.
func (engine *Engine) Result(ctx context.Context, input ResultInput) result.Envelope {
	if err := ctx.Err(); err != nil {
		return failure("request.cancelled", "", err.Error())
	}
	envelope, err := resultstore.Page(input.ProjectRoot, input.ResultID, input.Cursor, input.Limit)
	if err != nil {
		return failure("result.page.failed", "result_id", safeMessage(err))
	}
	if data, ok := envelope.Data.(resultstore.PageData); ok {
		envelope.Data = resultPageData{Metadata: data.Metadata, Items: data.Items}
	}
	return finalize(envelope)
}

func (engine *Engine) queryProject(rootPath string) result.Envelope {
	root, err := projectfs.Open(rootPath)
	if err != nil {
		return failure("project.root.invalid", "project_root", safeMessage(err))
	}
	loaded, err := manifest.Load(root)
	if err != nil {
		return failure("manifest.load.failed", manifest.Path, safeMessage(err))
	}
	if !loaded.Exists {
		return failure("manifest.missing", manifest.Path, "project has no Scaffold Agent ownership manifest")
	}
	return success("Current managed project state", projectData{
		BlueprintHash:    loaded.Document.BlueprintHash,
		CapabilityLock:   cloneLock(loaded.Document.CapabilityLock),
		ManagedFileCount: len(loaded.Document.Files),
	})
}

func (engine *Engine) validateProject(ctx context.Context, input ValidateInput) (spec.Project, string, []spec.Diagnostic) {
	if err := ctx.Err(); err != nil {
		return spec.Project{}, "", []spec.Diagnostic{diagnostic("request.cancelled", "", err.Error())}
	}
	root, err := projectfs.Open(input.ProjectRoot)
	if err != nil {
		return spec.Project{}, "", []spec.Diagnostic{diagnostic("project.root.invalid", "project_root", safeMessage(err))}
	}
	target, err := root.Resolve(input.BlueprintPath)
	if err != nil {
		return spec.Project{}, "", []spec.Diagnostic{diagnostic("project.blueprint_path.invalid", "blueprint_path", safeMessage(err))}
	}
	content, err := projectmeta.ReadRegularFile(target, maxBlueprintSize)
	if err != nil {
		return spec.Project{}, "", []spec.Diagnostic{diagnostic("project.blueprint.read_failed", input.BlueprintPath, safeMessage(err))}
	}
	project, err := spec.DecodeProject(content, spec.DetectFormat(content))
	if err != nil {
		return spec.Project{}, "", []spec.Diagnostic{diagnostic("project.decode.failed", input.BlueprintPath, safeMessage(err))}
	}
	hash, err := canonicaljson.Hash(project)
	if err != nil {
		return project, "", []spec.Diagnostic{diagnostic("project.hash.failed", input.BlueprintPath, safeMessage(err))}
	}
	diagnostics := spec.ValidateProject(project)
	if schemaErr := publicschema.Validate("v1alpha1", "project.schema.json", project); schemaErr != nil {
		diagnostics = append(diagnostics, diagnostic("project.schema.invalid", input.BlueprintPath, safeMessage(schemaErr)))
	}
	return project, hash, diagnostics
}

func applyToken(value plan.Plan) (string, error) {
	hash, err := canonicaljson.Hash(map[string]any{
		"plan_id":      value.ID,
		"project_hash": value.ProjectHash,
		"changes":      value.Changes,
	})
	if err != nil {
		return "", err
	}
	return "apply_" + hash, nil
}

func validAction(action plan.Action) bool {
	switch action {
	case plan.ActionCreate, plan.ActionModify, plan.ActionExtend, plan.ActionReduce, plan.ActionRepair, plan.ActionUpgrade:
		return true
	default:
		return false
	}
}

func success(summary string, data any) result.Envelope {
	return finalize(result.Envelope{APIVersion: result.APIVersionV1Alpha1, Status: result.StatusOK, Summary: summary, Data: data})
}

func failure(code, path, message string) result.Envelope {
	return finalize(result.Envelope{
		APIVersion:  result.APIVersionV1Alpha1,
		Status:      result.StatusError,
		Summary:     message,
		Diagnostics: []spec.Diagnostic{diagnostic(code, path, message)},
	})
}

func diagnostic(code, path, message string) spec.Diagnostic {
	return spec.Diagnostic{Code: code, Severity: spec.SeverityError, Path: path, Message: message}
}

func finalize(envelope result.Envelope) result.Envelope {
	envelope.EstimatedTokens = 0
	content, err := json.Marshal(envelope)
	if err == nil {
		envelope.EstimatedTokens = (len(content) + 3) / 4
	}
	return envelope
}

func safeMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) <= 500 {
		return message
	}
	return message[:500] + "..."
}

func cloneLock(lock map[string]string) map[string]string {
	cloned := make(map[string]string, len(lock))
	for name, version := range lock {
		cloned[name] = version
	}
	return cloned
}

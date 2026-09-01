package generator_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
	golangadapter "github.com/hkx5414375/scaffold-agent/internal/generator/golang"
	javaadapter "github.com/hkx5414375/scaffold-agent/internal/generator/java"
	pythonadapter "github.com/hkx5414375/scaffold-agent/internal/generator/python"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
	"go.yaml.in/yaml/v3"
)

type crmOperation struct {
	OperationID string
	Permission  string
}

type crmShape struct {
	AccountStatuses  []string
	ActivitySubjects []string
	ActivityTypes    []string
	OpportunityStage []string
	TransitionStage  []string
	AmountType       string
	AmountPattern    string
	CurrencyPattern  string
	VersionType      string
	VersionPattern   string
}

func TestCRMCoreConformsAcrossBackends(t *testing.T) {
	t.Parallel()

	adapters := map[string]generator.Adapter{
		"go":     golangadapter.New(),
		"java":   javaadapter.New(),
		"python": pythonadapter.New(),
	}
	wantOperations := map[string]crmOperation{
		"POST /api/v1/crm/accounts":                   {"createCRMAccount", "crm:write"},
		"GET /api/v1/crm/accounts":                    {"listCRMAccounts", "crm:read"},
		"GET /api/v1/crm/accounts/{id}":               {"getCRMAccount", "crm:read"},
		"PUT /api/v1/crm/accounts/{id}":               {"updateCRMAccount", "crm:write"},
		"POST /api/v1/crm/accounts/{id}/archive":      {"archiveCRMAccount", "crm:write"},
		"POST /api/v1/crm/accounts/{id}/reactivate":   {"reactivateCRMAccount", "crm:write"},
		"POST /api/v1/crm/contacts":                   {"createCRMContact", "crm:write"},
		"GET /api/v1/crm/contacts":                    {"listCRMContacts", "crm:read"},
		"GET /api/v1/crm/contacts/{id}":               {"getCRMContact", "crm:read"},
		"PUT /api/v1/crm/contacts/{id}":               {"updateCRMContact", "crm:write"},
		"POST /api/v1/crm/contacts/{id}/archive":      {"archiveCRMContact", "crm:write"},
		"POST /api/v1/crm/contacts/{id}/reactivate":   {"reactivateCRMContact", "crm:write"},
		"POST /api/v1/crm/activities":                 {"createCRMActivity", "crm:write"},
		"GET /api/v1/crm/activities":                  {"listCRMActivities", "crm:read"},
		"POST /api/v1/crm/opportunities":              {"createCRMOpportunity", "crm:write"},
		"GET /api/v1/crm/opportunities":               {"listCRMOpportunities", "crm:read"},
		"GET /api/v1/crm/opportunities/{id}":          {"getCRMOpportunity", "crm:read"},
		"PUT /api/v1/crm/opportunities/{id}":          {"updateCRMOpportunity", "crm:write"},
		"POST /api/v1/crm/opportunities/{id}/advance": {"advanceCRMOpportunity", "crm:pipeline:manage"},
	}
	wantShape := crmShape{
		AccountStatuses:  []string{"active", "archived"},
		ActivitySubjects: []string{"account", "contact", "opportunity"},
		ActivityTypes:    []string{"note", "call", "email", "meeting"},
		OpportunityStage: []string{"lead", "qualified", "proposal", "won", "lost"},
		TransitionStage:  []string{"qualified", "proposal", "won", "lost"},
		AmountType:       "string",
		AmountPattern:    "^(0|[1-9][0-9]{0,18})$",
		CurrencyPattern:  "^[A-Z]{3}$",
		VersionType:      "string",
		VersionPattern:   "^[1-9][0-9]*$",
	}
	wantSchemas := []string{
		"CRMAccount", "CRMAccountWrite", "CRMAccountUpdate", "CRMAccountPage",
		"CRMContact", "CRMContactWrite", "CRMContactUpdate", "CRMContactPage",
		"CRMActivity", "CRMActivityWrite", "CRMActivityPage",
		"CRMOpportunity", "CRMOpportunityWrite", "CRMOpportunityUpdate", "CRMOpportunityPage",
		"CRMTransition", "CRMStageTransition",
	}
	wantMigrationFragments := []string{
		"crm_accounts", "crm_contacts", "crm_activities", "crm_opportunities",
		"crm:read", "crm:write", "crm:pipeline:manage",
	}
	migrationPaths := map[string]string{
		"go":     "internal/platform/migrate/migrations/000280_crm_core.sql",
		"java":   "src/main/resources/db/migration/V000280__crm_core.sql",
		"python": "src/crm_conformance/migration/versions/000280_crm.py",
	}

	var sharedView []byte
	for backend, adapter := range adapters {
		backend := backend
		adapter := adapter
		t.Run(backend, func(t *testing.T) {
			project := crmProject(backend)
			generated, err := adapter.Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock["crm-core"] != "0.1.0" ||
				generated.CapabilityLock["organization-tenancy"] != "0.3.0" {
				t.Fatalf("capability lock = %#v", generated.CapabilityLock)
			}

			contract := decodeOpenAPI(t, output(t, generated, "api/openapi.yaml"))
			if got := crmOperations(t, contract); !reflect.DeepEqual(got, wantOperations) {
				t.Fatalf("CRM operations = %#v, want %#v", got, wantOperations)
			}
			if got := openAPIShape(t, contract); !reflect.DeepEqual(got, wantShape) {
				t.Fatalf("CRM shape = %#v, want %#v", got, wantShape)
			}
			schemas := mapping(t, mapping(t, contract, "components"), "schemas")
			for _, name := range wantSchemas {
				if _, exists := schemas[name]; !exists {
					t.Errorf("OpenAPI schema %s is missing", name)
				}
			}

			migration := string(output(t, generated, migrationPaths[backend]))
			for _, fragment := range wantMigrationFragments {
				if !strings.Contains(migration, fragment) {
					t.Errorf("CRM migration does not contain %q", fragment)
				}
			}
			view := output(t, generated, "web/admin/src/views/CRMView.vue")
			if sharedView == nil {
				sharedView = view
			} else if !reflect.DeepEqual(view, sharedView) {
				t.Error("shared CRM administration view differs across backends")
			}
		})
	}
}

func crmProject(backend string) spec.Project {
	return spec.Project{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindProject,
		Metadata: spec.Metadata{
			Name:        "crm-conformance",
			DisplayName: "CRM Conformance",
		},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: backend, AdminUI: "element-plus", Storefront: "none"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
			Capabilities: []spec.CapabilitySelection{
				{Name: "organization-tenancy", Version: "0.3.0"},
				{Name: "crm-core", Version: "0.1.0"},
			},
		},
	}
}

func crmOperations(t *testing.T, contract map[string]any) map[string]crmOperation {
	t.Helper()
	result := make(map[string]crmOperation)
	for path, rawPath := range mapping(t, contract, "paths") {
		if !strings.HasPrefix(path, "/api/v1/crm/") {
			continue
		}
		pathItem, ok := rawPath.(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI path %s is not an object", path)
		}
		for _, method := range []string{"get", "post", "put"} {
			rawOperation, exists := pathItem[method]
			if !exists {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				t.Fatalf("OpenAPI operation %s %s is not an object", method, path)
			}
			result[strings.ToUpper(method)+" "+path] = crmOperation{
				OperationID: scalarString(t, operation, "operationId"),
				Permission:  scalarString(t, operation, "x-required-permission"),
			}
		}
	}
	return result
}

func openAPIShape(t *testing.T, contract map[string]any) crmShape {
	t.Helper()
	schemas := mapping(t, mapping(t, contract, "components"), "schemas")
	account := properties(t, schemas, "CRMAccount")
	activity := properties(t, schemas, "CRMActivity")
	opportunity := properties(t, schemas, "CRMOpportunity")
	transition := properties(t, schemas, "CRMStageTransition")
	amount := mapping(t, opportunity, "amount_minor")
	currency := mapping(t, opportunity, "currency")
	version := mapping(t, account, "version")
	return crmShape{
		AccountStatuses:  stringSlice(t, mapping(t, account, "status"), "enum"),
		ActivitySubjects: stringSlice(t, mapping(t, activity, "subject_type"), "enum"),
		ActivityTypes:    stringSlice(t, mapping(t, activity, "type"), "enum"),
		OpportunityStage: stringSlice(t, mapping(t, opportunity, "stage"), "enum"),
		TransitionStage:  stringSlice(t, mapping(t, transition, "stage"), "enum"),
		AmountType:       scalarString(t, amount, "type"),
		AmountPattern:    scalarString(t, amount, "pattern"),
		CurrencyPattern:  scalarString(t, currency, "pattern"),
		VersionType:      scalarString(t, version, "type"),
		VersionPattern:   scalarString(t, version, "pattern"),
	}
}

func properties(t *testing.T, schemas map[string]any, name string) map[string]any {
	t.Helper()
	return mapping(t, mapping(t, schemas, name), "properties")
}

func decodeOpenAPI(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var contract map[string]any
	if err := yaml.Unmarshal(content, &contract); err != nil {
		t.Fatalf("OpenAPI is invalid YAML: %v", err)
	}
	return contract
}

func mapping(t *testing.T, source map[string]any, key string) map[string]any {
	t.Helper()
	value, exists := source[key]
	if !exists {
		t.Fatalf("OpenAPI key %s is missing", key)
	}
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI key %s is not an object", key)
	}
	return result
}

func scalarString(t *testing.T, source map[string]any, key string) string {
	t.Helper()
	value, exists := source[key]
	if !exists {
		t.Fatalf("OpenAPI key %s is missing", key)
	}
	result, ok := value.(string)
	if !ok {
		t.Fatalf("OpenAPI key %s is not a string", key)
	}
	return result
}

func stringSlice(t *testing.T, source map[string]any, key string) []string {
	t.Helper()
	value, exists := source[key]
	if !exists {
		t.Fatalf("OpenAPI key %s is missing", key)
	}
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("OpenAPI key %s is not an array", key)
	}
	result := make([]string, len(values))
	for index, value := range values {
		item, ok := value.(string)
		if !ok {
			t.Fatalf("OpenAPI key %s contains a non-string value", key)
		}
		result[index] = item
	}
	return result
}

func output(t *testing.T, generated generator.Result, path string) []byte {
	t.Helper()
	for _, candidate := range generated.Outputs {
		if candidate.Path == path {
			return candidate.Content
		}
	}
	t.Fatalf("generated output %s is missing", path)
	return nil
}

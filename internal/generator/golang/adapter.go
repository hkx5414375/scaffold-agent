// Package gogen implements the first-party Go modular-monolith adapter.
package gogen

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/hkx5414375/scaffold-agent/internal/capability"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	adminui "github.com/hkx5414375/scaffold-agent/internal/generator/admin"
	storefrontui "github.com/hkx5414375/scaffold-agent/internal/generator/storefront"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend                 = "go"
	baseCapability          = "go-service"
	baseVersion             = "0.3.0"
	businessCapability      = "go-crud"
	businessVersion         = "0.3.0"
	adminCapability         = adminui.Owner
	adminVersion            = adminui.Version
	storefrontCapability    = storefrontui.Owner
	storefrontVersion       = storefrontui.Version
	tenancyCapability       = "organization-tenancy"
	tenancyVersion          = "0.1.0"
	tenancyMembersVersion   = "0.2.0"
	tenancyLifecycleVersion = "0.3.0"
	jobsCapability          = "background-jobs"
	jobsVersion             = "0.1.0"
	notificationsCapability = "notifications"
	notificationsVersion    = "0.1.0"
	filesCapability         = "file-assets"
	filesVersion            = "0.1.0"
	cacheCapability         = "application-cache"
	cacheVersion            = "0.1.0"
	jobAdminCapability      = "job-administration"
	jobAdminVersion         = "0.1.0"
	observabilityCapability = "observability"
	observabilityVersion    = "0.1.0"
	csvTransferCapability   = "csv-import-export"
	csvTransferVersion      = "0.1.0"
	approvalsCapability     = "approval-workflows"
	approvalsVersion        = "0.1.0"
	catalogCapability       = "commerce-catalog"
	catalogVersion          = "0.1.0"
	customerCapability      = "customer-accounts"
	customerVersion         = "0.1.0"
)

//go:embed all:templates
var templateFS embed.FS

var outputTemplates = map[string]string{
	".gitignore":                                  "templates/gitignore.tmpl",
	"README.md":                                   "templates/README.md.tmpl",
	"api/openapi.yaml":                            "templates/openapi.yaml.tmpl",
	"cmd/server/main.go":                          "templates/main.go.tmpl",
	"internal/identity/httpapi/handler.go":        "templates/identity_handler.go.tmpl",
	"internal/identity/httpapi/handler_test.go":   "templates/identity_handler_test.go.tmpl",
	"internal/identity/password.go":               "templates/password.go.tmpl",
	"internal/identity/password_test.go":          "templates/password_test.go.tmpl",
	"internal/identity/service.go":                "templates/identity_service.go.tmpl",
	"internal/identity/service_test.go":           "templates/identity_service_test.go.tmpl",
	"internal/identity/store.go":                  "templates/identity_store.go.tmpl",
	"internal/platform/httpserver/server.go":      "templates/server.go.tmpl",
	"internal/platform/httpserver/server_test.go": "templates/server_test.go.tmpl",
	"internal/platform/httpjson/httpjson.go":      "templates/httpjson.go.tmpl",
}

type businessTemplate struct {
	PathSuffix   string
	TemplatePath string
}

var businessTemplates = []businessTemplate{
	{PathSuffix: "entity.go", TemplatePath: "templates/business_entity.go.tmpl"},
	{PathSuffix: "entity_test.go", TemplatePath: "templates/business_entity_test.go.tmpl"},
	{PathSuffix: "httpapi/handler.go", TemplatePath: "templates/business_handler.go.tmpl"},
}

type databaseTemplateSet struct {
	Data                              databaseData
	Outputs                           map[string]string
	BusinessStorePath                 string
	BusinessStoreTemplate             string
	BusinessMigrationTemplate         string
	IntegrationPath                   string
	IntegrationTemplate               string
	TenancyStorePath                  string
	TenancyStoreTemplate              string
	TenancyMigrationTemplate          string
	TenancyMembersStorePath           string
	TenancyMembersStoreTemplate       string
	TenancyMembersMigrationTemplate   string
	TenancyLifecycleStorePath         string
	TenancyLifecycleStoreTemplate     string
	TenancyLifecycleMigrationTemplate string
	JobsStorePath                     string
	JobsStoreTemplate                 string
	JobsMigrationTemplate             string
	FilesStorePath                    string
	FilesStoreTemplate                string
	FilesMigrationTemplate            string
	CacheStorePath                    string
	CacheStoreTemplate                string
	CacheMigrationTemplate            string
	JobAdminStorePath                 string
	JobAdminStoreTemplate             string
	JobAdminMigrationTemplate         string
	CSVTransferStorePath              string
	CSVTransferStoreTemplate          string
	CSVTransferMigrationTemplate      string
	ApprovalsStorePath                string
	ApprovalsStoreTemplate            string
	ApprovalsMigrationTemplate        string
	CatalogStorePath                  string
	CatalogStoreTemplate              string
	CatalogMigrationTemplate          string
	CustomerStorePath                 string
	CustomerStoreTemplate             string
	CustomerMigrationTemplate         string
}

var databaseTemplates = map[string]databaseTemplateSet{
	"postgresql": {
		Data: databaseData{Engine: "postgresql", DisplayName: "PostgreSQL", PackageName: "postgres"},
		Outputs: map[string]string{
			"go.mod":                               "templates/go.mod.tmpl",
			"go.sum":                               "templates/go.sum.tmpl",
			"internal/identity/postgres/store.go":  "templates/identity_postgres_store.go.tmpl",
			"internal/platform/migrate/migrate.go": "templates/migrate.go.tmpl",
			"internal/platform/migrate/migrations/000001_identity.sql": "templates/000001_identity.sql.tmpl",
			"internal/platform/postgres/pool.go":                       "templates/postgres_pool.go.tmpl",
		},
		BusinessStorePath:                 "postgres/store.go",
		BusinessStoreTemplate:             "templates/business_postgres_store.go.tmpl",
		BusinessMigrationTemplate:         "templates/business.sql.tmpl",
		IntegrationPath:                   "internal/integration/postgres_test.go",
		IntegrationTemplate:               "templates/postgres_integration_test.go.tmpl",
		TenancyStorePath:                  "internal/tenancy/postgres/store.go",
		TenancyStoreTemplate:              "templates/tenancy_postgres_store.go.tmpl",
		TenancyMigrationTemplate:          "templates/tenancy_postgres.sql.tmpl",
		TenancyMembersStorePath:           "internal/tenancy/postgres/members.go",
		TenancyMembersStoreTemplate:       "templates/tenancy_members_postgres_store.go.tmpl",
		TenancyMembersMigrationTemplate:   "templates/tenancy_members_postgres.sql.tmpl",
		TenancyLifecycleStorePath:         "internal/tenancy/postgres/lifecycle.go",
		TenancyLifecycleStoreTemplate:     "templates/tenancy_lifecycle_postgres_store.go.tmpl",
		TenancyLifecycleMigrationTemplate: "templates/tenancy_lifecycle_postgres.sql.tmpl",
		JobsStorePath:                     "internal/jobs/postgres/store.go",
		JobsStoreTemplate:                 "templates/jobs_postgres_store.go.tmpl",
		JobsMigrationTemplate:             "templates/jobs_postgres.sql.tmpl",
		FilesStorePath:                    "internal/files/postgres/store.go",
		FilesStoreTemplate:                "templates/files_postgres_store.go.tmpl",
		FilesMigrationTemplate:            "templates/files_postgres.sql.tmpl",
		CacheStorePath:                    "internal/cache/postgres/store.go",
		CacheStoreTemplate:                "templates/cache_postgres_store.go.tmpl",
		CacheMigrationTemplate:            "templates/cache_postgres.sql.tmpl",
		JobAdminStorePath:                 "internal/jobs/postgres/admin.go",
		JobAdminStoreTemplate:             "templates/jobadmin_postgres_store.go.tmpl",
		JobAdminMigrationTemplate:         "templates/jobadmin_postgres.sql.tmpl",
		CSVTransferStorePath:              "transfer/postgres/store.go",
		CSVTransferStoreTemplate:          "templates/csv_transfer_postgres_store.go.tmpl",
		CSVTransferMigrationTemplate:      "templates/csv_transfer_postgres.sql.tmpl",
		ApprovalsStorePath:                "internal/approvals/postgres/store.go",
		ApprovalsStoreTemplate:            "templates/approvals_postgres_store.go.tmpl",
		ApprovalsMigrationTemplate:        "templates/approvals_postgres.sql.tmpl",
		CatalogStorePath:                  "internal/catalog/postgres/store.go",
		CatalogStoreTemplate:              "templates/catalog_postgres_store.go.tmpl",
		CatalogMigrationTemplate:          "templates/catalog_postgres.sql.tmpl",
		CustomerStorePath:                 "internal/customeraccounts/postgres/store.go",
		CustomerStoreTemplate:             "templates/customer_accounts_postgres_store.go.tmpl",
		CustomerMigrationTemplate:         "templates/customer_accounts_postgres.sql.tmpl",
	},
	"mysql": {
		Data: databaseData{Engine: "mysql", DisplayName: "MySQL", PackageName: "mysql"},
		Outputs: map[string]string{
			"go.mod":                               "templates/mysql_go.mod.tmpl",
			"go.sum":                               "templates/mysql_go.sum.tmpl",
			"internal/identity/mysql/store.go":     "templates/identity_mysql_store.go.tmpl",
			"internal/platform/migrate/migrate.go": "templates/mysql_migrate.go.tmpl",
			"internal/platform/migrate/migrate_test.go":                "templates/mysql_migrate_test.go.tmpl",
			"internal/platform/migrate/migrations/000001_identity.sql": "templates/mysql_000001_identity.sql.tmpl",
			"internal/platform/mysql/pool.go":                          "templates/mysql_pool.go.tmpl",
		},
		BusinessStorePath:                 "mysql/store.go",
		BusinessStoreTemplate:             "templates/business_mysql_store.go.tmpl",
		BusinessMigrationTemplate:         "templates/mysql_business.sql.tmpl",
		IntegrationPath:                   "internal/integration/mysql_test.go",
		IntegrationTemplate:               "templates/mysql_integration_test.go.tmpl",
		TenancyStorePath:                  "internal/tenancy/mysql/store.go",
		TenancyStoreTemplate:              "templates/tenancy_mysql_store.go.tmpl",
		TenancyMigrationTemplate:          "templates/tenancy_mysql.sql.tmpl",
		TenancyMembersStorePath:           "internal/tenancy/mysql/members.go",
		TenancyMembersStoreTemplate:       "templates/tenancy_members_mysql_store.go.tmpl",
		TenancyMembersMigrationTemplate:   "templates/tenancy_members_mysql.sql.tmpl",
		TenancyLifecycleStorePath:         "internal/tenancy/mysql/lifecycle.go",
		TenancyLifecycleStoreTemplate:     "templates/tenancy_lifecycle_mysql_store.go.tmpl",
		TenancyLifecycleMigrationTemplate: "templates/tenancy_lifecycle_mysql.sql.tmpl",
		JobsStorePath:                     "internal/jobs/mysql/store.go",
		JobsStoreTemplate:                 "templates/jobs_mysql_store.go.tmpl",
		JobsMigrationTemplate:             "templates/jobs_mysql.sql.tmpl",
		FilesStorePath:                    "internal/files/mysql/store.go",
		FilesStoreTemplate:                "templates/files_mysql_store.go.tmpl",
		FilesMigrationTemplate:            "templates/files_mysql.sql.tmpl",
		CacheStorePath:                    "internal/cache/mysql/store.go",
		CacheStoreTemplate:                "templates/cache_mysql_store.go.tmpl",
		CacheMigrationTemplate:            "templates/cache_mysql.sql.tmpl",
		JobAdminStorePath:                 "internal/jobs/mysql/admin.go",
		JobAdminStoreTemplate:             "templates/jobadmin_mysql_store.go.tmpl",
		JobAdminMigrationTemplate:         "templates/jobadmin_mysql.sql.tmpl",
		CSVTransferStorePath:              "transfer/mysql/store.go",
		CSVTransferStoreTemplate:          "templates/csv_transfer_mysql_store.go.tmpl",
		CSVTransferMigrationTemplate:      "templates/csv_transfer_mysql.sql.tmpl",
		ApprovalsStorePath:                "internal/approvals/mysql/store.go",
		ApprovalsStoreTemplate:            "templates/approvals_mysql_store.go.tmpl",
		ApprovalsMigrationTemplate:        "templates/approvals_mysql.sql.tmpl",
		CatalogStorePath:                  "internal/catalog/mysql/store.go",
		CatalogStoreTemplate:              "templates/catalog_mysql_store.go.tmpl",
		CatalogMigrationTemplate:          "templates/catalog_mysql.sql.tmpl",
		CustomerStorePath:                 "internal/customeraccounts/mysql/store.go",
		CustomerStoreTemplate:             "templates/customer_accounts_mysql_store.go.tmpl",
		CustomerMigrationTemplate:         "templates/customer_accounts_mysql.sql.tmpl",
	},
}

var goCapabilityCatalog = capability.NewCatalog(
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: tenancyCapability, Version: tenancyVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Organization membership and tenant-scoped authorization",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: tenancyCapability, Version: tenancyLifecycleVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Organization ownership, lifecycle, invitations, and tenant-scoped authorization",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: tenancyCapability, Version: tenancyMembersVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Organization membership, invitations, and tenant-scoped authorization",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: jobsCapability, Version: jobsVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Reliable leased background jobs with retries and idempotent enqueue",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: notificationsCapability, Version: notificationsVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Idempotent email notifications delivered by reliable background jobs",
			Requires:    []spec.PackDependency{{Name: jobsCapability, Constraint: "^0.1.0"}},
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: filesCapability, Version: filesVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Tenant-aware file metadata and atomic local object storage",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: cacheCapability, Version: cacheVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Tenant-aware database TTL cache with bounded JSON values",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: jobAdminCapability, Version: jobAdminVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Permission-protected background job inspection and dead-job retry",
			Requires:    []spec.PackDependency{{Name: jobsCapability, Constraint: "^0.1.0"}},
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: observabilityCapability, Version: observabilityVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Request correlation, safe access logs, HTTP metrics, and database readiness",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: csvTransferCapability, Version: csvTransferVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Atomic bounded CSV import and audited safe CSV export for the generated business entity",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: approvalsCapability, Version: approvalsVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Tenant-aware single-stage approval state machine with separation of duties and immutable events",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: catalogCapability, Version: catalogVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Portable product catalog with audited publication and public storefront reads",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: customerCapability, Version: customerVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Separate storefront customer identities, sessions, lifecycle, and administration",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
)

var tenancyTemplates = map[string]string{
	"internal/tenancy/service.go":              "templates/tenancy_service.go.tmpl",
	"internal/tenancy/service_test.go":         "templates/tenancy_service_test.go.tmpl",
	"internal/tenancy/httpapi/handler.go":      "templates/tenancy_handler.go.tmpl",
	"internal/tenancy/httpapi/handler_test.go": "templates/tenancy_handler_test.go.tmpl",
}

var tenancyMembersTemplates = map[string]string{
	"internal/tenancy/members.go":                      "templates/tenancy_members.go.tmpl",
	"internal/tenancy/members_test.go":                 "templates/tenancy_members_test.go.tmpl",
	"internal/tenancy/httpapi/members_handler.go":      "templates/tenancy_members_handler.go.tmpl",
	"internal/tenancy/httpapi/members_handler_test.go": "templates/tenancy_members_handler_test.go.tmpl",
}

var tenancyLifecycleTemplates = map[string]string{
	"internal/tenancy/lifecycle.go":                      "templates/tenancy_lifecycle.go.tmpl",
	"internal/tenancy/lifecycle_test.go":                 "templates/tenancy_lifecycle_test.go.tmpl",
	"internal/tenancy/httpapi/lifecycle_handler.go":      "templates/tenancy_lifecycle_handler.go.tmpl",
	"internal/tenancy/httpapi/lifecycle_handler_test.go": "templates/tenancy_lifecycle_handler_test.go.tmpl",
}

var jobsTemplates = map[string]string{
	"cmd/worker/main.go":               "templates/jobs_main.go.tmpl",
	"internal/jobhandlers/registry.go": "templates/jobs_registry.go.tmpl",
	"internal/jobs/jobs.go":            "templates/jobs.go.tmpl",
	"internal/jobs/jobs_test.go":       "templates/jobs_test.go.tmpl",
	"internal/jobs/worker.go":          "templates/jobs_worker.go.tmpl",
	"internal/jobs/worker_test.go":     "templates/jobs_worker_test.go.tmpl",
}

var notificationsTemplates = map[string]string{
	"internal/jobhandlers/notifications.go":        "templates/notifications_jobhandler.go.tmpl",
	"internal/jobhandlers/notifications_test.go":   "templates/notifications_jobhandler_test.go.tmpl",
	"internal/notifications/notifications.go":      "templates/notifications.go.tmpl",
	"internal/notifications/notifications_test.go": "templates/notifications_test.go.tmpl",
	"internal/notifications/smtp/sender.go":        "templates/notifications_smtp.go.tmpl",
	"internal/notifications/smtp/sender_test.go":   "templates/notifications_smtp_test.go.tmpl",
}

var filesTemplates = map[string]string{
	"internal/files/files.go":                "templates/files.go.tmpl",
	"internal/files/files_test.go":           "templates/files_test.go.tmpl",
	"internal/files/httpapi/handler.go":      "templates/files_handler.go.tmpl",
	"internal/files/httpapi/handler_test.go": "templates/files_handler_test.go.tmpl",
	"internal/files/local/store.go":          "templates/files_local_store.go.tmpl",
	"internal/files/local/store_test.go":     "templates/files_local_store_test.go.tmpl",
}

var cacheTemplates = map[string]string{
	"internal/cache/cache.go":      "templates/cache.go.tmpl",
	"internal/cache/cache_test.go": "templates/cache_test.go.tmpl",
}

var jobAdminTemplates = map[string]string{
	"internal/jobadmin/service.go":              "templates/jobadmin.go.tmpl",
	"internal/jobadmin/service_test.go":         "templates/jobadmin_test.go.tmpl",
	"internal/jobadmin/httpapi/handler.go":      "templates/jobadmin_handler.go.tmpl",
	"internal/jobadmin/httpapi/handler_test.go": "templates/jobadmin_handler_test.go.tmpl",
}

var observabilityTemplates = map[string]string{
	"internal/platform/observability/observability.go":      "templates/observability.go.tmpl",
	"internal/platform/observability/observability_test.go": "templates/observability_test.go.tmpl",
}

var csvTransferTemplates = []businessTemplate{
	{PathSuffix: "transfer/service.go", TemplatePath: "templates/csv_transfer.go.tmpl"},
	{PathSuffix: "transfer/service_test.go", TemplatePath: "templates/csv_transfer_test.go.tmpl"},
	{PathSuffix: "transfer/httpapi/handler.go", TemplatePath: "templates/csv_transfer_handler.go.tmpl"},
	{PathSuffix: "transfer/httpapi/handler_test.go", TemplatePath: "templates/csv_transfer_handler_test.go.tmpl"},
}

var approvalsTemplates = map[string]string{
	"internal/approvals/service.go":              "templates/approvals.go.tmpl",
	"internal/approvals/service_test.go":         "templates/approvals_test.go.tmpl",
	"internal/approvals/httpapi/handler.go":      "templates/approvals_handler.go.tmpl",
	"internal/approvals/httpapi/handler_test.go": "templates/approvals_handler_test.go.tmpl",
}

var catalogTemplates = map[string]string{
	"internal/catalog/catalog.go":              "templates/catalog.go.tmpl",
	"internal/catalog/catalog_test.go":         "templates/catalog_test.go.tmpl",
	"internal/catalog/httpapi/handler.go":      "templates/catalog_handler.go.tmpl",
	"internal/catalog/httpapi/handler_test.go": "templates/catalog_handler_test.go.tmpl",
}

var customerTemplates = map[string]string{
	"internal/customeraccounts/accounts.go":             "templates/customer_accounts.go.tmpl",
	"internal/customeraccounts/accounts_test.go":        "templates/customer_accounts_test.go.tmpl",
	"internal/customeraccounts/httpapi/handler.go":      "templates/customer_accounts_handler.go.tmpl",
	"internal/customeraccounts/httpapi/handler_test.go": "templates/customer_accounts_handler_test.go.tmpl",
}

// Adapter generates the Go base service.
type Adapter struct{}

// New returns the first-party Go generator.
func New() Adapter {
	return Adapter{}
}

// Backend identifies this adapter.
func (Adapter) Backend() string {
	return backend
}

// Generate renders a deterministic service with no runtime Engine dependency.
func (Adapter) Generate(ctx context.Context, project spec.Project) (generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return generator.Result{}, err
	}
	if project.Spec.Stack.Backend != backend {
		return generator.Result{}, fmt.Errorf("backend must be %q for the Go adapter", backend)
	}
	database, supported := databaseTemplates[project.Spec.Database.Engine]
	if !supported {
		return generator.Result{}, fmt.Errorf("the Go adapter supports only PostgreSQL and MySQL")
	}
	storefrontEnabled := enabled(project.Spec.Stack.Storefront)
	if storefrontEnabled && project.Spec.Stack.Storefront != "nuxt" {
		return generator.Result{}, fmt.Errorf("the Go adapter supports only the Nuxt storefront")
	}
	adminEnabled := enabled(project.Spec.Stack.AdminUI)
	if adminEnabled && project.Spec.Stack.AdminUI != "element-plus" {
		return generator.Result{}, fmt.Errorf("the Go adapter supports only the Element Plus administration UI")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Go adapter currently requires both session and token authentication")
	}
	resolvedCapabilities, diagnostics := capability.Resolve(goCapabilityCatalog, project.Spec.Capabilities)
	if len(diagnostics) > 0 {
		return generator.Result{}, fmt.Errorf("resolve Go capabilities: %s", diagnostics[0].Message)
	}
	tenancyEnabled := false
	tenancyMembersEnabled := false
	tenancyLifecycleEnabled := false
	jobsEnabled := false
	notificationsEnabled := false
	filesEnabled := false
	cacheEnabled := false
	jobAdminEnabled := false
	observabilityEnabled := false
	csvTransferEnabled := false
	approvalsEnabled := false
	catalogEnabled := false
	customerEnabled := false
	for _, selection := range project.Spec.Capabilities {
		if len(selection.Config) > 0 {
			return generator.Result{}, fmt.Errorf("Go capability %q does not accept configuration in this version", selection.Name)
		}
	}
	for _, pack := range resolvedCapabilities {
		if pack.Metadata.Name == tenancyCapability {
			tenancyEnabled = true
			tenancyMembersEnabled = pack.Metadata.Version == tenancyMembersVersion || pack.Metadata.Version == tenancyLifecycleVersion
			tenancyLifecycleEnabled = pack.Metadata.Version == tenancyLifecycleVersion
		}
		if pack.Metadata.Name == jobsCapability {
			jobsEnabled = true
		}
		if pack.Metadata.Name == notificationsCapability {
			notificationsEnabled = true
		}
		if pack.Metadata.Name == filesCapability {
			filesEnabled = true
		}
		if pack.Metadata.Name == cacheCapability {
			cacheEnabled = true
		}
		if pack.Metadata.Name == jobAdminCapability {
			jobAdminEnabled = true
		}
		if pack.Metadata.Name == observabilityCapability {
			observabilityEnabled = true
		}
		if pack.Metadata.Name == csvTransferCapability {
			csvTransferEnabled = true
		}
		if pack.Metadata.Name == approvalsCapability {
			approvalsEnabled = true
		}
		if pack.Metadata.Name == catalogCapability {
			catalogEnabled = true
		}
		if pack.Metadata.Name == customerCapability {
			customerEnabled = true
		}
	}
	if catalogEnabled && tenancyEnabled && !tenancyLifecycleEnabled {
		return generator.Result{}, errors.New("commerce-catalog with organization-tenancy requires organization-tenancy 0.3.0")
	}
	if customerEnabled && tenancyEnabled && !tenancyLifecycleEnabled {
		return generator.Result{}, errors.New("customer-accounts with organization-tenancy requires organization-tenancy 0.3.0")
	}
	business, err := buildBusinessData(project.Spec.Modules, database.Data.Engine, approvalsEnabled)
	if err != nil {
		return generator.Result{}, err
	}
	if csvTransferEnabled && business == nil {
		return generator.Result{}, errors.New("csv-import-export requires one generated business entity")
	}
	if approvalsEnabled && business == nil {
		return generator.Result{}, errors.New("approval-workflows requires one generated business entity")
	}
	data := templateData{
		ProjectName:      project.Metadata.Name,
		ModulePath:       "example.com/" + project.Metadata.Name,
		Database:         database.Data,
		Business:         business,
		Admin:            adminEnabled,
		Tenancy:          tenancyEnabled,
		TenancyMembers:   tenancyMembersEnabled,
		TenancyLifecycle: tenancyLifecycleEnabled,
		Jobs:             jobsEnabled,
		Notifications:    notificationsEnabled,
		Files:            filesEnabled,
		Cache:            cacheEnabled,
		JobAdmin:         jobAdminEnabled,
		Observability:    observabilityEnabled,
		CSVTransfer:      csvTransferEnabled,
		Approvals:        approvalsEnabled,
		Catalog:          catalogEnabled,
		CustomerAccounts: customerEnabled,
	}
	data.MigrationCount = 1
	if business != nil {
		data.MigrationCount++
	}
	if tenancyEnabled {
		data.MigrationCount++
	}
	if tenancyMembersEnabled {
		data.MigrationCount++
	}
	if tenancyLifecycleEnabled {
		data.MigrationCount++
	}
	if jobsEnabled {
		data.MigrationCount++
	}
	if filesEnabled {
		data.MigrationCount++
	}
	if cacheEnabled {
		data.MigrationCount++
	}
	if jobAdminEnabled {
		data.MigrationCount++
	}
	if csvTransferEnabled {
		data.MigrationCount++
	}
	if approvalsEnabled {
		data.MigrationCount++
	}
	if catalogEnabled {
		data.MigrationCount++
	}
	if customerEnabled {
		data.MigrationCount++
	}
	targets := make(map[string]renderTarget, len(outputTemplates)+len(businessTemplates)+len(storefrontui.BaseTemplates)+1)
	for path, templatePath := range outputTemplates {
		targets[path] = renderTarget{TemplatePath: templatePath, Owner: baseCapability}
	}
	for path, templatePath := range database.Outputs {
		targets[path] = renderTarget{TemplatePath: templatePath, Owner: baseCapability}
	}
	capabilityLock := map[string]string{baseCapability: baseVersion}
	for _, pack := range resolvedCapabilities {
		capabilityLock[pack.Metadata.Name] = pack.Metadata.Version
	}
	if business != nil {
		for _, target := range businessTemplates {
			path := "internal/" + business.ModuleName + "/" + target.PathSuffix
			targets[path] = renderTarget{TemplatePath: target.TemplatePath, Owner: businessCapability}
		}
		storePath := "internal/" + business.ModuleName + "/" + database.BusinessStorePath
		targets[storePath] = renderTarget{TemplatePath: database.BusinessStoreTemplate, Owner: businessCapability}
		targets[database.IntegrationPath] = renderTarget{
			TemplatePath: database.IntegrationTemplate,
			Owner:        businessCapability,
		}
		migrationPath := "internal/platform/migrate/migrations/000100_" + business.ModuleName + "_" + business.EntityName + ".sql"
		targets[migrationPath] = renderTarget{TemplatePath: database.BusinessMigrationTemplate, Owner: businessCapability}
		capabilityLock[businessCapability] = businessVersion
	}
	if adminEnabled {
		for path, templatePath := range adminui.BaseTemplates {
			targets[path] = renderTarget{
				TemplatePath: templatePath, Owner: adminCapability, SharedAdmin: true,
			}
		}
		if business != nil {
			path := "web/admin/src/views/BusinessView.vue"
			targets[path] = renderTarget{
				TemplatePath: adminui.BusinessViewTemplate,
				Owner:        adminCapability,
				SharedAdmin:  true,
			}
		}
		capabilityLock[adminCapability] = adminVersion
	}
	if storefrontEnabled {
		for path, templatePath := range storefrontui.BaseTemplates {
			targets[path] = renderTarget{
				TemplatePath:     templatePath,
				Owner:            storefrontCapability,
				SharedStorefront: true,
			}
		}
		capabilityLock[storefrontCapability] = storefrontVersion
	}
	if tenancyEnabled {
		for path, templatePath := range tenancyTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: tenancyCapability}
		}
		targets[database.TenancyStorePath] = renderTarget{TemplatePath: database.TenancyStoreTemplate, Owner: tenancyCapability}
		tenancyMigrationPath := "internal/platform/migrate/migrations/000050_organization_tenancy.sql"
		targets[tenancyMigrationPath] = renderTarget{TemplatePath: database.TenancyMigrationTemplate, Owner: tenancyCapability}
	}
	if tenancyMembersEnabled {
		for path, templatePath := range tenancyMembersTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: tenancyCapability}
		}
		targets[database.TenancyMembersStorePath] = renderTarget{TemplatePath: database.TenancyMembersStoreTemplate, Owner: tenancyCapability}
		targets["internal/platform/migrate/migrations/000060_organization_members.sql"] = renderTarget{
			TemplatePath: database.TenancyMembersMigrationTemplate,
			Owner:        tenancyCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/MembersView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/MembersView.vue",
				Owner:        tenancyCapability,
				SharedAdmin:  true,
			}
		}
	}
	if tenancyLifecycleEnabled {
		for path, templatePath := range tenancyLifecycleTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: tenancyCapability}
		}
		targets[database.TenancyLifecycleStorePath] = renderTarget{TemplatePath: database.TenancyLifecycleStoreTemplate, Owner: tenancyCapability}
		targets["internal/platform/migrate/migrations/000070_organization_lifecycle.sql"] = renderTarget{
			TemplatePath: database.TenancyLifecycleMigrationTemplate,
			Owner:        tenancyCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/OrganizationSettingsView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/OrganizationSettingsView.vue",
				Owner:        tenancyCapability,
				SharedAdmin:  true,
			}
		}
	}
	if jobsEnabled {
		for path, templatePath := range jobsTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: jobsCapability}
		}
		targets[database.JobsStorePath] = renderTarget{TemplatePath: database.JobsStoreTemplate, Owner: jobsCapability}
		targets["internal/platform/migrate/migrations/000200_background_jobs.sql"] = renderTarget{
			TemplatePath: database.JobsMigrationTemplate,
			Owner:        jobsCapability,
		}
	}
	if notificationsEnabled {
		for path, templatePath := range notificationsTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: notificationsCapability}
		}
	}
	if filesEnabled {
		for path, templatePath := range filesTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: filesCapability}
		}
		targets[database.FilesStorePath] = renderTarget{TemplatePath: database.FilesStoreTemplate, Owner: filesCapability}
		targets["internal/platform/migrate/migrations/000210_file_assets.sql"] = renderTarget{
			TemplatePath: database.FilesMigrationTemplate,
			Owner:        filesCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/FilesView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/FilesView.vue",
				Owner:        filesCapability,
				SharedAdmin:  true,
			}
		}
	}
	if cacheEnabled {
		for path, templatePath := range cacheTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: cacheCapability}
		}
		targets[database.CacheStorePath] = renderTarget{TemplatePath: database.CacheStoreTemplate, Owner: cacheCapability}
		targets["internal/platform/migrate/migrations/000220_application_cache.sql"] = renderTarget{
			TemplatePath: database.CacheMigrationTemplate,
			Owner:        cacheCapability,
		}
	}
	if jobAdminEnabled {
		for path, templatePath := range jobAdminTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: jobAdminCapability}
		}
		targets[database.JobAdminStorePath] = renderTarget{TemplatePath: database.JobAdminStoreTemplate, Owner: jobAdminCapability}
		targets["internal/platform/migrate/migrations/000230_job_administration.sql"] = renderTarget{
			TemplatePath: database.JobAdminMigrationTemplate,
			Owner:        jobAdminCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/JobsView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/JobsView.vue",
				Owner:        jobAdminCapability,
				SharedAdmin:  true,
			}
		}
	}
	if observabilityEnabled {
		for path, templatePath := range observabilityTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: observabilityCapability}
		}
	}
	if csvTransferEnabled {
		for _, target := range csvTransferTemplates {
			path := "internal/" + business.ModuleName + "/" + target.PathSuffix
			targets[path] = renderTarget{TemplatePath: target.TemplatePath, Owner: csvTransferCapability}
		}
		storePath := "internal/" + business.ModuleName + "/" + database.CSVTransferStorePath
		targets[storePath] = renderTarget{TemplatePath: database.CSVTransferStoreTemplate, Owner: csvTransferCapability}
		targets["internal/platform/migrate/migrations/000240_csv_import_export.sql"] = renderTarget{
			TemplatePath: database.CSVTransferMigrationTemplate,
			Owner:        csvTransferCapability,
		}
	}
	if approvalsEnabled {
		for path, templatePath := range approvalsTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: approvalsCapability}
		}
		targets[database.ApprovalsStorePath] = renderTarget{TemplatePath: database.ApprovalsStoreTemplate, Owner: approvalsCapability}
		targets["internal/platform/migrate/migrations/000250_approval_workflows.sql"] = renderTarget{
			TemplatePath: database.ApprovalsMigrationTemplate,
			Owner:        approvalsCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/ApprovalsView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/ApprovalsView.vue.tmpl",
				Owner:        approvalsCapability,
				SharedAdmin:  true,
			}
		}
	}
	if catalogEnabled {
		for path, templatePath := range catalogTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: catalogCapability}
		}
		targets[database.CatalogStorePath] = renderTarget{
			TemplatePath: database.CatalogStoreTemplate,
			Owner:        catalogCapability,
		}
		targets["internal/platform/migrate/migrations/000260_commerce_catalog.sql"] = renderTarget{
			TemplatePath: database.CatalogMigrationTemplate,
			Owner:        catalogCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/CatalogView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/CatalogView.vue.tmpl",
				Owner:        catalogCapability,
				SharedAdmin:  true,
			}
		}
		if storefrontEnabled {
			for path, templatePath := range storefrontui.CatalogTemplates {
				targets[path] = renderTarget{
					TemplatePath: templatePath, Owner: catalogCapability, SharedStorefront: true,
				}
			}
		}
	}
	if customerEnabled {
		for path, templatePath := range customerTemplates {
			targets[path] = renderTarget{TemplatePath: templatePath, Owner: customerCapability}
		}
		targets[database.CustomerStorePath] = renderTarget{
			TemplatePath: database.CustomerStoreTemplate,
			Owner:        customerCapability,
		}
		targets["internal/platform/migrate/migrations/000270_customer_accounts.sql"] = renderTarget{
			TemplatePath: database.CustomerMigrationTemplate,
			Owner:        customerCapability,
		}
		if adminEnabled {
			targets["web/admin/src/views/CustomerAccountsView.vue"] = renderTarget{
				TemplatePath: "templates/src/views/CustomerAccountsView.vue.tmpl",
				Owner:        customerCapability,
				SharedAdmin:  true,
			}
		}
		if storefrontEnabled {
			for path, templatePath := range storefrontui.CustomerAccountTemplates {
				targets[path] = renderTarget{
					TemplatePath: templatePath, Owner: customerCapability, SharedStorefront: true,
				}
			}
		}
	}
	paths := make([]string, 0, len(targets))
	for path := range targets {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	outputs := make([]change.Output, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return generator.Result{}, err
		}
		target := targets[path]
		var content []byte
		var err error
		if target.SharedAdmin {
			content, err = adminui.Render(target.TemplatePath, data)
		} else if target.SharedStorefront {
			storefrontData := storefrontui.NewData(project.Metadata.Name, project.Metadata.DisplayName)
			storefrontData.Catalog = catalogEnabled
			storefrontData.CustomerAccounts = customerEnabled
			storefrontData.Tenancy = tenancyEnabled
			content, err = storefrontui.Render(
				target.TemplatePath,
				storefrontData,
			)
		} else {
			content, err = render(target.TemplatePath, data)
		}
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		if strings.HasSuffix(path, ".go") {
			content, err = format.Source(content)
			if err != nil {
				return generator.Result{}, fmt.Errorf("format generated %q: %w", path, err)
			}
		}
		outputs = append(outputs, change.Output{Path: path, Owner: target.Owner, Content: content})
	}
	return generator.Result{
		CapabilityLock: capabilityLock,
		Outputs:        outputs,
	}, nil
}

func enabled(value string) bool {
	return value != "" && value != "none"
}

func hasExactAuthModes(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	want := make(map[string]struct{}, len(expected))
	for _, mode := range expected {
		want[mode] = struct{}{}
	}
	for _, mode := range actual {
		if _, exists := want[mode]; !exists {
			return false
		}
		delete(want, mode)
	}
	return len(want) == 0
}

type templateData struct {
	ProjectName      string
	ModulePath       string
	Database         databaseData
	Business         *businessData
	Admin            bool
	Tenancy          bool
	TenancyMembers   bool
	TenancyLifecycle bool
	Jobs             bool
	Notifications    bool
	Files            bool
	Cache            bool
	JobAdmin         bool
	Observability    bool
	CSVTransfer      bool
	Approvals        bool
	Catalog          bool
	CustomerAccounts bool
	MigrationCount   int
}

type databaseData struct {
	Engine      string
	DisplayName string
	PackageName string
}

type renderTarget struct {
	TemplatePath     string
	Owner            string
	SharedAdmin      bool
	SharedStorefront bool
}

type businessData struct {
	ModuleName       string
	PackageName      string
	EntityName       string
	EntityType       string
	TableName        string
	RoutePath        string
	PermissionPrefix string
	Fields           []businessField
	RequiredFields   []string
}

type businessField struct {
	Name              string
	GoName            string
	GoType            string
	EntityType        string
	SQLType           string
	Required          bool
	Unique            bool
	StringLike        bool
	SampleOne         string
	SampleTwo         string
	OpenAPIType       string
	OpenAPIFormat     string
	TypeScriptType    string
	TypeScriptDefault string
	InputKind         string
	JSONOptions       string
	OpenAPIPattern    string
}

type supportedFieldType struct {
	GoType            string
	SQLType           string
	StringLike        bool
	SampleOne         string
	SampleTwo         string
	OpenAPIType       string
	OpenAPIFormat     string
	OpenAPIPattern    string
	TypeScriptType    string
	TypeScriptDefault string
	InputKind         string
	JSONOptions       string
}

func buildBusinessData(modules []spec.Module, databaseEngine string, approvalsEnabled bool) (*businessData, error) {
	if len(modules) == 0 {
		return nil, nil
	}
	if len(modules) != 1 {
		return nil, fmt.Errorf("the first Go CRUD slice supports exactly one business module")
	}
	module := modules[0]
	if strings.Contains(module.Name, "-") {
		return nil, fmt.Errorf("Go business module names must use lowercase letters and digits without hyphens")
	}
	if _, reserved := goKeywords[module.Name]; reserved {
		return nil, fmt.Errorf("Go business module name %q is a language keyword", module.Name)
	}
	if len(module.Entities) != 1 {
		return nil, fmt.Errorf("the first Go CRUD slice supports exactly one entity")
	}
	if len(module.Pages) > 0 {
		return nil, fmt.Errorf("Go generated pages are not available in the first CRUD slice")
	}
	if approvalsEnabled {
		if len(module.Workflows) != 1 || module.Workflows[0].Name != "approval" ||
			!slices.Equal(module.Workflows[0].States, []string{"pending", "approved", "rejected", "cancelled"}) {
			return nil, fmt.Errorf("approval-workflows requires workflow approval with states pending, approved, rejected, cancelled in that order")
		}
	} else if len(module.Workflows) > 0 {
		return nil, fmt.Errorf("Go workflows require a selected workflow capability")
	}
	entity := module.Entities[0]
	prefix := module.Name + ":" + entity.Name
	wantPermissions := map[string]struct{}{
		prefix + ":create": {},
		prefix + ":read":   {},
		prefix + ":update": {},
		prefix + ":delete": {},
	}
	if len(module.Permissions) != len(wantPermissions) {
		return nil, fmt.Errorf("business module permissions must declare create, read, update, and delete codes for its entity")
	}
	for _, permission := range module.Permissions {
		if _, exists := wantPermissions[permission.Code]; !exists {
			return nil, fmt.Errorf("unsupported business permission %q", permission.Code)
		}
		delete(wantPermissions, permission.Code)
	}
	fields := make([]businessField, 0, len(entity.Fields))
	requiredFields := make([]string, 0, len(entity.Fields))
	reserved := map[string]struct{}{"id": {}, "version": {}, "created_at": {}, "updated_at": {}}
	for _, field := range entity.Fields {
		if _, exists := reserved[field.Name]; exists {
			return nil, fmt.Errorf("business field %q is reserved", field.Name)
		}
		if databaseEngine == "mysql" && field.Type == "text" && field.Unique {
			return nil, fmt.Errorf("MySQL text field %q cannot use the portable unique constraint", field.Name)
		}
		fieldType, ok := businessFieldType(field.Type)
		if !ok {
			return nil, fmt.Errorf("business field %q uses unsupported Go CRUD type %q", field.Name, field.Type)
		}
		if field.Type == "datetime" && databaseEngine == "mysql" {
			fieldType.SQLType = "datetime(6)"
		}
		entityType := fieldType.GoType
		if !field.Required {
			entityType = "*" + fieldType.GoType
		}
		fields = append(fields, businessField{
			Name: field.Name, GoName: exportedName(field.Name), GoType: fieldType.GoType,
			EntityType: entityType, SQLType: fieldType.SQLType, Required: field.Required,
			Unique: field.Unique, StringLike: fieldType.StringLike,
			SampleOne: fieldType.SampleOne, SampleTwo: fieldType.SampleTwo,
			OpenAPIType: fieldType.OpenAPIType, OpenAPIFormat: fieldType.OpenAPIFormat,
			OpenAPIPattern: fieldType.OpenAPIPattern, JSONOptions: fieldType.JSONOptions,
			TypeScriptType:    fieldType.TypeScriptType,
			TypeScriptDefault: fieldType.TypeScriptDefault, InputKind: fieldType.InputKind,
		})
		if field.Required {
			requiredFields = append(requiredFields, field.Name)
		}
	}
	return &businessData{
		ModuleName: module.Name, PackageName: module.Name, EntityName: entity.Name,
		EntityType: exportedName(entity.Name), TableName: module.Name + "_" + entity.Name,
		RoutePath: "/api/v1/" + module.Name, PermissionPrefix: prefix,
		Fields: fields, RequiredFields: requiredFields,
	}, nil
}

func businessFieldType(value string) (supportedFieldType, bool) {
	switch value {
	case "string":
		return supportedFieldType{GoType: "string", SQLType: "varchar(255)", StringLike: true, SampleOne: `"first"`, SampleTwo: `"second"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "text"}, true
	case "text":
		return supportedFieldType{GoType: "string", SQLType: "text", StringLike: true, SampleOne: `"first text"`, SampleTwo: `"second text"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "textarea"}, true
	case "bool":
		return supportedFieldType{GoType: "bool", SQLType: "boolean", SampleOne: "true", SampleTwo: "false", OpenAPIType: "boolean", TypeScriptType: "boolean", TypeScriptDefault: "false", InputKind: "boolean"}, true
	case "int64":
		return supportedFieldType{GoType: "int64", SQLType: "bigint", SampleOne: "int64(1)", SampleTwo: "int64(2)", OpenAPIType: "string", OpenAPIPattern: `"^-?[0-9]+$"`, TypeScriptType: "string", TypeScriptDefault: `"0"`, InputKind: "text", JSONOptions: ",string"}, true
	case "datetime":
		return supportedFieldType{GoType: "time.Time", SQLType: "timestamptz", SampleOne: "time.Unix(1_700_000_000, 0).UTC()", SampleTwo: "time.Unix(1_700_000_100, 0).UTC()", OpenAPIType: "string", OpenAPIFormat: "date-time", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "datetime"}, true
	default:
		return supportedFieldType{}, false
	}
}

func exportedName(value string) string {
	parts := strings.Split(value, "_")
	for index, part := range parts {
		if part != "" {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

var goKeywords = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
	"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {}, "chan": {},
	"else": {}, "goto": {}, "package": {}, "switch": {}, "const": {}, "fallthrough": {},
	"if": {}, "range": {}, "type": {}, "continue": {}, "for": {}, "import": {},
	"return": {}, "var": {},
}

func render(templatePath string, data templateData) ([]byte, error) {
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	functions := template.FuncMap{
		"sql":   func(value string) string { return quoteSQLIdentifier(data.Database.Engine, value) },
		"goSQL": func(value string) string { return quoteGoSQLIdentifier(data.Database.Engine, value) },
		"list":  func(values ...string) []string { return values },
	}
	parsed, err := template.New(templatePath).Funcs(functions).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func quoteSQLIdentifier(databaseEngine, value string) string {
	if databaseEngine == "mysql" {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteGoSQLIdentifier(databaseEngine, value string) string {
	if databaseEngine == "mysql" {
		escaped := strings.ReplaceAll(value, "`", "``")
		return "`+\"`" + escaped + "`\"+`"
	}
	return quoteSQLIdentifier(databaseEngine, value)
}

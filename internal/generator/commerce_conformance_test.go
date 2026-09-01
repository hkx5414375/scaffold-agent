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
)

type commerceOperation struct {
	OperationID string
	Permission  string
}

type commerceShape struct {
	OrderStatuses    []string
	PaymentStatuses  []string
	CampaignKinds    []string
	CampaignStatuses []string
	MoneyType        string
	MoneyPattern     string
	QuantityType     string
	QuantityPattern  string
	VersionType      string
	VersionPattern   string
}

func TestCommerceOperationsConformAcrossBackends(t *testing.T) {
	t.Parallel()

	adapters := map[string]generator.Adapter{
		"go":     golangadapter.New(),
		"java":   javaadapter.New(),
		"python": pythonadapter.New(),
	}
	wantOperations := map[string]commerceOperation{
		"GET /api/v1/storefront/cart":                                      {"getCommerceCart", ""},
		"PUT /api/v1/storefront/cart/lines/{product_id}":                   {"putCommerceCartLine", ""},
		"POST /api/v1/storefront/cart/lines/{product_id}/remove":           {"removeCommerceCartLine", ""},
		"POST /api/v1/storefront/checkout":                                 {"checkoutCommerceCart", ""},
		"GET /api/v1/storefront/orders":                                    {"listCustomerCommerceOrders", ""},
		"GET /api/v1/storefront/orders/{id}":                               {"getCustomerCommerceOrder", ""},
		"POST /api/v1/storefront/orders/{id}/return":                       {"requestCommerceReturn", ""},
		"POST /api/v1/storefront/sandbox/payments/{provider_ref}/complete": {"completeSandboxCommercePayment", ""},
		"GET /api/v1/commerce/orders":                                      {"listCommerceOrders", "commerce:orders:read"},
		"POST /api/v1/commerce/orders/{id}/fulfillment/start":              {"startCommerceFulfillment", "commerce:fulfillment:manage"},
		"POST /api/v1/commerce/orders/{id}/fulfillment/complete":           {"completeCommerceFulfillment", "commerce:fulfillment:manage"},
		"POST /api/v1/commerce/orders/{id}/return/complete":                {"completeCommerceReturn", "commerce:fulfillment:manage"},
		"POST /api/v1/commerce/orders/{id}/refund":                         {"refundCommercePayment", "commerce:payments:manage"},
		"POST /api/v1/commerce/orders/{id}/reconcile":                      {"reconcileCommercePayment", "commerce:payments:manage"},
		"GET /api/v1/commerce/campaigns":                                   {"listCommerceCampaigns", "commerce:marketing:manage"},
		"POST /api/v1/commerce/campaigns":                                  {"createCommerceCampaign", "commerce:marketing:manage"},
		"POST /api/v1/commerce/campaigns/{id}/activate":                    {"activateCommerceCampaign", "commerce:marketing:manage"},
		"POST /api/v1/commerce/campaigns/{id}/archive":                     {"archiveCommerceCampaign", "commerce:marketing:manage"},
		"POST /api/v1/commerce/coupons":                                    {"createCommerceCoupon", "commerce:marketing:manage"},
	}
	wantShape := commerceShape{
		OrderStatuses:    []string{"pending_payment", "confirmed", "fulfilling", "fulfilled", "return_requested", "returned", "cancelled"},
		PaymentStatuses:  []string{"requires_action", "succeeded", "failed"},
		CampaignKinds:    []string{"fixed", "percent"},
		CampaignStatuses: []string{"draft", "active", "archived"},
		MoneyType:        "string",
		MoneyPattern:     "^(0|[1-9][0-9]{0,18})$",
		QuantityType:     "string",
		QuantityPattern:  "^[1-9][0-9]{0,2}$",
		VersionType:      "string",
		VersionPattern:   "^[1-9][0-9]*$",
	}
	wantSchemas := []string{
		"CommerceCartLine", "CommerceCart", "CommerceLineWrite", "CommerceCartTransition",
		"CommerceCheckout", "CommerceOrderLine", "CommercePaymentIntent", "CommerceOrder",
		"CommerceOrderPage", "CommerceReturnWrite", "CommercePaymentEvent", "CommerceTransition",
		"CommerceRefundWrite", "CommerceCampaign", "CommerceCampaignWrite", "CommerceCampaignPage",
		"CommerceCoupon", "CommerceCouponWrite",
	}
	wantMigrationFragments := []string{
		"commerce_carts", "commerce_cart_lines", "commerce_campaigns", "commerce_coupons",
		"commerce_orders", "commerce_order_lines", "commerce_checkout_idempotency",
		"commerce_payment_intents", "commerce_payment_events", "commerce_refunds",
		"commerce_order_events",
		"commerce:orders:read", "commerce:fulfillment:manage", "commerce:marketing:manage",
		"commerce:payments:manage",
	}
	migrationPaths := map[string]string{
		"go":     "internal/platform/migrate/migrations/000300_commerce_operations.sql",
		"java":   "src/main/resources/db/migration/V000300__commerce_operations.sql",
		"python": "src/commerce_conformance/migration/versions/000300_commerce.py",
	}

	var sharedAdmin []byte
	sharedStorefront := make(map[string][]byte)
	for backend, adapter := range adapters {
		backend := backend
		adapter := adapter
		t.Run(backend, func(t *testing.T) {
			generated, err := adapter.Generate(context.Background(), commerceProject(backend))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			for name, version := range map[string]string{
				"organization-tenancy": "0.3.0",
				"commerce-catalog":     "0.1.0",
				"customer-accounts":    "0.1.0",
				"commerce-operations":  "0.1.0",
			} {
				if generated.CapabilityLock[name] != version {
					t.Errorf("capability %s = %q, want %q", name, generated.CapabilityLock[name], version)
				}
			}

			contract := decodeOpenAPI(t, output(t, generated, "api/openapi.yaml"))
			if got := commerceOperations(t, contract); !reflect.DeepEqual(got, wantOperations) {
				t.Fatalf("commerce operations = %#v, want %#v", got, wantOperations)
			}
			if got := commerceOpenAPIShape(t, contract); !reflect.DeepEqual(got, wantShape) {
				t.Fatalf("commerce shape = %#v, want %#v", got, wantShape)
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
					t.Errorf("commerce migration does not contain %q", fragment)
				}
			}
			admin := output(t, generated, "web/admin/src/views/CommerceView.vue")
			if sharedAdmin == nil {
				sharedAdmin = admin
			} else if !reflect.DeepEqual(admin, sharedAdmin) {
				t.Error("shared commerce administration view differs across backends")
			}
			for _, path := range []string{
				"web/storefront/app/pages/cart.vue",
				"web/storefront/app/pages/checkout.vue",
				"web/storefront/app/pages/account/orders/index.vue",
				"web/storefront/app/pages/account/orders/[id].vue",
				"web/storefront/server/utils/commerce.ts",
				"web/storefront/shared/types/commerce.ts",
				"web/storefront/test/commerce.test.ts",
			} {
				content := output(t, generated, path)
				if reference, exists := sharedStorefront[path]; !exists {
					sharedStorefront[path] = content
				} else if !reflect.DeepEqual(content, reference) {
					t.Errorf("shared commerce storefront %s differs across backends", path)
				}
			}
		})
	}
}

func commerceProject(backend string) spec.Project {
	return spec.Project{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindProject,
		Metadata: spec.Metadata{
			Name:        "commerce-conformance",
			DisplayName: "Commerce Conformance",
		},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: backend, AdminUI: "element-plus", Storefront: "nuxt"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
			Capabilities: []spec.CapabilitySelection{
				{Name: "organization-tenancy", Version: "0.3.0"},
				{Name: "commerce-operations", Version: "0.1.0"},
			},
		},
	}
}

func commerceOperations(t *testing.T, contract map[string]any) map[string]commerceOperation {
	t.Helper()
	result := make(map[string]commerceOperation)
	for path, rawPath := range mapping(t, contract, "paths") {
		if !strings.HasPrefix(path, "/api/v1/commerce/") &&
			path != "/api/v1/storefront/cart" &&
			!strings.HasPrefix(path, "/api/v1/storefront/cart/") &&
			path != "/api/v1/storefront/checkout" &&
			!strings.HasPrefix(path, "/api/v1/storefront/orders") &&
			!strings.HasPrefix(path, "/api/v1/storefront/sandbox/payments/") {
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
			result[strings.ToUpper(method)+" "+path] = commerceOperation{
				OperationID: scalarString(t, operation, "operationId"),
				Permission:  optionalScalarString(t, operation, "x-required-permission"),
			}
		}
	}
	return result
}

func commerceOpenAPIShape(t *testing.T, contract map[string]any) commerceShape {
	t.Helper()
	schemas := mapping(t, mapping(t, contract, "components"), "schemas")
	order := properties(t, schemas, "CommerceOrder")
	payment := properties(t, schemas, "CommercePaymentIntent")
	campaign := properties(t, schemas, "CommerceCampaign")
	line := properties(t, schemas, "CommerceCartLine")
	money := mapping(t, order, "total_minor")
	quantity := mapping(t, line, "quantity")
	version := mapping(t, order, "version")
	return commerceShape{
		OrderStatuses:    stringSlice(t, mapping(t, order, "status"), "enum"),
		PaymentStatuses:  stringSlice(t, mapping(t, payment, "status"), "enum"),
		CampaignKinds:    stringSlice(t, mapping(t, campaign, "kind"), "enum"),
		CampaignStatuses: stringSlice(t, mapping(t, campaign, "status"), "enum"),
		MoneyType:        scalarString(t, money, "type"),
		MoneyPattern:     scalarString(t, money, "pattern"),
		QuantityType:     scalarString(t, quantity, "type"),
		QuantityPattern:  scalarString(t, quantity, "pattern"),
		VersionType:      scalarString(t, version, "type"),
		VersionPattern:   scalarString(t, version, "pattern"),
	}
}

func optionalScalarString(t *testing.T, source map[string]any, key string) string {
	t.Helper()
	value, exists := source[key]
	if !exists {
		return ""
	}
	result, ok := value.(string)
	if !ok {
		t.Fatalf("OpenAPI key %s is not a string", key)
	}
	return result
}

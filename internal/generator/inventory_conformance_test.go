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

type inventoryOperation struct {
	OperationID string
	Permission  string
}

type inventoryShape struct {
	ItemStatuses        []string
	ReservationStatuses []string
	PurchaseStatuses    []string
	MovementKinds       []string
	QuantityType        string
	QuantityPattern     string
	DeltaPattern        string
	VersionType         string
	VersionPattern      string
}

func TestERPInventoryConformsAcrossBackends(t *testing.T) {
	t.Parallel()

	adapters := map[string]generator.Adapter{
		"go":     golangadapter.New(),
		"java":   javaadapter.New(),
		"python": pythonadapter.New(),
	}
	wantOperations := map[string]inventoryOperation{
		"POST /api/v1/inventory/items":                        {"createInventoryItem", "inventory:catalog:write"},
		"GET /api/v1/inventory/items":                         {"listInventoryItems", "inventory:read"},
		"GET /api/v1/inventory/items/{id}":                    {"getInventoryItem", "inventory:read"},
		"PUT /api/v1/inventory/items/{id}":                    {"updateInventoryItem", "inventory:catalog:write"},
		"POST /api/v1/inventory/items/{id}/archive":           {"archiveInventoryItem", "inventory:catalog:write"},
		"POST /api/v1/inventory/items/{id}/reactivate":        {"reactivateInventoryItem", "inventory:catalog:write"},
		"POST /api/v1/inventory/warehouses":                   {"createInventoryWarehouse", "inventory:catalog:write"},
		"GET /api/v1/inventory/warehouses":                    {"listInventoryWarehouses", "inventory:read"},
		"GET /api/v1/inventory/warehouses/{id}":               {"getInventoryWarehouse", "inventory:read"},
		"PUT /api/v1/inventory/warehouses/{id}":               {"updateInventoryWarehouse", "inventory:catalog:write"},
		"POST /api/v1/inventory/warehouses/{id}/archive":      {"archiveInventoryWarehouse", "inventory:catalog:write"},
		"POST /api/v1/inventory/warehouses/{id}/reactivate":   {"reactivateInventoryWarehouse", "inventory:catalog:write"},
		"GET /api/v1/inventory/balances":                      {"listInventoryBalances", "inventory:read"},
		"POST /api/v1/inventory/adjustments":                  {"adjustInventoryStock", "inventory:stock:manage"},
		"GET /api/v1/inventory/movements":                     {"listInventoryMovements", "inventory:read"},
		"POST /api/v1/inventory/reservations":                 {"createInventoryReservation", "inventory:stock:manage"},
		"GET /api/v1/inventory/reservations":                  {"listInventoryReservations", "inventory:read"},
		"GET /api/v1/inventory/reservations/{id}":             {"getInventoryReservation", "inventory:read"},
		"POST /api/v1/inventory/reservations/{id}/release":    {"releaseInventoryReservation", "inventory:stock:manage"},
		"POST /api/v1/inventory/reservations/{id}/consume":    {"consumeInventoryReservation", "inventory:stock:manage"},
		"POST /api/v1/inventory/purchase-orders":              {"createInventoryPurchaseOrder", "inventory:procurement:manage"},
		"GET /api/v1/inventory/purchase-orders":               {"listInventoryPurchaseOrders", "inventory:read"},
		"GET /api/v1/inventory/purchase-orders/{id}":          {"getInventoryPurchaseOrder", "inventory:read"},
		"POST /api/v1/inventory/purchase-orders/{id}/submit":  {"submitInventoryPurchaseOrder", "inventory:procurement:manage"},
		"POST /api/v1/inventory/purchase-orders/{id}/cancel":  {"cancelInventoryPurchaseOrder", "inventory:procurement:manage"},
		"POST /api/v1/inventory/purchase-orders/{id}/receive": {"receiveInventoryPurchaseOrder", "inventory:procurement:manage"},
	}
	wantShape := inventoryShape{
		ItemStatuses:        []string{"active", "archived"},
		ReservationStatuses: []string{"active", "released", "consumed"},
		PurchaseStatuses:    []string{"draft", "submitted", "partially_received", "received", "cancelled"},
		MovementKinds:       []string{"adjustment", "reservation_created", "reservation_released", "reservation_consumed", "purchase_received"},
		QuantityType:        "string",
		QuantityPattern:     "^(0|[1-9][0-9]*)$",
		DeltaPattern:        "^-?(0|[1-9][0-9]*)$",
		VersionType:         "string",
		VersionPattern:      "^[1-9][0-9]*$",
	}
	wantSchemas := []string{
		"InventoryItem", "InventoryItemWrite", "InventoryItemUpdate",
		"InventoryWarehouse", "InventoryWarehouseWrite", "InventoryWarehouseUpdate",
		"InventoryTransition", "InventoryBalance", "InventoryMovement", "InventoryAdjustment",
		"InventoryReservation", "InventoryReservationWrite", "InventoryReservationTransition",
		"InventoryPurchaseOrderLine", "InventoryPurchaseOrderLineWrite",
		"InventoryPurchaseOrder", "InventoryPurchaseOrderWrite", "InventoryReceipt",
		"InventoryStockResult", "InventoryReservationResult", "InventoryReceiptResult",
		"InventoryItemPage", "InventoryWarehousePage", "InventoryBalancePage",
		"InventoryMovementPage", "InventoryReservationPage", "InventoryPurchaseOrderPage",
	}
	wantMigrationFragments := []string{
		"inventory_items", "inventory_warehouses", "inventory_balances", "inventory_movements",
		"inventory_reservations", "inventory_purchase_orders", "inventory_purchase_order_lines",
		"inventory_idempotency", "reserved <= on_hand", "received_quantity <= ordered_quantity",
		"inventory:read", "inventory:catalog:write", "inventory:stock:manage",
		"inventory:procurement:manage",
	}
	migrationPaths := map[string]string{
		"go":     "internal/platform/migrate/migrations/000290_erp_inventory.sql",
		"java":   "src/main/resources/db/migration/V000290__erp_inventory.sql",
		"python": "src/inventory_conformance/migration/versions/000290_erp_inventory.py",
	}
	behaviorPaths := map[string]string{
		"go":     "internal/integration/postgres_test.go",
		"java":   "src/test/java/com/scaffold/generated/inventoryconformance/inventory/InventoryDatabaseIntegrationTest.java",
		"python": "tests/test_inventory_database.py",
	}
	behaviorFragments := map[string][]string{
		"go": {
			"AdjustStock(replay)", "CreateReservation(replay)", "ReceivePurchaseOrder(replay)",
			"AdjustStock(below zero)", "AdjustStock(replay inactive organization)",
		},
		"java":   {"adjustedReplay", "reservedReplay", "partialReplay", "invalid negative", "status = 'inactive'"},
		"python": {"adjusted_replay", "reserved_replay", "partial_replay", "invalid negative", "inactive_replay"},
	}

	var sharedView []byte
	for backend, adapter := range adapters {
		backend := backend
		adapter := adapter
		t.Run(backend, func(t *testing.T) {
			generated, err := adapter.Generate(context.Background(), inventoryProject(backend))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock["erp-inventory"] != "0.1.0" ||
				generated.CapabilityLock["organization-tenancy"] != "0.3.0" {
				t.Fatalf("capability lock = %#v", generated.CapabilityLock)
			}

			contract := decodeOpenAPI(t, output(t, generated, "api/openapi.yaml"))
			if got := inventoryOperations(t, contract); !reflect.DeepEqual(got, wantOperations) {
				t.Fatalf("inventory operations = %#v, want %#v", got, wantOperations)
			}
			if got := inventoryOpenAPIShape(t, contract); !reflect.DeepEqual(got, wantShape) {
				t.Fatalf("inventory shape = %#v, want %#v", got, wantShape)
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
					t.Errorf("inventory migration does not contain %q", fragment)
				}
			}
			behavior := string(output(t, generated, behaviorPaths[backend]))
			for _, fragment := range behaviorFragments[backend] {
				if !strings.Contains(behavior, fragment) {
					t.Errorf("inventory behavior contract does not contain %q", fragment)
				}
			}
			view := output(t, generated, "web/admin/src/views/InventoryView.vue")
			if sharedView == nil {
				sharedView = view
			} else if !reflect.DeepEqual(view, sharedView) {
				t.Error("shared inventory administration view differs across backends")
			}
		})
	}
}

func inventoryProject(backend string) spec.Project {
	return spec.Project{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindProject,
		Metadata: spec.Metadata{
			Name:        "inventory-conformance",
			DisplayName: "Inventory Conformance",
		},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: backend, AdminUI: "element-plus", Storefront: "none"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
			Modules: []spec.Module{{
				Name: "tasks",
				Entities: []spec.Entity{{
					Name: "task",
					Fields: []spec.Field{
						{Name: "title", Type: "string", Required: true, Unique: true},
						{Name: "completed", Type: "bool", Required: true},
					},
				}},
				Permissions: []spec.Permission{
					{Code: "tasks:task:create"},
					{Code: "tasks:task:read"},
					{Code: "tasks:task:update"},
					{Code: "tasks:task:delete"},
				},
			}},
			Capabilities: []spec.CapabilitySelection{
				{Name: "organization-tenancy", Version: "0.3.0"},
				{Name: "erp-inventory", Version: "0.1.0"},
			},
		},
	}
}

func inventoryOperations(t *testing.T, contract map[string]any) map[string]inventoryOperation {
	t.Helper()
	result := make(map[string]inventoryOperation)
	for path, rawPath := range mapping(t, contract, "paths") {
		if !strings.HasPrefix(path, "/api/v1/inventory/") {
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
			result[strings.ToUpper(method)+" "+path] = inventoryOperation{
				OperationID: scalarString(t, operation, "operationId"),
				Permission:  scalarString(t, operation, "x-required-permission"),
			}
		}
	}
	return result
}

func inventoryOpenAPIShape(t *testing.T, contract map[string]any) inventoryShape {
	t.Helper()
	schemas := mapping(t, mapping(t, contract, "components"), "schemas")
	item := properties(t, schemas, "InventoryItem")
	balance := properties(t, schemas, "InventoryBalance")
	movement := properties(t, schemas, "InventoryMovement")
	reservation := properties(t, schemas, "InventoryReservation")
	purchase := properties(t, schemas, "InventoryPurchaseOrder")
	quantity := mapping(t, balance, "on_hand")
	delta := mapping(t, movement, "quantity_delta")
	version := mapping(t, item, "version")
	return inventoryShape{
		ItemStatuses:        stringSlice(t, mapping(t, item, "status"), "enum"),
		ReservationStatuses: stringSlice(t, mapping(t, reservation, "status"), "enum"),
		PurchaseStatuses:    stringSlice(t, mapping(t, purchase, "status"), "enum"),
		MovementKinds:       stringSlice(t, mapping(t, movement, "kind"), "enum"),
		QuantityType:        scalarString(t, quantity, "type"),
		QuantityPattern:     scalarString(t, quantity, "pattern"),
		DeltaPattern:        scalarString(t, delta, "pattern"),
		VersionType:         scalarString(t, version, "type"),
		VersionPattern:      scalarString(t, version, "pattern"),
	}
}

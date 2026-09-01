# ADR 0054: Portable commerce operations contract

- Status: Accepted
- Date: 2026-09-02

## Context

The portable catalog and customer account capabilities stop deliberately before
cart and transaction processing. A useful AI-first scaffold must also remove the
repeated design work around price snapshots, promotion order, checkout
idempotency, order and payment state, callbacks, refunds, fulfillment, returns,
and local demonstrations. Those rules must remain consistent across generated
Go, Java, and Python services and must not assume one payment provider.

## Decision

1. `commerce-operations` version `0.1.0` is one composable vertical capability.
   It requires `commerce-catalog ^0.1.0` and `customer-accounts ^0.1.0`. It owns
   pricing, carts, checkout, orders, fulfillment, returns, campaigns, coupons,
   and payment orchestration while keeping each concern in a separate generated
   module or type boundary.
2. Money crosses JSON and OpenAPI as signed 64-bit decimal strings in the
   currency's smallest unit. Every order line stores an immutable product, SKU,
   name, unit-price, quantity, and line-total snapshot. Checkout never derives
   historical totals from a subsequently edited catalog product.
3. A customer has at most one active cart in an organization and currency. Cart
   mutations use optimistic versions. Checkout requires a bounded idempotency
   key and atomically closes the cart, creates the order and lines, records the
   applied promotion snapshot, and creates one payment intent. Retrying the same
   key returns the same order; reusing it for different input conflicts.
4. Promotion evaluation is deterministic: active campaigns are ordered by
   priority descending and stable identifier ascending; the first eligible
   campaign wins. A coupon can unlock only its linked campaign. Percentage
   discounts use integer floor division and every discount is capped at the
   subtotal. Coupon redemption is reserved in the checkout transaction.
5. Orders move only through `pending_payment -> confirmed -> fulfilling ->
   fulfilled`, with `cancelled` allowed before fulfillment. A fulfilled order can
   enter `return_requested -> returned`. Every state change and administrator
   mutation appends an immutable event and audit record in the same transaction.
6. Payment intents move only through `requires_action -> succeeded` or
   `requires_action -> failed`, followed by bounded partial or full refunds of a
   succeeded amount. Provider callback event identifiers and refund
   idempotency keys are unique per organization. Duplicate delivery returns the
   original result; conflicting reuse is rejected. Reconciliation reads provider
   state through the same adapter interface and applies the same transition.
7. Generated runtime code depends on a small payment gateway interface. The
   default `sandbox` gateway is deterministic, performs no network I/O, exposes
   an explicit development-only completion endpoint, and never accepts real
   credentials. A real provider is an application-owned adapter, not an Engine
   or model dependency.
8. Inventory is not silently decremented. `commerce-operations` exposes stable
   order and fulfillment boundaries so a separately selected inventory adapter
   can reserve and commit stock explicitly in a future compatible capability.
9. Storefront routes authenticate through the separate customer session. Staff
   routes use distinct read, fulfillment, marketing, and payment permissions.
   Tenant-enabled projects repeat the active organization scope in every SQL
   predicate and uniqueness boundary.
10. Email and object-storage demonstrations use the existing local, no-network
    adapters when their capabilities are selected. Commerce examples use the
    sandbox payment gateway and seeded examples only in tests; generators never
    emit customer data, secrets, or production-like credentials.

## Consequences

- An AI selects one named capability instead of designing eight coupled state
  machines and their security invariants from scratch.
- The initial pack deliberately supports one currency per cart/order, one winning
  promotion, whole-line quantities, one fulfillment record, and one return case.
  Taxes, shipping rating, split fulfillment, subscriptions, marketplaces, and
  multi-promotion stacking require later independent capability versions.
- The capability must not be advertised by a backend until its service, SQL,
  HTTP, OpenAPI, shared administration, Nuxt pages, and PostgreSQL/MySQL gates
  pass for that backend.
- This ADR becomes Accepted only after the same contract passes those gates for
  Go, Java, and Python.

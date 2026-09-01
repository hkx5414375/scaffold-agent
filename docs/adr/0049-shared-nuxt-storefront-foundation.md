# ADR 0049: Shared Nuxt storefront foundation

- Status: Accepted
- Date: 2026-09-01

## Context

AI coding agents should not recreate SSR setup, dependency locks, backend proxy
rules, safe failure handling, responsive layout, and frontend quality
configuration for every commerce project. The storefront base also must not
diverge across Go, Java, and Python before business capability pages exist.

## Decision

1. `storefront: nuxt` generates a shared Nuxt 4 SSR project under
   `web/storefront` for every first-party backend adapter.
2. The shared owner is `nuxt-storefront` version `0.1.0`. All direct dependencies
   and relevant compatibility overrides are exact, with a committed npm lockfile
   and a Node.js 24.11 minimum.
3. `SCAFFOLD_API_BASE_URL` remains server-only. The browser initially calls a
   local status endpoint whose backend readiness request has a two-second timeout,
   forwards only a bounded request identifier, and exposes no upstream address or
   failure detail.
4. The generated shell is SSR-enabled, responsive, keyboard accessible, reduced-
   motion aware, and contains a stable error boundary. It does not invent catalog,
   cart, checkout, payment, or account semantics before those capability packs.
5. Go, Java, and Python outputs are compared byte-for-byte. The locked project
   must pass ESLint, Vitest, Nuxt type checking, production build, Prettier, and
   dependency audit.

## Consequences

- Backend parity is established once; later commerce packs add owned routes,
  components, and server endpoints without cloning the storefront foundation.
- Credentials and private backend topology remain outside browser-visible runtime
  configuration.
- This decision completes only the storefront foundation, not the M7 commerce
  domain or customer-facing feature pages.

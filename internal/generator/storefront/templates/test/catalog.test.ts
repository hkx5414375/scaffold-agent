import { describe, expect, it } from "vitest";

import { formatCatalogPrice } from "../app/utils/catalog";
import {
  normalizeCatalogCursor,
  normalizeCatalogLimit,
  normalizeCatalogOrganization,
} from "../server/utils/catalog";

describe("catalog boundary", () => {
  it("validates bounded pagination", () => {
    expect(normalizeCatalogLimit(undefined)).toBe(24);
    expect(normalizeCatalogLimit("100")).toBe(100);
    expect(normalizeCatalogCursor("product-1")).toBe("product-1");
    expect(() => normalizeCatalogLimit("101")).toThrow();
    expect(() => normalizeCatalogCursor("bad\nvalue")).toThrow();
  });

  it("requires configured tenant scope only for tenant storefronts", () => {
    expect(normalizeCatalogOrganization("", false)).toBeUndefined();
    expect(normalizeCatalogOrganization("organization-1", true)).toBe(
      "organization-1",
    );
    expect(() => normalizeCatalogOrganization("", true)).toThrow();
  });

  it("formats safe minor-unit prices and rejects malformed values", () => {
    expect(formatCatalogPrice("1299", "USD")).toContain("12.99");
    expect(formatCatalogPrice("-1", "USD")).toBe("Price unavailable");
  });
});

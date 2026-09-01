import { describe, expect, it } from "vitest";

import {
  normalizeCustomerOrganization,
  parseCustomerSetCookie,
} from "../server/utils/customer";

describe("customer account boundary", () => {
  it("keeps tenant configuration server-scoped and validated", () => {
    expect(normalizeCustomerOrganization("", false)).toBeUndefined();
    expect(normalizeCustomerOrganization("organization-1", true)).toBe(
      "organization-1",
    );
    expect(() => normalizeCustomerOrganization("", true)).toThrow();
    expect(() =>
      normalizeCustomerOrganization("bad\norganization", true),
    ).toThrow();
  });

  it("accepts only the separate customer session cookie", () => {
    expect(
      parseCustomerSetCookie(
        "scaffold_customer_session=customer-token; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax",
      ),
    ).toEqual({ token: "customer-token", clear: false });
    expect(
      parseCustomerSetCookie(
        "scaffold_customer_session=; Path=/; Max-Age=0; HttpOnly",
      ),
    ).toEqual({ token: null, clear: true });
    expect(
      parseCustomerSetCookie("scaffold_session=staff-token; Path=/"),
    ).toEqual({
      token: null,
      clear: false,
    });
  });
});

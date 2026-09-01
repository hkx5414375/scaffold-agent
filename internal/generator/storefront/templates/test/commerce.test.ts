import { describe, expect, it } from "vitest";

import { formatCommerceMoney } from "../app/utils/commerce";
import { commerceCurrency, commerceIdentifier } from "../server/utils/commerce";

describe("commerce boundary", () => {
  it("formats minor units without floating point arithmetic", () => {
    expect(formatCommerceMoney("12345", "USD")).toBe("USD 123.45");
    expect(formatCommerceMoney("invalid", "USD")).toBe("—");
  });
  it("rejects unbounded proxy input", () => {
    expect(commerceIdentifier("order_123")).toBe("order_123");
    expect(() => commerceIdentifier("../order")).toThrow();
    expect(commerceCurrency("USD")).toBe("USD");
    expect(() => commerceCurrency("usd")).toThrow();
  });
});

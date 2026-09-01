import { describe, expect, it } from "vitest";

import {
  normalizeBackendBase,
  normalizeRequestID,
} from "../server/utils/backend";

describe("storefront backend boundary", () => {
  it("accepts explicit HTTP backends and removes one trailing slash", () => {
    expect(normalizeBackendBase("https://api.example.com/")).toBe(
      "https://api.example.com",
    );
    expect(normalizeBackendBase("http://127.0.0.1:8080/api")).toBe(
      "http://127.0.0.1:8080/api",
    );
  });

  it("rejects credential, query, fragment, and non-HTTP URLs", () => {
    for (const value of [
      "https://user:secret@example.com",
      "https://api.example.com?token=secret",
      "https://api.example.com/#fragment",
      "file:///etc/passwd",
      " https://api.example.com",
    ]) {
      expect(() => normalizeBackendBase(value)).toThrow(
        "backend base URL is invalid",
      );
    }
  });

  it("forwards only bounded low-risk request identifiers", () => {
    expect(normalizeRequestID("request-01:edge")).toBe("request-01:edge");
    expect(normalizeRequestID("bad request")).toBeNull();
    expect(normalizeRequestID("x".repeat(129))).toBeNull();
  });
});

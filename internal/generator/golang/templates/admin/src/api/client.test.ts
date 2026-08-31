import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, request } from "./client";

describe("request", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends browser credentials and decodes JSON", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(request<{ status: string }>("/healthz")).resolves.toEqual({ status: "ok" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/healthz",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("preserves stable server error codes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: "tasks.conflict", message: "changed" } }), {
          status: 409,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );

    const failure = await request("/api/v1/tasks").catch((error: unknown) => error);
    expect(failure).toBeInstanceOf(ApiError);
    expect(failure).toMatchObject({ status: 409, code: "tasks.conflict", message: "changed" });
  });
});

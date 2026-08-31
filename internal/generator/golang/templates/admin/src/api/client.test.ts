import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError,{{if .Files}} download,{{end}} request } from "./client";

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
{{- if .Files}}

  it("lets the browser set multipart boundaries", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "file-1" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const form = new FormData();
    form.set("file", new Blob(["content"]), "file.txt");

    await request("/api/v1/files", { method: "POST", body: form });

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(new Headers(init.headers).has("Content-Type")).toBe(false);
  });

  it("returns downloaded binary content", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(new Blob(["content"]))));

    await expect(download("/api/v1/files/file-1/content")).resolves.toBeInstanceOf(Blob);
  });
{{- end}}
});

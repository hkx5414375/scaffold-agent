interface ErrorEnvelope {
  error?: {
    code?: string;
    message?: string;
  };
}

const apiBase = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers,
    credentials: "include",
  });
  if (!response.ok) {
    const envelope = await readError(response);
    throw new ApiError(
      response.status,
      envelope.error?.code ?? "request.failed",
      envelope.error?.message ?? `request failed with status ${response.status}`,
    );
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

async function readError(response: Response): Promise<ErrorEnvelope> {
  try {
    return (await response.json()) as ErrorEnvelope;
  } catch {
    return {};
  }
}

import type { H3Event } from "h3";

import { normalizeBackendBase, normalizeRequestID } from "./backend";

const CUSTOMER_COOKIE = "scaffold_customer_session";
const SESSION_MAX_AGE = 30 * 24 * 60 * 60;

interface CustomerBackendResult<T> {
  data: T;
  status: number;
  sessionToken: string | null;
  clearSession: boolean;
}

export function normalizeCustomerOrganization(
  value: unknown,
  required: boolean,
): string | undefined {
  if (typeof value !== "string") {
    throw createError({
      statusCode: 500,
      statusMessage: "Storefront organization is invalid",
    });
  }
  if (!required && value === "") return undefined;
  if (
    value === "" ||
    value !== value.trim() ||
    value.length > 191 ||
    hasCharacter(value, (codePoint) => codePoint <= 31 || codePoint === 127)
  ) {
    throw createError({
      statusCode: 500,
      statusMessage: required
        ? "Storefront organization is not configured"
        : "Storefront organization is invalid",
    });
  }
  return value;
}

export function parseCustomerSetCookie(value: string | null): {
  token: string | null;
  clear: boolean;
} {
  if (!value || !value.toLowerCase().startsWith(`${CUSTOMER_COOKIE}=`)) {
    return { token: null, clear: false };
  }
  const rawValue =
    value.slice(CUSTOMER_COOKIE.length + 1).split(";", 1)[0] ?? "";
  const clear = /(?:^|;)\s*max-age=0(?:;|$)/i.test(value) || rawValue === "";
  if (clear) return { token: null, clear: true };
  if (
    rawValue.length > 512 ||
    hasCharacter(
      rawValue,
      (codePoint) =>
        codePoint <= 32 ||
        codePoint === 127 ||
        codePoint === 44 ||
        codePoint === 59,
    )
  ) {
    throw createError({
      statusCode: 502,
      statusMessage: "Customer service is unavailable",
    });
  }
  return { token: rawValue, clear: false };
}

function hasCharacter(
  value: string,
  rejected: (codePoint: number) => boolean,
): boolean {
  return Array.from(value).some((character) =>
    rejected(character.codePointAt(0) ?? 0),
  );
}

export async function callCustomerBackend<T>(
  event: H3Event,
  path: string,
  method: "GET" | "POST" | "PUT",
  body?: Record<string, unknown>,
): Promise<CustomerBackendResult<T>> {
  const config = useRuntimeConfig(event);
  const base = normalizeBackendBase(config.apiBaseUrl);
  const organizationID = normalizeCustomerOrganization(
    config.organizationId,
    [[if .Tenancy]]true[[else]]false[[end]],
  );
  const headers = new Headers();
  if (organizationID) headers.set("X-Organization-ID", organizationID);
  const requestID = normalizeRequestID(getRequestHeader(event, "x-request-id"));
  if (requestID) headers.set("X-Request-ID", requestID);
  const sessionToken = getCookie(event, CUSTOMER_COOKIE);
  if (sessionToken) headers.set("Cookie", `${CUSTOMER_COOKIE}=${sessionToken}`);

  let response;
  try {
    response = await $fetch.raw<T>(`${base}${path}`, {
      method,
      body,
      headers,
      ignoreResponseError: true,
      retry: 0,
      timeout: 5_000,
    });
  } catch {
    throw createError({
      statusCode: 502,
      statusMessage: "Customer service is unavailable",
    });
  }
  if (response.status >= 400) {
    const statusCode = [400, 401, 404, 409].includes(response.status)
      ? response.status
      : 502;
    throw createError({
      statusCode,
      statusMessage:
        statusCode === 401
          ? "Customer authentication is required"
          : statusCode === 409
            ? "Customer account changed; reload and try again"
            : statusCode === 400
              ? "Customer account request is invalid"
              : statusCode === 404
                ? "Customer account was not found"
                : "Customer service is unavailable",
    });
  }
  const cookie = parseCustomerSetCookie(response.headers.get("set-cookie"));
  return {
    data: response._data as T,
    status: response.status,
    sessionToken: cookie.token,
    clearSession: cookie.clear,
  };
}

export function applyCustomerSession(
  event: H3Event,
  result: CustomerBackendResult<unknown>,
): void {
  if (result.clearSession) {
    deleteCookie(event, CUSTOMER_COOKIE, { path: "/" });
    return;
  }
  if (!result.sessionToken) return;
  setCookie(event, CUSTOMER_COOKIE, result.sessionToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: SESSION_MAX_AGE,
  });
}

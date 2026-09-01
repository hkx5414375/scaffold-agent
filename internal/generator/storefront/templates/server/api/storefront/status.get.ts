import type { StorefrontStatus } from "../../../shared/types/storefront";
import { normalizeBackendBase, normalizeRequestID } from "../../utils/backend";

export default defineEventHandler(async (event): Promise<StorefrontStatus> => {
  const requestID = normalizeRequestID(getHeader(event, "x-request-id"));
  const headers = requestID ? { "x-request-id": requestID } : undefined;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2000);

  try {
    const configuration = useRuntimeConfig(event);
    await $fetch(`${normalizeBackendBase(configuration.apiBaseUrl)}/readyz`, {
      headers,
      signal: controller.signal,
    });
    return { available: true, request_id: requestID };
  } catch {
    setResponseStatus(event, 503);
    return { available: false, request_id: requestID };
  } finally {
    clearTimeout(timeout);
  }
});

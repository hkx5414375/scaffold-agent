import type { CatalogProduct } from "../../../../shared/types/catalog";
import {
  normalizeBackendBase,
  normalizeRequestID,
} from "../../../utils/backend";
import {
  normalizeCatalogIdentifier,
  normalizeCatalogOrganization,
} from "../../../utils/catalog";

export default defineEventHandler(async (event): Promise<CatalogProduct> => {
  const id = normalizeCatalogIdentifier(
    getRouterParam(event, "id"),
    "product identifier",
  );
  const configuration = useRuntimeConfig(event);
  const organizationID = normalizeCatalogOrganization(
    configuration.organizationId,
    [[if .Tenancy]]true[[else]]false[[end]],
  );
  const requestID = normalizeRequestID(getHeader(event, "x-request-id"));
  const headers: Record<string, string> = {};
  if (requestID) headers["x-request-id"] = requestID;
  if (organizationID) headers["x-organization-id"] = organizationID;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2000);
  try {
    return await $fetch<CatalogProduct>(
      `${normalizeBackendBase(configuration.apiBaseUrl)}/api/v1/storefront/products/${encodeURIComponent(id)}`,
      { headers, signal: controller.signal },
    );
  } catch (error: unknown) {
    const status =
      typeof error === "object" && error !== null && "statusCode" in error
        ? Number(error.statusCode)
        : 0;
    throw createError({
      statusCode: status === 404 ? 404 : 502,
      statusMessage:
        status === 404
          ? "Product was not found"
          : "Catalog is temporarily unavailable",
    });
  } finally {
    clearTimeout(timeout);
  }
});

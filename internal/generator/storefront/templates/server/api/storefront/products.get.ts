import type { CatalogPage } from "../../../shared/types/catalog";
import { normalizeBackendBase, normalizeRequestID } from "../../utils/backend";
import {
  normalizeCatalogCursor,
  normalizeCatalogLimit,
  normalizeCatalogOrganization,
} from "../../utils/catalog";

export default defineEventHandler(async (event): Promise<CatalogPage> => {
  const query = getQuery(event);
  const limit = normalizeCatalogLimit(query.limit);
  const cursor = normalizeCatalogCursor(query.cursor);
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
    return await $fetch<CatalogPage>(
      `${normalizeBackendBase(configuration.apiBaseUrl)}/api/v1/storefront/products`,
      { query: { limit, cursor }, headers, signal: controller.signal },
    );
  } catch {
    throw createError({
      statusCode: 502,
      statusMessage: "Catalog is temporarily unavailable",
    });
  } finally {
    clearTimeout(timeout);
  }
});

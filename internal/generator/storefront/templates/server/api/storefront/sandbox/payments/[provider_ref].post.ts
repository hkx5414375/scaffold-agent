import type { CommerceOrder } from "../../../../../shared/types/commerce";
import { callCommerce, commerceIdentifier } from "../../../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceOrder> => {
  const config = useRuntimeConfig(event);
  if (!config.public.commerceSandboxEnabled)
    throw createError({ statusCode: 404, statusMessage: "Not found" });
  const reference = commerceIdentifier(getRouterParam(event, "provider_ref"));
  const body = await readBody<Record<string, unknown>>(event);
  return callCommerce(
    event,
    `/api/v1/storefront/sandbox/payments/${encodeURIComponent(reference)}/complete`,
    "POST",
    body,
  );
});

import type { CommerceCart } from "../../../../../../shared/types/commerce";
import {
  callCommerce,
  commerceIdentifier,
} from "../../../../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceCart> => {
  const productID = commerceIdentifier(getRouterParam(event, "product_id"));
  const body = await readBody<Record<string, unknown>>(event);
  return callCommerce(
    event,
    `/api/v1/storefront/cart/lines/${encodeURIComponent(productID)}/remove`,
    "POST",
    body,
  );
});

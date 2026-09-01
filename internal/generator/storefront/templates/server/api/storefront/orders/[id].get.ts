import type { CommerceOrder } from "../../../../shared/types/commerce";
import { callCommerce, commerceIdentifier } from "../../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceOrder> => {
  const id = commerceIdentifier(getRouterParam(event, "id"));
  return callCommerce(
    event,
    `/api/v1/storefront/orders/${encodeURIComponent(id)}`,
    "GET",
  );
});

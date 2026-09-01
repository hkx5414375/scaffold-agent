import type { CommerceOrder } from "../../../shared/types/commerce";
import { callCommerce } from "../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceOrder> => {
  const body = await readBody<Record<string, unknown>>(event);
  return callCommerce(event, "/api/v1/storefront/checkout", "POST", body);
});

import type { CommerceOrderPage } from "../../../shared/types/commerce";
import { callCommerce } from "../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceOrderPage> => {
  const query = getQuery(event);
  const params = new URLSearchParams({ limit: "20" });
  if (typeof query.cursor === "string" && query.cursor)
    params.set("cursor", query.cursor);
  return callCommerce(event, `/api/v1/storefront/orders?${params}`, "GET");
});

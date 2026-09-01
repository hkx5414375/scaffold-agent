import type { CommerceCart } from "../../../shared/types/commerce";
import { callCommerce, commerceCurrency } from "../../utils/commerce";

export default defineEventHandler(async (event): Promise<CommerceCart> => {
  const query = getQuery(event);
  const currency = commerceCurrency(query.currency);
  return callCommerce(
    event,
    `/api/v1/storefront/cart?currency=${currency}`,
    "GET",
  );
});

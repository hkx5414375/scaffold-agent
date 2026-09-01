import type { CustomerEnvelope } from "../../../../shared/types/customer";
import { callCustomerBackend } from "../../../utils/customer";

export default defineEventHandler(async (event): Promise<CustomerEnvelope> => {
  const result = await callCustomerBackend<CustomerEnvelope>(
    event,
    "/api/v1/storefront/account/me",
    "GET",
  );
  return result.data;
});

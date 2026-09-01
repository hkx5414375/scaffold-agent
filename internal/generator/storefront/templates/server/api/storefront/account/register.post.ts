import type { CustomerEnvelope } from "../../../../shared/types/customer";
import {
  applyCustomerSession,
  callCustomerBackend,
} from "../../../utils/customer";

export default defineEventHandler(async (event): Promise<CustomerEnvelope> => {
  const input = await readBody<Record<string, unknown>>(event);
  const result = await callCustomerBackend<CustomerEnvelope>(
    event,
    "/api/v1/storefront/account/register",
    "POST",
    input,
  );
  applyCustomerSession(event, result);
  setResponseStatus(event, 201);
  return result.data;
});

import type { CustomerEnvelope } from "../../../../shared/types/customer";
import {
  applyCustomerSession,
  callCustomerBackend,
} from "../../../utils/customer";

export default defineEventHandler(async (event): Promise<CustomerEnvelope> => {
  const input = await readBody<Record<string, unknown>>(event);
  const result = await callCustomerBackend<CustomerEnvelope>(
    event,
    "/api/v1/storefront/account/close",
    "POST",
    input,
  );
  applyCustomerSession(event, { ...result, clearSession: true });
  return result.data;
});

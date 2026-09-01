import type { CustomerPasswordChanged } from "../../../../shared/types/customer";
import {
  applyCustomerSession,
  callCustomerBackend,
} from "../../../utils/customer";

export default defineEventHandler(
  async (event): Promise<CustomerPasswordChanged> => {
    const input = await readBody<Record<string, unknown>>(event);
    const result = await callCustomerBackend<CustomerPasswordChanged>(
      event,
      "/api/v1/storefront/account/password",
      "PUT",
      input,
    );
    applyCustomerSession(event, { ...result, clearSession: true });
    return result.data;
  },
);

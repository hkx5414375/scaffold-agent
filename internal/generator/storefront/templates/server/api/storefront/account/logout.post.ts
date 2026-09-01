import {
  applyCustomerSession,
  callCustomerBackend,
} from "../../../utils/customer";

export default defineEventHandler(async (event): Promise<null> => {
  const result = await callCustomerBackend<null>(
    event,
    "/api/v1/storefront/account/logout",
    "POST",
  );
  applyCustomerSession(event, { ...result, clearSession: true });
  setResponseStatus(event, 204);
  return null;
});

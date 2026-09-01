import type { H3Event } from "h3";

import { callCustomerBackend } from "./customer";

const IDENTIFIER = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$/;

export function commerceIdentifier(value: string | undefined): string {
  if (!value || !IDENTIFIER.test(value)) {
    throw createError({
      statusCode: 400,
      statusMessage: "Commerce identifier is invalid",
    });
  }
  return value;
}

export function commerceCurrency(value: unknown): string {
  if (typeof value !== "string" || !/^[A-Z]{3}$/.test(value)) {
    throw createError({
      statusCode: 400,
      statusMessage: "Currency is invalid",
    });
  }
  return value;
}

export async function callCommerce<T>(
  event: H3Event,
  path: string,
  method: "GET" | "POST" | "PUT",
  body?: Record<string, unknown>,
): Promise<T> {
  const result = await callCustomerBackend<T>(event, path, method, body);
  return result.data;
}

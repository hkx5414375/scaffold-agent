const REQUEST_ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;

export function normalizeBackendBase(value: unknown): string {
  if (
    typeof value !== "string" ||
    value !== value.trim() ||
    value.length > 2048
  ) {
    throw new Error("backend base URL is invalid");
  }
  const parsed = new URL(value);
  if (
    !["http:", "https:"].includes(parsed.protocol) ||
    parsed.username ||
    parsed.password ||
    parsed.search ||
    parsed.hash
  ) {
    throw new Error("backend base URL is invalid");
  }
  return parsed.toString().replace(/\/$/, "");
}

export function normalizeRequestID(value: string | undefined): string | null {
  if (!value || !REQUEST_ID_PATTERN.test(value)) {
    return null;
  }
  return value;
}

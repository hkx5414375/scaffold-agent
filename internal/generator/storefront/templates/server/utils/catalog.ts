function hasControlCharacter(value: string): boolean {
  return Array.from(value).some((character) => {
    const codePoint = character.codePointAt(0) ?? 0;
    return codePoint <= 31 || codePoint === 127;
  });
}

function oneString(value: unknown): string {
  if (Array.isArray(value)) {
    throw createError({
      statusCode: 400,
      statusMessage: "Invalid catalog query",
    });
  }
  return typeof value === "string" ? value : "";
}

export function normalizeCatalogLimit(value: unknown): number {
  const raw = oneString(value);
  if (raw === "") return 24;
  if (!/^[0-9]{1,3}$/.test(raw)) {
    throw createError({
      statusCode: 400,
      statusMessage: "Invalid catalog limit",
    });
  }
  const limit = Number(raw);
  if (limit < 1 || limit > 100) {
    throw createError({
      statusCode: 400,
      statusMessage: "Invalid catalog limit",
    });
  }
  return limit;
}

export function normalizeCatalogIdentifier(
  value: unknown,
  label: string,
): string {
  const identifier = oneString(value);
  if (
    identifier === "" ||
    identifier !== identifier.trim() ||
    identifier.length > 191 ||
    hasControlCharacter(identifier)
  ) {
    throw createError({
      statusCode: 400,
      statusMessage: `Invalid catalog ${label}`,
    });
  }
  return identifier;
}

export function normalizeCatalogCursor(value: unknown): string | undefined {
  const cursor = oneString(value);
  if (cursor === "") return undefined;
  return normalizeCatalogIdentifier(cursor, "cursor");
}

export function normalizeCatalogOrganization(
  value: unknown,
  required: boolean,
): string | undefined {
  const organizationID = oneString(value);
  if (!required && organizationID === "") return undefined;
  if (required && organizationID === "") {
    throw createError({
      statusCode: 500,
      statusMessage: "Storefront organization is not configured",
    });
  }
  return normalizeCatalogIdentifier(organizationID, "organization");
}

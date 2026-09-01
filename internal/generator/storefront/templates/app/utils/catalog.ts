const DECIMAL_INTEGER = /^(0|[1-9][0-9]*)$/;

export function formatCatalogPrice(
  priceMinor: string,
  currency: string,
): string {
  if (!DECIMAL_INTEGER.test(priceMinor) || !/^[A-Z]{3}$/.test(currency)) {
    return "Price unavailable";
  }
  let formatter: Intl.NumberFormat;
  try {
    formatter = new Intl.NumberFormat("en", { style: "currency", currency });
  } catch {
    return `${currency} ${priceMinor} minor units`;
  }
  const fractionDigits = formatter.resolvedOptions().maximumFractionDigits ?? 2;
  const numeric = Number(priceMinor);
  if (!Number.isSafeInteger(numeric)) {
    return `${currency} ${priceMinor} minor units`;
  }
  return formatter.format(numeric / 10 ** fractionDigits);
}

export function formatCommerceMoney(value: string, currency: string): string {
  if (!/^(0|[1-9][0-9]{0,18})$/.test(value) || !/^[A-Z]{3}$/.test(currency))
    return "—";
  const minor = BigInt(value);
  const whole = minor / 100n;
  const fraction = (minor % 100n).toString().padStart(2, "0");
  return `${currency} ${whole}.${fraction}`;
}

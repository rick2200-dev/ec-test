/** Formats an amount in JPY with locale-aware thousands separator. */
export function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

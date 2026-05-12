function swapSeparators(s: string): string {
  return s.replace(/,/g, "\x00").replace(/\./g, ",").replace(/\x00/g, ".");
}

function intlUSD(value: number, opts: Intl.NumberFormatOptions = {}): string {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", ...opts }).format(value);
}

/** Compact: ≥10 K → compact notation; otherwise 2-decimal full. */
export function formatUSD(value: number): string {
  const abs = Math.abs(value);
  let str: string;
  if (abs >= 1_000_000) {
    str = intlUSD(value, { notation: "compact", compactDisplay: "short", maximumFractionDigits: 2 });
  } else if (abs >= 10_000) {
    str = intlUSD(value, { notation: "compact", compactDisplay: "short", maximumFractionDigits: 1 });
  } else {
    str = intlUSD(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }
  return str.replace(/[\d,.]+/, (m) => swapSeparators(m));
}

/** Always 2 decimal places, no compact. */
export function formatUSDFull(value: number): string {
  return intlUSD(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
    .replace(/[\d,.]+/, (m) => swapSeparators(m));
}

/** 0 decimal places, no compact. For chart Y-axes and integer amounts. */
export function formatUSDNoFrac(value: number): string {
  return intlUSD(value, { maximumFractionDigits: 0 })
    .replace(/[\d,.]+/, (m) => swapSeparators(m));
}

/** Like formatUSDFull but prepends "+" for positive values. */
export function formatUSDSigned(value: number): string {
  const sign = value >= 0 ? "+" : "";
  return sign + formatUSDFull(value);
}

/** Percentage with mandatory sign (+/-). */
export function formatPct(value: number, decimals = 2): string {
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(decimals).replace(".", ",")}%`;
}

/** Percentage without sign prefix. */
export function formatPctPlain(value: number, decimals = 2): string {
  return `${value.toFixed(decimals).replace(".", ",")}%`;
}

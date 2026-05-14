function swapSeparators(s: string): string {
  // eslint-disable-next-line no-control-regex
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

/** Percentage with sign, max 2 decimals, no trailing zeros. */
export function formatPctTrimmed(value: number): string {
  const sign = value >= 0 ? "+" : "";
  const rounded = Math.round(value * 100) / 100;
  return `${sign}${rounded.toString().replace(".", ",")}%`;
}

/** Compact USD for chart axes: B for billions, M for millions, k for thousands. */
export function formatUSDCompact(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? "-" : "";

  if (abs >= 1_000_000_000) {
    return `$${sign}${(abs / 1_000_000_000).toFixed(1)}B`;
  } else if (abs >= 1_000_000) {
    return `$${sign}${(abs / 1_000_000).toFixed(0)}M`;
  } else if (abs >= 1_000) {
    return `$${sign}${(abs / 1_000).toFixed(0)}k`;
  } else {
    return `$${sign}${Math.floor(abs)}`;
  }
}

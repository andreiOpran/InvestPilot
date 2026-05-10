import { useEffect, useRef, useState } from "react";
import { Input } from "@/components/ui/input";

interface CurrencyInputProps {
  value: number;
  onChange: (value: number) => void;
  onBlur?: () => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

function toDisplay(clean: string): string {
  if (!clean) return "";
  const [intPart, decPart] = clean.split(",");
  const formattedInt = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const base = decPart !== undefined ? `${formattedInt},${decPart}` : formattedInt;
  return base ? `${base} $` : "";
}

function toNumber(clean: string): number {
  if (!clean) return 0;
  return parseFloat(clean.replace(",", ".")) || 0;
}

function fromNumber(n: number): string {
  if (!n) return "";
  return n.toFixed(2).replace(/\.?0+$/, "").replace(".", ",");
}

export function CurrencyInput({
  value,
  onChange,
  onBlur,
  placeholder,
  className,
  disabled,
}: CurrencyInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [clean, setClean] = useState(() => fromNumber(value));
  const isFocused = useRef(false);

  useEffect(() => {
    if (!isFocused.current && toNumber(clean) !== value) {
      setClean(fromNumber(value));
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    const selStart = e.target.selectionStart ?? raw.length;

    let newClean = "";
    let commaFound = false;
    let decDigits = 0;
    let cursorInClean = 0;

    for (let i = 0; i < raw.length; i++) {
      if (i === selStart) cursorInClean = newClean.length;
      const ch = raw[i];
      if (ch >= "0" && ch <= "9") {
        if (commaFound) {
          if (decDigits < 2) { newClean += ch; decDigits++; }
        } else {
          newClean += ch;
        }
      } else if (ch === "," && !commaFound) {
        newClean += ch;
        commaFound = true;
      }
    }
    if (selStart >= raw.length) cursorInClean = newClean.length;

    // Remove leading zeros from integer part
    const ci = newClean.indexOf(",");
    const intClean = ci === -1 ? newClean : newClean.slice(0, ci);
    const decClean = ci === -1 ? "" : newClean.slice(ci);
    const trimmedInt = intClean.replace(/^0+(\d)/, "$1");
    const removed = intClean.length - trimmedInt.length;
    newClean = trimmedInt + decClean;
    if (ci === -1 || cursorInClean <= intClean.length) {
      cursorInClean = Math.max(0, cursorInClean - removed);
    }

    const display = toDisplay(newClean);

    // Map cursor from clean-string position → display position
    let displayCursor = 0;
    let cleanIdx = 0;
    for (let i = 0; i <= display.length; i++) {
      if (cleanIdx === cursorInClean) { displayCursor = i; break; }
      const ch = display[i];
      if (ch !== undefined && ((ch >= "0" && ch <= "9") || ch === ",")) cleanIdx++;
      displayCursor = i + 1;
    }
    const suffixStart = display.lastIndexOf(" $");
    if (suffixStart >= 0 && displayCursor > suffixStart) displayCursor = suffixStart;

    setClean(newClean);
    onChange(toNumber(newClean));

    requestAnimationFrame(() => {
      inputRef.current?.setSelectionRange(displayCursor, displayCursor);
    });
  };

  return (
    <Input
      ref={inputRef}
      type="text"
      inputMode="decimal"
      placeholder={placeholder}
      className={className}
      disabled={disabled}
      value={toDisplay(clean)}
      onChange={handleChange}
      onFocus={() => { isFocused.current = true; }}
      onBlur={() => {
        isFocused.current = false;
        onBlur?.();
      }}
    />
  );
}

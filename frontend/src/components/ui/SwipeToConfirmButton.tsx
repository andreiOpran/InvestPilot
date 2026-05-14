import { useRef, useState, useEffect, useCallback } from "react";
import { Loader2, Check, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface SwipeToConfirmButtonProps {
  label: string;
  onConfirm: () => Promise<void> | void;
  isLoading?: boolean;
  disabled?: boolean;
  open?: boolean;
  variant?: "default" | "destructive";
  className?: string;
}

type State = "idle" | "dragging" | "confirmed" | "loading" | "success";

const THUMB_SIZE = 32;
const CONFIRM_THRESHOLD = 0.85;

export function SwipeToConfirmButton({
  label,
  onConfirm,
  isLoading = false,
  disabled = false,
  open,
  variant = "default",
  className,
}: SwipeToConfirmButtonProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<State>("idle");
  const [dragX, setDragX] = useState(0);
  const [trackWidth, setTrackWidth] = useState(0);

  // Refs so event handlers always read current values without causing effect re-runs
  const dragStartX = useRef(0);
  const isDragging = useRef(false);
  const currentDragX = useRef(0);
  const onConfirmRef = useRef(onConfirm);
  const maxDragRef = useRef(0);
  const stateRef = useRef<State>("idle");

  useEffect(() => {
    onConfirmRef.current = onConfirm;
  }, [onConfirm]);

  const maxDrag = trackWidth - THUMB_SIZE - 8;

  useEffect(() => {
    maxDragRef.current = trackWidth - THUMB_SIZE - 8;
  }, [trackWidth]);

  const progress = maxDrag > 0 ? dragX / maxDrag : 0;

  const reset = useCallback(() => {
    setState("idle");
    stateRef.current = "idle";
    setDragX(0);
    currentDragX.current = 0;
  }, []);

  // Reset on dialog reopen
  useEffect(() => {
    if (!open) return;
    const id = setTimeout(reset, 0);
    return () => clearTimeout(id);
  }, [open, reset]);

  // Track loading state transitions
  useEffect(() => {
    if (isLoading && stateRef.current === "confirmed") {
      setState("loading");
      stateRef.current = "loading";
    }
    if (!isLoading && stateRef.current === "loading") {
      setState("success");
      stateRef.current = "success";
      const t = setTimeout(() => reset(), 2500);
      return () => clearTimeout(t);
    }
  }, [isLoading, reset]);

  // Validation failed guard: if confirmed but isLoading never flips, snap back
  useEffect(() => {
    if (state !== "confirmed") return;
    const t = setTimeout(() => {
      if (stateRef.current === "confirmed") reset();
    }, 600);
    return () => clearTimeout(t);
  }, [state, reset]);

  const updateTrackWidth = useCallback(() => {
    if (trackRef.current) {
      setTrackWidth(trackRef.current.offsetWidth);
    }
  }, []);

  useEffect(() => {
    updateTrackWidth();
    const ro = new ResizeObserver(updateTrackWidth);
    if (trackRef.current) ro.observe(trackRef.current);
    return () => ro.disconnect();
  }, [updateTrackWidth]);

  const getClientX = (e: MouseEvent | TouchEvent) =>
    "touches" in e ? e.touches[0].clientX : e.clientX;

  // Single effect with no external deps — reads everything via refs
  useEffect(() => {
    const onMove = (e: MouseEvent | TouchEvent) => {
      if (!isDragging.current) return;
      const delta = getClientX(e) - dragStartX.current;
      const clamped = Math.max(0, Math.min(delta, maxDragRef.current));
      currentDragX.current = clamped;
      setDragX(clamped);
    };

    const onUp = () => {
      if (!isDragging.current) return;
      isDragging.current = false;

      const prog = maxDragRef.current > 0
        ? currentDragX.current / maxDragRef.current
        : 0;

      if (prog >= CONFIRM_THRESHOLD) {
        setState("confirmed");
        stateRef.current = "confirmed";
        setDragX(maxDragRef.current);
        currentDragX.current = maxDragRef.current;
        setTimeout(async () => {
          try {
            await onConfirmRef.current();
          } catch {
            // onConfirm threw (e.g. validation/API error) — snap back
            setState("idle");
            stateRef.current = "idle";
            setDragX(0);
            currentDragX.current = 0;
          }
        }, 120);
      } else {
        setState("idle");
        stateRef.current = "idle";
        setDragX(0);
        currentDragX.current = 0;
      }
    };

    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    window.addEventListener("touchmove", onMove, { passive: false });
    window.addEventListener("touchend", onUp);
    return () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      window.removeEventListener("touchmove", onMove);
      window.removeEventListener("touchend", onUp);
    };
  }, []); // stable — all mutable state via refs

  const onPointerDown = (e: React.MouseEvent | React.TouchEvent) => {
    if (stateRef.current !== "idle" || isLoading || disabled) return;
    updateTrackWidth();
    isDragging.current = true;
    dragStartX.current = "touches" in e ? e.touches[0].clientX : e.clientX;
    setState("dragging");
    stateRef.current = "dragging";
    e.preventDefault();
  };

  const isDestructive = variant === "destructive";
  const isActive = state === "dragging";
  const isConfirmed = state === "confirmed" || state === "loading";
  const isSuccess = state === "success";
  const isDisabled = isLoading || isConfirmed || isSuccess || disabled;

  const fillWidth = isConfirmed || isSuccess
    ? "100%"
    : `${THUMB_SIZE / 2 + dragX + THUMB_SIZE / 2}px`;

  const thumbTranslate = isConfirmed || isSuccess ? maxDrag : dragX;

  return (
    <div
      ref={trackRef}
      className={cn(
        "relative h-10 w-full select-none overflow-hidden rounded-xl",
        isDestructive
          ? "bg-red-950/40 border border-red-900/30"
          : "bg-primary/10 border border-primary/20",
        "transition-all duration-200",
        disabled && "opacity-40 cursor-not-allowed",
        className
      )}
      style={{ userSelect: "none", WebkitUserSelect: "none" }}
    >
      {/* Fill bar */}
      <div
        className={cn(
          "absolute inset-y-0 left-0 rounded-xl",
          isDestructive
            ? "bg-gradient-to-r from-red-700 to-red-500"
            : "bg-gradient-to-r from-primary/80 to-primary"
        )}
        style={{
          width: fillWidth,
          transition: isActive ? "none" : "width 0.45s cubic-bezier(0.22, 1, 0.36, 1)",
          opacity: isConfirmed || isSuccess ? 1 : 0.7 + progress * 0.3,
        }}
      />

      {/* Label */}
      <div
        className="absolute inset-0 flex items-center justify-center pointer-events-none z-10"
        style={{
          opacity: isSuccess ? 0 : Math.max(0, 1 - progress * 1.8),
          transition: isActive ? "none" : "opacity 0.3s ease",
        }}
      >
        <span className={cn(
          "text-sm font-semibold tracking-wide",
          isDestructive ? "text-red-200" : "text-primary"
        )}>
          {label}
        </span>
      </div>

      {/* Success checkmark */}
      {isSuccess && (
        <div className="absolute inset-0 flex items-center justify-center z-10 pointer-events-none">
          <Check className={cn(
            "h-5 w-5",
            isDestructive ? "text-red-100" : "text-primary-foreground"
          )} />
        </div>
      )}

      {/* Thumb */}
      <div
        className={cn(
          "absolute top-1 bottom-1 left-1 z-20 flex items-center justify-center rounded-lg",
          isDestructive
            ? "bg-gradient-to-br from-red-500 to-red-700"
            : "bg-gradient-to-br from-primary/90 to-primary",
          "shadow-lg shadow-black/20",
          !isDisabled && "cursor-grab active:cursor-grabbing"
        )}
        style={{
          width: THUMB_SIZE,
          transform: `translateX(${thumbTranslate}px)`,
          transition: isActive ? "none" : "transform 0.45s cubic-bezier(0.22, 1, 0.36, 1)",
        }}
        onMouseDown={onPointerDown}
        onTouchStart={onPointerDown}
      >
        {isLoading || state === "confirmed" ? (
          <Loader2 className="h-3.5 w-3.5 text-primary-foreground animate-spin" />
        ) : isSuccess ? (
          <Check className="h-3.5 w-3.5 text-primary-foreground" />
        ) : (
          <div
            className="flex items-center"
            style={{ opacity: Math.max(0.2, 1 - progress * 2.5) }}
          >
            <ChevronRight className="h-3 w-3 text-primary-foreground -mr-1" />
            <ChevronRight className="h-3 w-3 text-primary-foreground opacity-60" />
          </div>
        )}
      </div>
    </div>
  );
}

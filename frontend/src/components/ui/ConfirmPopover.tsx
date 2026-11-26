import { Loader2 } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from "react";

interface ConfirmPopoverProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  busy?: boolean;
  onConfirm: () => void | Promise<void>;
  children: (props: {
    open: boolean;
    toggle: (e: React.MouseEvent<HTMLElement>) => void;
    close: () => void;
  }) => React.ReactNode;
}

interface PopoverCoords {
  top: number;
  left: number;
  placement: "top" | "bottom";
  arrowLeft: number;
}

/**
 * Top-layer confirmation popover anchored above (or below near top edge) its trigger.
 * Uses native `popover="auto"` to escape overflow clipping and stack cleanly over modals.
 */
export function ConfirmPopover({
  title,
  description,
  confirmLabel = "Xóa",
  cancelLabel = "Hủy",
  busy = false,
  onConfirm,
  children,
}: ConfirmPopoverProps) {
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [coords, setCoords] = useState<PopoverCoords | null>(null);
  const triggerWrapRef = useRef<HTMLSpanElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const cancelBtnRef = useRef<HTMLButtonElement>(null);
  const popoverId = useId();
  const titleId = useId();

  const updatePosition = useCallback(() => {
    const triggerEl = triggerWrapRef.current;
    const popEl = popoverRef.current;
    if (!triggerEl || !popEl) return;

    const tr = triggerEl.getBoundingClientRect();
    const pr = popEl.getBoundingClientRect();
    const gap = 8;
    const margin = 8;
    const popWidth = pr.width || 240;
    const popHeight = pr.height || 100;

    const fitsAbove = tr.top - popHeight - gap >= margin;
    const placement: "top" | "bottom" = fitsAbove ? "top" : "bottom";
    const top =
      placement === "top"
        ? Math.max(margin, tr.top - popHeight - gap)
        : Math.min(window.innerHeight - popHeight - margin, tr.bottom + gap);

    const idealLeft = tr.right - popWidth;
    const left = Math.max(
      margin,
      Math.min(idealLeft, window.innerWidth - popWidth - margin)
    );
    const triggerCenter = tr.left + tr.width / 2;
    const arrowLeft = Math.max(14, Math.min(popWidth - 14, triggerCenter - left));

    setCoords({ top, left, placement, arrowLeft });
  }, []);

  const close = useCallback(() => {
    const popEl = popoverRef.current as (HTMLDivElement & {
      hidePopover?: () => void;
    }) | null;
    try {
      if (popEl?.matches(":popover-open")) {
        popEl.hidePopover?.();
      }
    } catch {
      // ignore if popover API unavailable
    }
    setOpen(false);
  }, []);

  const toggle = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      e.stopPropagation();
      if (open) {
        close();
      } else {
        setOpen(true);
      }
    },
    [close, open]
  );

  useLayoutEffect(() => {
    const popEl = popoverRef.current as (HTMLDivElement & {
      showPopover?: () => void;
      hidePopover?: () => void;
    }) | null;
    if (!popEl) return;

    if (open) {
      try {
        if (!popEl.matches(":popover-open")) {
          popEl.showPopover?.();
        }
      } catch {
        // fallback handled by fixed positioning
      }
      updatePosition();
      requestAnimationFrame(() => {
        updatePosition();
        cancelBtnRef.current?.focus();
      });
    } else {
      try {
        if (popEl.matches(":popover-open")) {
          popEl.hidePopover?.();
        }
      } catch {
        // ignore
      }
    }
  }, [open, updatePosition]);

  useEffect(() => {
    const popEl = popoverRef.current;
    if (!popEl) return;

    const handleToggle = (ev: Event) => {
      const toggleEv = ev as Event & { newState?: string };
      if (toggleEv.newState === "closed") {
        setOpen(false);
      }
    };
    popEl.addEventListener("toggle", handleToggle);
    return () => popEl.removeEventListener("toggle", handleToggle);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onResizeOrScroll = () => updatePosition();
    window.addEventListener("resize", onResizeOrScroll);
    window.addEventListener("scroll", onResizeOrScroll, true);
    return () => {
      window.removeEventListener("resize", onResizeOrScroll);
      window.removeEventListener("scroll", onResizeOrScroll, true);
    };
  }, [open, updatePosition]);

  const isBusy = busy || submitting;

  const handleConfirmClick = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isBusy) return;
    setSubmitting(true);
    try {
      await onConfirm();
      close();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <span
      ref={triggerWrapRef}
      className="relative inline-flex items-center shrink-0"
      onClick={(e) => e.stopPropagation()}
    >
      {children({ open, toggle, close })}

      <div
        ref={popoverRef}
        id={popoverId}
        {...({ popover: "auto" } as React.HTMLAttributes<HTMLDivElement>)}
        role="alertdialog"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        style={
          coords
            ? {
                position: "fixed",
                inset: "auto",
                margin: 0,
                top: `${coords.top}px`,
                left: `${coords.left}px`,
              }
            : {
                position: "fixed",
                inset: "auto",
                margin: 0,
                top: "-9999px",
                left: "-9999px",
              }
        }
        className="w-[244px] max-w-[calc(100vw-16px)] rounded-md border border-edge bg-elevated p-3 text-left text-primary shadow-xl overflow-visible z-50"
      >
        {coords && (
          <span
            aria-hidden="true"
            style={{ left: `${coords.arrowLeft - 5}px` }}
            className={`absolute w-2.5 h-2.5 rotate-45 bg-elevated border-edge ${
              coords.placement === "top"
                ? "-bottom-[6px] border-r border-b"
                : "-top-[6px] border-l border-t"
            }`}
          />
        )}

        <div id={titleId} className="text-xs font-semibold text-primary leading-snug">
          {title}
        </div>
        {description && (
          <div className="mt-1 text-[11px] text-muted leading-relaxed">
            {description}
          </div>
        )}

        <div className="mt-2.5 flex items-center justify-end gap-1.5">
          <button
            ref={cancelBtnRef}
            type="button"
            disabled={isBusy}
            onClick={(e) => {
              e.stopPropagation();
              close();
            }}
            className="h-7 px-2.5 rounded border border-edge bg-surface hover:bg-canvas text-[11px] font-medium text-primary transition-colors disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            disabled={isBusy}
            onClick={handleConfirmClick}
            className="h-7 px-2.5 rounded bg-status-danger hover:bg-status-danger/90 text-white text-[11px] font-semibold inline-flex items-center gap-1 transition-colors disabled:opacity-50"
          >
            {isBusy && <Loader2 className="w-3 h-3 animate-spin" />}
            <span>{confirmLabel}</span>
          </button>
        </div>
      </div>
    </span>
  );
}


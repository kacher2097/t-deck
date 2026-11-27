import { X } from "lucide-react";
import type React from "react";
import { useEffect, useId, useRef } from "react";

interface ModalDialogProps {
  open: boolean;
  onClose: () => void;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  headerExtra?: React.ReactNode;
  subHeader?: React.ReactNode;
  size?: "md" | "lg" | "xl" | "2xl";
  /** Lock height so switching tabs never resizes the dialog. */
  fixedHeight?: boolean;
  children: React.ReactNode;
}

const SIZE_CLASSES: Record<NonNullable<ModalDialogProps["size"]>, string> = {
  md: "w-[560px] max-w-[94vw]",
  lg: "w-[760px] max-w-[94vw]",
  xl: "w-[1080px] max-w-[94vw]",
  "2xl": "w-[1260px] max-w-[93vw]",
};

function isPointerOutsideDialog(
  e: React.PointerEvent<HTMLDialogElement>,
  dialog: HTMLDialogElement
): boolean {
  const r = dialog.getBoundingClientRect();
  return (
    e.clientX < r.left ||
    e.clientX > r.right ||
    e.clientY < r.top ||
    e.clientY > r.bottom
  );
}

export function ModalDialog({
  open,
  onClose,
  title,
  subtitle,
  headerExtra,
  subHeader,
  size = "md",
  fixedHeight = false,
  children,
}: ModalDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const pressedOutsideRef = useRef(false);
  const titleId = useId();

  useEffect(() => {
    const dlg = dialogRef.current;
    if (!dlg) return;
    if (open) {
      if (!dlg.open) dlg.showModal();
    } else if (dlg.open) {
      dlg.close();
    }
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      onClose={onClose}
      onPointerDown={(e) => {
        const dlg = dialogRef.current;
        pressedOutsideRef.current = dlg ? isPointerOutsideDialog(e, dlg) : false;
      }}
      onPointerUp={(e) => {
        const dlg = dialogRef.current;
        const wasOutside = pressedOutsideRef.current;
        pressedOutsideRef.current = false;
        if (wasOutside && dlg && isPointerOutsideDialog(e, dlg)) {
          onClose();
        }
      }}
      className={`tdeck-dialog ${SIZE_CLASSES[size]} ${fixedHeight ? "h-[780px] max-h-[88vh]" : "max-h-[88vh]"} rounded-xl bg-surface border border-white/10 text-primary p-0 overflow-hidden open:flex open:flex-col`}
    >
      <div className="px-7 py-4 border-b border-edge flex items-center justify-between gap-4 shrink-0">
        <div className="min-w-0 flex-1">
          <div id={titleId} className="flex items-center gap-2.5 flex-wrap">
            {title}
          </div>
          {subtitle && (
            <div className="text-xs text-muted mt-0.5 truncate">{subtitle}</div>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {headerExtra}
          <button
            type="button"
            onClick={onClose}
            aria-label="Đóng"
            className="h-8 w-8 rounded-md border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {subHeader && <div className="shrink-0">{subHeader}</div>}

      <div
        className="flex-1 min-h-0 overflow-y-auto"
        style={{ scrollbarGutter: "stable" }}
      >
        {children}
      </div>
    </dialog>
  );
}


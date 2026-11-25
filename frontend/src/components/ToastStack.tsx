import { AlertTriangle, CheckCircle2, X, XCircle } from "lucide-react";
import type { ToastMessage } from "../hooks/useToast";

interface ToastStackProps {
  toasts: ToastMessage[];
  onDismiss: (id: number) => void;
}

export function ToastStack({ toasts, onDismiss }: ToastStackProps) {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`px-3 py-2.5 rounded border bg-surface text-xs flex items-start gap-2 ${
            t.kind === "success"
              ? "border-status-ready/60 text-primary"
              : t.kind === "warn"
              ? "border-status-warn/60 text-primary"
              : "border-status-danger/60 text-primary"
          }`}
        >
          {t.kind === "success" ? (
            <CheckCircle2 className="w-4 h-4 text-status-ready shrink-0 mt-0.5" />
          ) : t.kind === "warn" ? (
            <AlertTriangle className="w-4 h-4 text-status-warn shrink-0 mt-0.5" />
          ) : (
            <XCircle className="w-4 h-4 text-status-danger shrink-0 mt-0.5" />
          )}
          <span className="flex-1 leading-snug">{t.text}</span>
          <button
            type="button"
            onClick={() => onDismiss(t.id)}
            aria-label="Đóng thông báo"
            className="text-muted hover:text-primary"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      ))}
    </div>
  );
}


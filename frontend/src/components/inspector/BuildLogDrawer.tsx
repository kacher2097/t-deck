import { Copy, Square, Terminal, X } from "lucide-react";
import { useEffect, useRef } from "react";
import type {
  BuildLogEvent,
  BuildStatusEvent,
  ReleaseProgressEvent,
} from "../../lib/events";

interface BuildLogDrawerProps {
  open: boolean;
  jobId: string;
  status: BuildStatusEvent["status"] | null;
  errorMsg?: string;
  progress: ReleaseProgressEvent | null;
  logs: BuildLogEvent[];
  onCancel: () => void;
  onClose: () => void;
}

export function BuildLogDrawer({
  open,
  jobId,
  status,
  errorMsg,
  progress,
  logs,
  onCancel,
  onClose,
}: BuildLogDrawerProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs.length]);

  if (!open) return null;

  const pct = progress ? Math.min(100, Math.max(0, progress.percent || 0)) : 0;

  return (
    <div className="mt-4 rounded-md border border-edge bg-canvas overflow-hidden">
      <div className="px-3 py-2 border-b border-edge bg-surface flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <Terminal className="w-3.5 h-3.5 text-status-ready shrink-0" />
          <span className="text-xs font-mono font-semibold text-primary truncate">
            Pipeline {jobId ? jobId.slice(0, 8) : ""} · {status || "idle"}
          </span>
        </div>

        <div className="flex items-center gap-1.5 shrink-0">
          <button
            type="button"
            onClick={() => {
              const txt = logs.map((l) => `[${l.stream}] ${l.line}`).join("\n");
              navigator.clipboard?.writeText(txt).catch(() => {});
            }}
            className="h-6 px-2 rounded border border-edge bg-canvas hover:bg-elevated text-[11px] text-muted hover:text-primary inline-flex items-center gap-1"
          >
            <Copy className="w-3 h-3" />
            <span>Copy</span>
          </button>
          {status === "running" && jobId && (
            <button
              type="button"
              onClick={onCancel}
              className="h-6 px-2 rounded border border-status-danger/60 bg-status-danger/10 text-[11px] text-status-danger inline-flex items-center gap-1"
            >
              <Square className="w-2.5 h-2.5 fill-current" />
              <span>Hủy</span>
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            aria-label="Đóng log"
            className="h-6 w-6 rounded border border-edge bg-canvas text-muted hover:text-primary flex items-center justify-center"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      </div>

      {progress && (
        <div className="px-3 py-1.5 border-b border-edge bg-surface/50 space-y-1">
          <div className="flex items-center justify-between text-[11px] font-mono text-muted">
            <span>Stage: {progress.stage}</span>
            <span>{pct.toFixed(0)}%</span>
          </div>
          <div className="h-1.5 rounded-full bg-canvas overflow-hidden">
            <div
              className="h-full bg-status-ready transition-all duration-150"
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      )}

      <div
        ref={scrollRef}
        className="p-3 max-h-[180px] overflow-y-auto font-mono text-[11px] leading-relaxed space-y-0.5"
      >
        {logs.length === 0 ? (
          <div className="text-muted/70">Đang chờ output...</div>
        ) : (
          logs.map((l, idx) => (
            <div
              key={`${l.ts}-${idx}`}
              className={
                l.stream === "stderr" ? "text-status-warn" : "text-primary/90"
              }
            >
              {l.line}
            </div>
          ))
        )}
        {errorMsg && (
          <div className="text-status-danger pt-1 font-semibold">
            ERROR: {errorMsg}
          </div>
        )}
      </div>
    </div>
  );
}

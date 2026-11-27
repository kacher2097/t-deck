import {
  AlertTriangle,
  Download,
  ExternalLink,
  Loader2,
  Play,
  Square,
} from "lucide-react";
import type { ProjectView, RunningProcess } from "../../types/project";

export const PROJECT_STATUS_META: Record<
  ProjectView["status"],
  { label: string; dotClass: string; textClass: string; pillClass: string }
> = {
  ready: {
    label: "READY",
    dotClass: "bg-status-ready",
    textClass: "text-status-ready",
    pillClass:
      "bg-status-ready/15 border border-status-ready/40 text-[#34D399]",
  },
  missing_env: {
    label: "MISSING .ENV",
    dotClass: "bg-status-warn",
    textClass: "text-status-warn",
    pillClass: "bg-status-warn/15 border border-status-warn/45 text-[#FBBF24]",
  },
  cloud_only: {
    label: "CLOUD ONLY",
    dotClass: "bg-status-cloud",
    textClass: "text-status-cloud",
    pillClass:
      "bg-status-cloud/15 border border-status-cloud/40 text-status-cloud",
  },
  missing_dir: {
    label: "MISSING DIR",
    dotClass: "bg-status-danger",
    textClass: "text-status-danger",
    pillClass:
      "bg-status-danger/15 border border-status-danger/45 text-[#F87171]",
  },
};

interface ProjectPrimaryActionButtonProps {
  project: ProjectView;
  runningProc?: RunningProcess;
  isBusy: boolean;
  stopPropagation?: boolean;
  size?: "card" | "table";
  onLaunch: (project: ProjectView) => void;
  onHydrate: (project: ProjectView) => void;
  onSelect: (project: ProjectView) => void;
  onOpenLocalPort?: (port: number) => void;
  onStopProcess?: (project: ProjectView) => void;
}

export function ProjectPrimaryActionButton({
  project,
  runningProc,
  isBusy,
  stopPropagation = false,
  size = "card",
  onLaunch,
  onHydrate,
  onSelect,
  onOpenLocalPort,
  onStopProcess,
}: ProjectPrimaryActionButtonProps) {
  const wrapClick = (fn: () => void) => (e: React.MouseEvent) => {
    if (stopPropagation) e.stopPropagation();
    fn();
  };

  const primaryTextSize = size === "table" ? "text-xs" : "text-[13px]";
  const primaryLayout = size === "table" ? "inline-flex" : "flex";
  const isRunning = Boolean(runningProc && runningProc.pid > 0);
  const firstPort = runningProc?.ports?.[0];

  if (isRunning) {
    return (
      <div className="inline-flex items-center gap-1.5 shrink-0">
        {firstPort && onOpenLocalPort ? (
          <button
            type="button"
            onClick={wrapClick(() => onOpenLocalPort(firstPort))}
            title={`Mở http://localhost:${firstPort}`}
            className={`h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-ready/60 bg-status-ready/20 hover:bg-status-ready/30 text-[#34D399] ${primaryTextSize} font-semibold ${primaryLayout} items-center gap-1.5 transition-colors`}
          >
            <ExternalLink className="w-3.5 h-3.5 shrink-0" />
            <span className="font-mono">Mở :{firstPort}</span>
          </button>
        ) : null}
        {onStopProcess && (
          <button
            type="button"
            disabled={isBusy}
            onClick={wrapClick(() => onStopProcess(project))}
            title="Dừng tiến trình đang chạy"
            aria-label="Dừng tiến trình"
            className={`h-8 ${firstPort && onOpenLocalPort ? "px-2.5" : "px-3"} shrink-0 whitespace-nowrap rounded-md border border-status-danger/60 bg-status-danger/15 hover:bg-status-danger/25 text-[#F87171] ${primaryTextSize} font-semibold ${primaryLayout} items-center gap-1.5 disabled:opacity-50 transition-colors`}
          >
            <Square className="w-3 h-3 shrink-0 fill-current" />
            <span>Dừng</span>
          </button>
        )}
      </div>
    );
  }

  if (project.status === "cloud_only") {
    return (
      <button
        type="button"
        disabled={isBusy}
        onClick={wrapClick(() => onHydrate(project))}
        title={
          size === "table"
            ? project.type === "desktop"
              ? "Install về máy"
              : "Clone về máy"
            : undefined
        }
        className={`h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas ${primaryTextSize} font-semibold shadow-[inset_0_1px_0_rgba(255,255,255,0.55)] ${primaryLayout} items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150`}
      >
        {isBusy ? (
          <Loader2 className="w-3.5 h-3.5 shrink-0 animate-spin" />
        ) : (
          <Download className="w-3.5 h-3.5 shrink-0" />
        )}
        <span>{project.type === "desktop" ? "Install" : "Clone"}</span>
      </button>
    );
  }

  if (project.status === "missing_env") {
    return (
      <button
        type="button"
        disabled={isBusy}
        onClick={wrapClick(() => onLaunch(project))}
        title={size === "table" ? "Khởi tạo / cấu hình .env" : undefined}
        className={`h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-warn/70 bg-status-warn/20 hover:bg-status-warn/30 text-[#FBBF24] text-[12px] font-semibold ${primaryLayout} items-center gap-1.5 disabled:opacity-50 transition-colors`}
      >
        <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
        <span>Cấu hình .env</span>
      </button>
    );
  }

  if (project.status === "missing_dir") {
    return (
      <button
        type="button"
        onClick={wrapClick(() => onSelect(project))}
        title={
          size === "table"
            ? "Thư mục cục bộ không tồn tại — mở chi tiết để cập nhật"
            : undefined
        }
        className={`h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-danger/70 bg-status-danger/15 hover:bg-status-danger/25 text-[#F87171] text-[12px] font-semibold ${primaryLayout} items-center gap-1.5 transition-colors`}
      >
        <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
        <span>Kiểm tra thư mục</span>
      </button>
    );
  }

  return (
    <button
      type="button"
      disabled={isBusy}
      onClick={wrapClick(() => onLaunch(project))}
      title={size === "table" ? "Khởi chạy" : undefined}
      className={`h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas ${primaryTextSize} font-semibold shadow-[inset_0_1px_0_rgba(255,255,255,0.55)] ${primaryLayout} items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150`}
    >
      {isBusy ? (
        <Loader2 className="w-3.5 h-3.5 shrink-0 animate-spin" />
      ) : (
        <Play className="w-3.5 h-3.5 shrink-0 fill-current" />
      )}
      <span>{project.type === "web" ? "Run .bat" : "Launch"}</span>
    </button>
  );
}


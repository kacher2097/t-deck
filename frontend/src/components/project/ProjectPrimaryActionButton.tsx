import { AlertTriangle, Download, Loader2, Play } from "lucide-react";
import type { ProjectView } from "../../types/project";

export const PROJECT_STATUS_META: Record<
  ProjectView["status"],
  { label: string; dotClass: string; textClass: string }
> = {
  ready: {
    label: "READY",
    dotClass: "bg-status-ready",
    textClass: "text-status-ready",
  },
  missing_env: {
    label: "MISSING .ENV",
    dotClass: "bg-status-warn",
    textClass: "text-status-warn",
  },
  cloud_only: {
    label: "CLOUD ONLY",
    dotClass: "bg-status-cloud",
    textClass: "text-muted",
  },
  missing_dir: {
    label: "MISSING DIR",
    dotClass: "bg-status-danger",
    textClass: "text-status-danger",
  },
};

interface ProjectPrimaryActionButtonProps {
  project: ProjectView;
  isBusy: boolean;
  stopPropagation?: boolean;
  size?: "card" | "table";
  onLaunch: (project: ProjectView) => void;
  onHydrate: (project: ProjectView) => void;
  onSelect: (project: ProjectView) => void;
}

export function ProjectPrimaryActionButton({
  project,
  isBusy,
  stopPropagation = false,
  size = "card",
  onLaunch,
  onHydrate,
  onSelect,
}: ProjectPrimaryActionButtonProps) {
  const wrapClick = (fn: () => void) => (e: React.MouseEvent) => {
    if (stopPropagation) e.stopPropagation();
    fn();
  };

  const primaryTextSize = size === "table" ? "text-xs" : "text-[13px]";
  const primaryLayout = size === "table" ? "inline-flex" : "flex";

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
        className={`h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas ${primaryTextSize} font-semibold ${primaryLayout} items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150`}
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
        className={`h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-warn/60 bg-status-warn/15 hover:bg-status-warn/25 text-status-warn text-[12px] font-medium ${primaryLayout} items-center gap-1.5 disabled:opacity-50 transition-colors`}
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
        className={`h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-danger/60 bg-status-danger/10 hover:bg-status-danger/20 text-status-danger text-[12px] font-medium ${primaryLayout} items-center gap-1.5 transition-colors`}
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
      className={`h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas ${primaryTextSize} font-semibold ${primaryLayout} items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150`}
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


import {
  AlertTriangle,
  Check,
  Code2,
  Download,
  FolderOpen,
  Globe,
  ImagePlus,
  Loader2,
  Play,
  SlidersHorizontal,
  Square,
  Trash2,
  Upload,
} from "lucide-react";
import type React from "react";
import { memo, useRef, useState } from "react";
import {
  extractPastedImageAsPNG,
  imageFileToPNGDataURL,
} from "../lib/clipboard";
import type { ProjectView, RunningProcess } from "../types/project";

interface ProjectCardProps {
  project: ProjectView;
  runningProc?: RunningProcess;
  busyAction: string | null;
  selectionMode?: boolean;
  selected?: boolean;
  onToggleSelect?: (project: ProjectView) => void;
  onRequestDelete?: (project: ProjectView) => void;
  onSelect: (project: ProjectView) => void;
  onLaunch: (project: ProjectView) => void;
  onHydrate: (project: ProjectView) => void;
  onOpenLive?: (project: ProjectView) => void;
  onStopProcess?: (project: ProjectView) => void;
  onOpenLocalPort?: (port: number) => void;
  onOpenExplorer: (project: ProjectView) => void;
  onOpenVSCode: (project: ProjectView) => void;
  onUploadThumbnail: (project: ProjectView, pngDataUrl: string) => Promise<void>;
  onUploadError: (msg: string) => void;
  onTagClick: (tag: string) => void;
  onCategoryClick: (category: string) => void;
}

const STATUS_META: Record<
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

export const ProjectCard = memo(function ProjectCard({
  project,
  runningProc,
  busyAction,
  selectionMode = false,
  selected = false,
  onToggleSelect,
  onRequestDelete,
  onSelect,
  onLaunch,
  onHydrate,
  onOpenLive,
  onStopProcess,
  onOpenLocalPort,
  onOpenExplorer,
  onOpenVSCode,
  onUploadThumbnail,
  onUploadError,
  onTagClick,
  onCategoryClick,
}: ProjectCardProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const statusMeta = STATUS_META[project.status] ?? STATUS_META.ready;
  const isBusy = busyAction === project.id;
  const isRunning = Boolean(runningProc && runningProc.pid > 0);
  const firstPort = runningProc?.ports?.[0];

  const visibleTags = project.tags?.slice(0, 3) || [];
  const extraTagCount = Math.max(0, (project.tags?.length || 0) - 3);

  const savePNG = async (dataUrl: string) => {
    setUploading(true);
    try {
      await onUploadThumbnail(project, dataUrl);
    } finally {
      setUploading(false);
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    try {
      const dataUrl = await extractPastedImageAsPNG(e);
      if (!dataUrl) return;
      e.preventDefault();
      e.stopPropagation();
      await savePNG(dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    try {
      const dataUrl = await imageFileToPNGDataURL(file);
      await savePNG(dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    try {
      const dataUrl = await imageFileToPNGDataURL(file);
      await savePNG(dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
  };

  const renderPrimaryButton = () => {
    if (project.status === "cloud_only") {
      return (
        <button
          type="button"
          disabled={isBusy}
          onClick={(e) => {
            e.stopPropagation();
            onHydrate(project);
          }}
          className="h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas text-[13px] font-semibold flex items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150"
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
          onClick={(e) => {
            e.stopPropagation();
            onLaunch(project);
          }}
          className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-warn/60 bg-status-warn/15 hover:bg-status-warn/25 text-status-warn text-[12px] font-medium flex items-center gap-1.5 disabled:opacity-50 transition-colors"
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
          onClick={(e) => {
            e.stopPropagation();
            onSelect(project);
          }}
          className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-danger/60 bg-status-danger/10 hover:bg-status-danger/20 text-status-danger text-[12px] font-medium flex items-center gap-1.5 transition-colors"
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
        onClick={(e) => {
          e.stopPropagation();
          onLaunch(project);
        }}
        className="h-8 px-3 shrink-0 whitespace-nowrap rounded-md bg-accent hover:bg-white text-canvas text-[13px] font-semibold flex items-center gap-1.5 disabled:opacity-50 active:scale-[0.98] transition-[background-color,transform] duration-150"
      >
        {isBusy ? (
          <Loader2 className="w-3.5 h-3.5 shrink-0 animate-spin" />
        ) : (
          <Play className="w-3.5 h-3.5 shrink-0 fill-current" />
        )}
        <span>{project.type === "web" ? "Run .bat" : "Launch"}</span>
      </button>
    );
  };

  return (
    <article
      onPaste={handlePaste}
      className={`bg-surface border rounded-md transition-colors flex flex-col overflow-hidden group min-w-0 ${
        selected
          ? "border-accent ring-1 ring-accent/60 bg-elevated/40"
          : "border-edge hover:bg-elevated/70 hover:border-muted/40"
      }`}
    >
      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,image/bmp"
        onChange={handleFileChange}
        className="hidden"
      />

      {/* Dedicated Card Header Strip (keeps thumbnail 100% unobstructed) */}
      <div className="h-9 px-3 border-b border-edge bg-canvas/50 flex items-center justify-between gap-2 shrink-0 min-w-0">
        <div className="flex items-center gap-1.5 min-w-0">
          {onToggleSelect && (selectionMode || selected) && (
            <button
              type="button"
              title={selected ? "Bỏ chọn" : "Chọn để xóa hàng loạt"}
              aria-label={selected ? "Bỏ chọn dự án" : "Chọn dự án"}
              aria-pressed={selected}
              onClick={(e) => {
                e.stopPropagation();
                onToggleSelect(project);
              }}
              className={`w-4 h-4 shrink-0 rounded border flex items-center justify-center transition-colors ${
                selected
                  ? "bg-accent border-accent text-canvas"
                  : "bg-surface border-muted/70 text-transparent hover:border-accent"
              }`}
            >
              <Check className="w-3 h-3 stroke-[3]" />
            </button>
          )}
          <span className="px-2 py-0.5 rounded bg-surface border border-edge text-[11px] font-mono font-semibold text-primary shrink-0 whitespace-nowrap">
            {project.type === "web"
              ? "WEB"
              : project.type === "hybrid"
              ? "HYBRID"
              : "EXE"}
          </span>
          {project.latest_version && (
            <span className="px-2 py-0.5 rounded bg-surface border border-edge text-[11px] font-mono text-muted shrink-0 whitespace-nowrap">
              {project.latest_version}
            </span>
          )}
          {project.has_update && (
            <span className="px-2 py-0.5 rounded bg-status-warn/20 border border-status-warn/60 text-[11px] font-mono font-semibold text-status-warn shrink-0 whitespace-nowrap">
              UPDATE
            </span>
          )}
        </div>

        <div className="px-2 py-0.5 rounded bg-surface border border-edge flex items-center gap-1.5 shrink-0 whitespace-nowrap">
          <span
            className={`w-1.5 h-1.5 rounded-full shrink-0 ${
              isRunning ? "bg-status-ready animate-pulse" : statusMeta.dotClass
            }`}
          />
          <span
            className={`text-[11px] font-mono ${
              isRunning ? "text-status-ready font-semibold" : statusMeta.textClass
            }`}
          >
            {isRunning
              ? firstPort
                ? `RUNNING :${firstPort}`
                : "RUNNING"
              : statusMeta.label}
          </span>
        </div>
      </div>

      {/* Unobstructed 16:9 Preview Area — Upload file, Drag & Drop, or Click & Ctrl+V */}
      <div
        tabIndex={0}
        role="button"
        aria-label={`Ảnh preview ${project.name}. Kéo thả ảnh, chọn file hoặc click và nhấn Ctrl+V`}
        onClick={(e) => e.currentTarget.focus()}
        onDoubleClick={() => onSelect(project)}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={`relative aspect-video bg-canvas border-b overflow-hidden focus:outline-none focus:ring-1 focus:ring-inset focus:ring-accent/70 cursor-pointer shrink-0 ${
          dragging ? "border-accent bg-elevated/60" : "border-edge"
        }`}
      >
        {project.thumbnail_url ? (
          <img
            src={project.thumbnail_url}
            alt={project.name}
            loading="lazy"
            decoding="async"
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full flex flex-col items-center justify-center gap-2 text-muted/80 px-4 text-center">
            <ImagePlus className="w-5 h-5" />
            <span className="text-xs">
              Kéo thả,{" "}
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  fileInputRef.current?.click();
                }}
                className="underline text-primary hover:text-white"
              >
                tải ảnh
              </button>{" "}
              hoặc Ctrl+V
            </span>
          </div>
        )}

        {/* Quick Upload Trigger on Hover */}
        <button
          type="button"
          title="Tải ảnh từ máy"
          onClick={(e) => {
            e.stopPropagation();
            fileInputRef.current?.click();
          }}
          className="absolute bottom-2 right-2 h-7 px-2.5 rounded bg-canvas/90 border border-edge text-xs font-medium text-primary hover:bg-elevated flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <Upload className="w-3 h-3" />
          <span>Tải ảnh</span>
        </button>

        {uploading && (
          <div className="absolute inset-0 bg-canvas/85 flex items-center justify-center gap-2 text-xs text-primary">
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
            <span>Đang lưu ảnh...</span>
          </div>
        )}
      </div>

      {/* Uniform Card Body */}
      <div
        onClick={() =>
          selectionMode && onToggleSelect
            ? onToggleSelect(project)
            : onSelect(project)
        }
        className="p-3.5 flex-1 flex flex-col gap-2 cursor-pointer min-w-0"
      >
        <div className="flex items-center justify-between gap-2 min-w-0">
          <h3
            title={project.name}
            className="font-heading text-[15px] font-semibold text-primary leading-snug truncate min-w-0 flex-1"
          >
            {project.name}
          </h3>
          {project.category && (
            <button
              type="button"
              title={project.category}
              onClick={(e) => {
                e.stopPropagation();
                onCategoryClick(project.category);
              }}
              className="max-w-[46%] truncate px-2 py-0.5 rounded bg-canvas border border-edge hover:border-muted text-[11px] font-medium text-muted hover:text-primary shrink-0 transition-colors"
            >
              {project.category}
            </button>
          )}
        </div>

        <p className="text-[13px] text-muted line-clamp-2 h-[2.6rem] leading-relaxed">
          {project.summary || (
            <span className="italic text-muted/60">
              Chưa có mô tả chức năng (mở chi tiết để bổ sung)
            </span>
          )}
        </p>

        {/* Single-row Tags (prevents uneven multi-line card heights) */}
        <div className="mt-auto pt-1 h-7 flex items-center gap-1.5 overflow-hidden flex-nowrap">
          {visibleTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onTagClick(tag);
              }}
              className="px-2 py-0.5 rounded bg-canvas border border-edge hover:border-muted text-xs text-muted hover:text-primary shrink-0 whitespace-nowrap transition-colors"
            >
              #{tag}
            </button>
          ))}
          {extraTagCount > 0 && (
            <span className="px-1.5 py-0.5 text-[11px] font-mono text-muted shrink-0 whitespace-nowrap">
              +{extraTagCount}
            </span>
          )}
        </div>
      </div>

      {/* Overflow-Proof Action Footer */}
      <div className="h-[50px] px-3.5 border-t border-edge bg-canvas/40 flex items-center justify-between gap-2 shrink-0 min-w-0">
        {renderPrimaryButton()}

        <div className="flex items-center gap-1 shrink-0">
          {isRunning && firstPort && onOpenLocalPort && (
            <button
              type="button"
              title={`Mở http://localhost:${firstPort}`}
              onClick={(e) => {
                e.stopPropagation();
                onOpenLocalPort(firstPort);
              }}
              className="h-7 px-2 shrink-0 whitespace-nowrap rounded-md border border-status-ready/50 bg-status-ready/10 text-status-ready text-[11px] font-mono font-semibold hover:bg-status-ready/20 transition-colors"
            >
              :{firstPort}
            </button>
          )}
          {isRunning && onStopProcess && (
            <button
              type="button"
              title="Dừng tiến trình đang chạy"
              aria-label="Dừng tiến trình"
              onClick={(e) => {
                e.stopPropagation();
                onStopProcess(project);
              }}
              className="h-7 w-7 shrink-0 rounded-md border border-status-danger/50 bg-status-danger/10 text-status-danger hover:bg-status-danger/20 flex items-center justify-center transition-colors"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
            </button>
          )}
          {project.live_url && onOpenLive && (
            <button
              type="button"
              title={`Mở Live Web (${project.live_url})`}
              aria-label="Mở Live Web"
              onClick={(e) => {
                e.stopPropagation();
                onOpenLive(project);
              }}
              className="h-7 w-7 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated text-status-ready hover:text-white flex items-center justify-center transition-colors"
            >
              <Globe className="w-3.5 h-3.5" />
            </button>
          )}
          {project.local_dir && project.status !== "cloud_only" && (
            <>
              <button
                type="button"
                title="Mở thư mục trong Explorer"
                aria-label="Mở thư mục"
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenExplorer(project);
                }}
                className="h-7 w-7 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated text-muted hover:text-primary flex items-center justify-center transition-colors"
              >
                <FolderOpen className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                title="Mở bằng VS Code"
                aria-label="Mở VS Code"
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenVSCode(project);
                }}
                className="h-7 w-7 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated text-muted hover:text-primary flex items-center justify-center transition-colors"
              >
                <Code2 className="w-3.5 h-3.5" />
              </button>
            </>
          )}
          <button
            type="button"
            title="Chi tiết & Cấu hình"
            aria-label="Mở chi tiết"
            onClick={(e) => {
              e.stopPropagation();
              onSelect(project);
            }}
            className="h-7 w-7 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated text-muted hover:text-primary flex items-center justify-center transition-colors"
          >
            <SlidersHorizontal className="w-3.5 h-3.5" />
          </button>
          {onRequestDelete && (
            <button
              type="button"
              title="Xóa dự án này"
              aria-label={`Xóa ${project.name}`}
              onClick={(e) => {
                e.stopPropagation();
                onRequestDelete(project);
              }}
              className="h-7 w-7 shrink-0 rounded-md border border-edge bg-surface hover:border-status-danger/60 hover:bg-status-danger/10 text-muted hover:text-status-danger flex items-center justify-center transition-colors"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>
    </article>
  );
});

import {
  Check,
  Code2,
  FolderOpen,
  Globe,
  SlidersHorizontal,
  Square,
  Trash2,
} from "lucide-react";
import type React from "react";
import { memo } from "react";
import { extractPastedImageAsPNG } from "../lib/clipboard";
import type { ProjectView, RunningProcess } from "../types/project";
import { ProjectCardPreview } from "./project/ProjectCardPreview";
import {
  PROJECT_STATUS_META,
  ProjectPrimaryActionButton,
} from "./project/ProjectPrimaryActionButton";

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
  const statusMeta = PROJECT_STATUS_META[project.status] ?? PROJECT_STATUS_META.ready;
  const isBusy = busyAction === project.id;
  const isRunning = Boolean(runningProc && runningProc.pid > 0);
  const firstPort = runningProc?.ports?.[0];

  const visibleTags = project.tags?.slice(0, 3) || [];
  const extraTagCount = Math.max(0, (project.tags?.length || 0) - 3);

  const handlePaste = async (e: React.ClipboardEvent) => {
    try {
      const dataUrl = await extractPastedImageAsPNG(e);
      if (!dataUrl) return;
      e.preventDefault();
      e.stopPropagation();
      await onUploadThumbnail(project, dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
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
      {/* Dedicated Card Header Strip */}
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

      <ProjectCardPreview
        project={project}
        onOpenDetail={() => onSelect(project)}
        onUploadThumbnail={onUploadThumbnail}
        onUploadError={onUploadError}
      />

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
          <div className="flex items-center gap-2 min-w-0 flex-1">
            {project.icon_url && (
              <img
                src={project.icon_url}
                alt=""
                loading="lazy"
                decoding="async"
                className="w-5 h-5 rounded-[5px] object-cover border border-edge bg-canvas shrink-0"
              />
            )}
            <h3
              title={project.name}
              className="font-heading text-[15px] font-semibold text-primary leading-snug truncate min-w-0 flex-1"
            >
              {project.name}
            </h3>
          </div>
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
        <ProjectPrimaryActionButton
          project={project}
          isBusy={isBusy}
          stopPropagation
          size="card"
          onLaunch={onLaunch}
          onHydrate={onHydrate}
          onSelect={onSelect}
        />

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

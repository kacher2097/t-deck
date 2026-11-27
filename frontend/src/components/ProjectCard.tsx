import {
  Check,
  Code2,
  FolderOpen,
  Globe,
  MoreHorizontal,
  SlidersHorizontal,
  TerminalSquare,
  Trash2,
} from "lucide-react";
import type React from "react";
import { memo, useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
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

  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const handleOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMenuOpen(false);
    };
    document.addEventListener("mousedown", handleOutside);
    window.addEventListener("keydown", handleEsc);
    return () => {
      document.removeEventListener("mousedown", handleOutside);
      window.removeEventListener("keydown", handleEsc);
    };
  }, [menuOpen]);

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

  const hasLocalTools = Boolean(project.local_dir && project.status !== "cloud_only");
  const typeLabel =
    project.type === "web" ? "WEB" : project.type === "hybrid" ? "HYBRID" : "EXE";

  return (
    <article
      onPaste={handlePaste}
      className={`relative bg-surface border rounded-lg transition-[transform,border-color,background-color,box-shadow] duration-200 ease-out flex flex-col group min-w-0 ${
        menuOpen ? "z-30" : ""
      } ${
        selected
          ? "border-white bg-elevated/60 ring-1 ring-white/50 shadow-[0_12px_30px_-6px_rgba(0,0,0,0.75),0_0_20px_-2px_rgba(255,255,255,0.22)]"
          : isRunning
            ? "border-status-ready/50 shadow-[0_0_20px_-6px_rgba(16,185,129,0.22)] hover:border-white hover:ring-1 hover:ring-white/55 hover:bg-[#1A1E27] hover:-translate-y-[3px] hover:shadow-[0_16px_36px_-6px_rgba(0,0,0,0.88),0_0_26px_-2px_rgba(255,255,255,0.32)]"
            : "border-edge hover:border-white hover:ring-1 hover:ring-white/55 hover:bg-[#1A1E27] hover:-translate-y-[3px] hover:shadow-[0_16px_36px_-6px_rgba(0,0,0,0.88),0_0_26px_-2px_rgba(255,255,255,0.30)]"
      }`}
    >
      {/* Dedicated Card Header Strip */}
      <div className="h-9 px-3 rounded-t-lg border-b border-edge/80 bg-canvas/60 flex items-center justify-between gap-2 shrink-0 min-w-0">
        <div className="flex items-center gap-2 min-w-0">
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
                  : "bg-surface border-muted/70 text-transparent hover:border-white"
              }`}
            >
              <Check className="w-3 h-3 stroke-[3]" />
            </button>
          )}
          <span className="text-[11px] font-mono font-semibold tracking-wide text-muted/85 group-hover:text-white shrink-0 whitespace-nowrap transition-colors duration-150">
            {typeLabel}
            {project.latest_version ? ` · ${project.latest_version}` : ""}
          </span>
          {project.has_update && (
            <span className="px-1.5 py-0.5 rounded bg-status-warn/20 border border-status-warn/60 text-[10.5px] font-mono font-semibold text-[#FBBF24] shrink-0 whitespace-nowrap">
              UPDATE
            </span>
          )}
        </div>

        <div
          className={`px-2 py-0.5 rounded-full flex items-center gap-1.5 shrink-0 whitespace-nowrap ${
            isRunning
              ? "bg-status-ready/15 border border-status-ready/50 text-[#34D399]"
              : statusMeta.pillClass
          }`}
        >
          <span
            className={`w-1.5 h-1.5 rounded-full shrink-0 ${
              isRunning ? "bg-status-ready animate-pulse" : statusMeta.dotClass
            }`}
          />
          <span className="text-[10.5px] font-mono font-semibold tracking-wide">
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
              className="font-heading text-[15.5px] font-semibold text-primary group-hover:text-[#34D399] leading-snug truncate min-w-0 flex-1 transition-colors duration-150"
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
              className="max-w-[44%] truncate text-[11px] font-medium text-muted/80 group-hover:text-gray-200 hover:!text-white hover:underline underline-offset-2 shrink-0 transition-colors duration-150"
            >
              {project.category}
            </button>
          )}
        </div>

        <p className="text-[13px] text-muted/90 group-hover:text-gray-200 line-clamp-2 h-[2.6rem] leading-relaxed transition-colors duration-150">
          {project.summary || (
            <span className="italic text-muted/60">
              Chưa có mô tả chức năng (mở chi tiết để bổ sung)
            </span>
          )}
        </p>

        <div className="mt-auto pt-1 h-6 flex items-center gap-2.5 overflow-hidden flex-nowrap">
          {visibleTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onTagClick(tag);
              }}
              className="text-[11.5px] text-muted/70 group-hover:text-[#6EE7B7]/85 hover:!text-white shrink-0 whitespace-nowrap transition-colors duration-150"
            >
              #{tag}
            </button>
          ))}
          {extraTagCount > 0 && (
            <span className="text-[11px] font-mono text-muted/70 shrink-0 whitespace-nowrap">
              +{extraTagCount}
            </span>
          )}
        </div>
      </div>

      {/* Miller's Law Action Footer (<= 4 controls + overflow popover) */}
      <div className="h-[50px] px-3.5 rounded-b-lg border-t border-edge/80 bg-canvas/45 flex items-center justify-between gap-2 shrink-0 min-w-0">
        <ProjectPrimaryActionButton
          project={project}
          runningProc={runningProc}
          isBusy={isBusy}
          stopPropagation
          size="card"
          onLaunch={onLaunch}
          onHydrate={onHydrate}
          onSelect={onSelect}
          onOpenLocalPort={onOpenLocalPort}
          onStopProcess={onStopProcess}
        />

        <div className="flex items-center gap-1 shrink-0">
          {hasLocalTools && (
            <>
              <button
                type="button"
                title="Mở thư mục trong Explorer"
                aria-label="Mở thư mục"
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenExplorer(project);
                }}
                className="h-8 w-8 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated hover:border-white/25 text-muted hover:text-white flex items-center justify-center transition-colors"
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
                className="h-8 w-8 shrink-0 rounded-md border border-edge bg-surface hover:bg-elevated hover:border-white/25 text-muted hover:text-white flex items-center justify-center transition-colors"
              >
                <Code2 className="w-3.5 h-3.5" />
              </button>
            </>
          )}

          <div ref={menuRef} className="relative">
            <button
              type="button"
              title="Thao tác khác"
              aria-label={`Thêm thao tác cho ${project.name}`}
              aria-expanded={menuOpen}
              aria-haspopup="menu"
              onClick={(e) => {
                e.stopPropagation();
                setMenuOpen((prev) => !prev);
              }}
              className={`h-8 w-8 shrink-0 rounded-md border flex items-center justify-center transition-colors ${
                menuOpen
                  ? "border-white/40 bg-elevated text-white"
                  : "border-edge bg-surface hover:bg-elevated hover:border-white/25 text-muted hover:text-white"
              }`}
            >
              <MoreHorizontal className="w-4 h-4" />
            </button>

            {menuOpen && (
              <div
                role="menu"
                onClick={(e) => e.stopPropagation()}
                className="absolute right-0 bottom-full mb-1.5 w-52 rounded-lg bg-[#1A1D24] border border-white/15 py-1 shadow-[0_16px_40px_rgba(0,0,0,0.85)] z-30"
              >
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false);
                    onSelect(project);
                  }}
                  className="w-full px-3 py-2 text-left text-[13px] text-primary hover:bg-white/[0.08] flex items-center gap-2.5 transition-colors"
                >
                  <SlidersHorizontal className="w-3.5 h-3.5 text-muted shrink-0" />
                  <span>Chi tiết &amp; Cấu hình</span>
                </button>

                {hasLocalTools && (
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => {
                      setMenuOpen(false);
                      void api
                        .openInEditor(project.id, "terminal")
                        .catch((err) =>
                          onUploadError(
                            err instanceof Error ? err.message : String(err)
                          )
                        );
                    }}
                    className="w-full px-3 py-2 text-left text-[13px] text-primary hover:bg-white/[0.08] flex items-center gap-2.5 transition-colors"
                  >
                    <TerminalSquare className="w-3.5 h-3.5 text-muted shrink-0" />
                    <span>Mở Terminal</span>
                  </button>
                )}

                {project.live_url && onOpenLive && (
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => {
                      setMenuOpen(false);
                      onOpenLive(project);
                    }}
                    className="w-full px-3 py-2 text-left text-[13px] text-[#34D399] hover:bg-white/[0.08] flex items-center gap-2.5 transition-colors"
                  >
                    <Globe className="w-3.5 h-3.5 shrink-0" />
                    <span>Mở Live Web</span>
                  </button>
                )}

                {onRequestDelete && (
                  <>
                    <div className="my-1 border-t border-white/10" />
                    <button
                      type="button"
                      role="menuitem"
                      onClick={() => {
                        setMenuOpen(false);
                        onRequestDelete(project);
                      }}
                      className="w-full px-3 py-2 text-left text-[13px] text-[#F87171] hover:bg-status-danger/15 flex items-center gap-2.5 transition-colors"
                    >
                      <Trash2 className="w-3.5 h-3.5 shrink-0" />
                      <span>Gỡ khỏi danh sách</span>
                    </button>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </article>
  );
});

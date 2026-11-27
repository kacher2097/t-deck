import {
  Check,
  Code2,
  FolderOpen,
  Globe,
  Link2,
  RefreshCw,
  SlidersHorizontal,
  Square,
  Trash2,
} from "lucide-react";
import type {
  ProjectView,
  RunningProcess,
  WebHealthResult,
} from "../../types/project";
import {
  PROJECT_STATUS_META,
  ProjectPrimaryActionButton,
} from "./ProjectPrimaryActionButton";

interface ReleaseMatrixRowProps {
  project: ProjectView;
  healthMap: Record<string, WebHealthResult>;
  running?: RunningProcess;
  isBusy: boolean;
  selectionMode: boolean;
  isSelected: boolean;
  onToggleSelect?: (p: ProjectView) => void;
  onRequestDelete?: (p: ProjectView) => void;
  onSelect: (p: ProjectView) => void;
  onLaunch: (p: ProjectView) => void;
  onHydrate: (p: ProjectView) => void;
  onUpdateDesktop: (p: ProjectView) => void;
  onOpenLive: (p: ProjectView) => void;
  onOpenLocalPort?: (port: number) => void;
  onStopProcess: (p: ProjectView) => void;
  onCopyReleaseLink: (releaseId: string) => void;
  onOpenExplorer: (p: ProjectView) => void;
  onOpenVSCode: (p: ProjectView) => void;
}

export function ReleaseMatrixRow({
  project: p,
  healthMap,
  running,
  isBusy,
  selectionMode,
  isSelected,
  onToggleSelect,
  onRequestDelete,
  onSelect,
  onLaunch,
  onHydrate,
  onUpdateDesktop,
  onOpenLive,
  onOpenLocalPort,
  onStopProcess,
  onCopyReleaseLink,
  onOpenExplorer,
  onOpenVSCode,
}: ReleaseMatrixRowProps) {
  const webTarget = (p.targets || []).find((t) => t.kind === "web");
  const deskTarget = (p.targets || []).find((t) => t.kind === "desktop");
  const webRel = webTarget ? p.latest_releases?.[webTarget.id] : undefined;
  const deskRel = deskTarget ? p.latest_releases?.[deskTarget.id] : undefined;
  const webHealth = webTarget ? healthMap[webTarget.id] : undefined;
  const isRunning = Boolean(running && running.pid > 0);
  const firstPort = running?.ports?.[0];
  const statusMeta = PROJECT_STATUS_META[p.status] ?? PROJECT_STATUS_META.ready;

  const rowBg = isSelected ? "bg-elevated" : "bg-surface group-hover:bg-elevated";

  const hasWebTarget = Boolean(webTarget || p.live_url);
  const hasDeskInfo = Boolean(
    deskTarget ||
      p.type !== "web" ||
      deskRel ||
      p.latest_version ||
      p.installed_version ||
      p.has_update
  );

  return (
    <tr
      onClick={() =>
        selectionMode && onToggleSelect ? onToggleSelect(p) : onSelect(p)
      }
      className={`group cursor-pointer transition-colors ${rowBg}`}
    >
      {selectionMode && onToggleSelect && (
        <td
          className={`py-2.5 pl-3.5 pr-1 w-9 border-b border-edge transition-colors ${rowBg}`}
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelect(p);
          }}
        >
          <button
            type="button"
            aria-label={isSelected ? "Bỏ chọn" : "Chọn"}
            className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
              isSelected
                ? "bg-accent border-accent text-canvas"
                : "bg-canvas border-edge text-transparent hover:border-muted"
            }`}
          >
            <Check className="w-3 h-3 stroke-[3]" />
          </button>
        </td>
      )}

      {/* 1. Dự án */}
      <td
        className={`py-2.5 px-3.5 min-w-[210px] max-w-[310px] border-b border-edge transition-colors ${rowBg}`}
      >
        <div className="flex items-center gap-2.5 min-w-0">
          {p.thumbnail_url ? (
            <img
              src={p.thumbnail_url}
              alt={p.name}
              loading="lazy"
              decoding="async"
              className="w-12 h-7 rounded object-cover border border-edge shrink-0 bg-canvas"
            />
          ) : (
            <div className="w-12 h-7 rounded border border-edge bg-canvas shrink-0 flex items-center justify-center text-[10px] font-mono text-muted">
              {p.type.toUpperCase()}
            </div>
          )}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5 min-w-0">
              {p.icon_url && (
                <img
                  src={p.icon_url}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  className="w-4 h-4 rounded-[4px] object-cover border border-edge bg-canvas shrink-0"
                />
              )}
              <span title={p.name} className="font-semibold text-primary truncate">
                {p.name}
              </span>
              <span className="px-1.5 py-0.2 rounded bg-canvas border border-edge text-[10px] font-mono text-muted shrink-0">
                {p.type === "hybrid"
                  ? "HYBRID"
                  : p.type === "web"
                  ? "WEB"
                  : "EXE"}
              </span>
              {p.has_update && (
                <span className="px-1.5 py-0.2 rounded bg-status-warn/15 border border-status-warn/50 text-[10px] font-mono font-semibold text-status-warn shrink-0">
                  UPDATE
                </span>
              )}
            </div>
            <div className="text-[11px] text-muted truncate">
              {p.category || "Chưa phân nhóm"}
            </div>
          </div>
        </div>
      </td>

      {/* 2. Trạng thái & Git */}
      <td
        className={`py-2.5 px-3 min-w-[140px] max-w-[210px] border-b border-edge transition-colors ${rowBg}`}
      >
        <div className="space-y-1 min-w-0">
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-canvas border border-edge text-[11px] font-mono">
            <span
              className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                isRunning ? "bg-status-ready animate-pulse" : statusMeta.dotClass
              }`}
            />
            <span
              className={
                isRunning
                  ? "text-status-ready font-semibold"
                  : statusMeta.textClass
              }
            >
              {isRunning
                ? firstPort
                  ? `RUNNING :${firstPort}`
                  : "RUNNING"
                : statusMeta.label}
            </span>
          </span>
          {p.git_branch && (
            <div
              title={`${p.git_branch}${p.git_commit ? ` @ ${p.git_commit}` : ""}`}
              className="text-[11px] font-mono text-muted/85 truncate"
            >
              {p.git_branch}
              {p.git_commit ? ` @ ${p.git_commit}` : ""}
            </div>
          )}
        </div>
      </td>

      {/* 3. Phát hành & Live */}
      <td
        className={`py-2.5 px-3 min-w-[135px] max-w-[220px] border-b border-edge transition-colors ${rowBg}`}
      >
        {hasWebTarget || hasDeskInfo ? (
          <div className="space-y-0.5 min-w-0">
            {hasWebTarget && (
              <div className="flex items-center gap-1.5 min-w-0">
                {webHealth ? (
                  <span
                    className={`text-[11px] font-mono font-semibold shrink-0 ${
                      webHealth.online
                        ? webHealth.drift
                          ? "text-status-warn"
                          : "text-status-ready"
                        : "text-status-danger"
                    }`}
                  >
                    {webHealth.online ? `ON ${webHealth.latency_ms}ms` : "OFFLINE"}
                  </span>
                ) : (
                  <span className="text-[11px] font-mono text-muted shrink-0">
                    {webRel?.version || (p.live_url ? "Web Live" : "-")}
                  </span>
                )}
                {webHealth?.live_version && (
                  <span className="text-[11px] font-mono text-primary truncate">
                    {webHealth.live_version}
                  </span>
                )}
                {p.live_url && (
                  <span
                    title={p.live_url}
                    className="text-[11px] font-mono text-muted truncate"
                  >
                    {p.live_url.replace(/^https?:\/\//, "")}
                  </span>
                )}
              </div>
            )}
            {hasDeskInfo && (
              <div className="flex items-center gap-1.5 font-mono text-[11px] min-w-0">
                <span className="text-primary font-semibold shrink-0">
                  {deskRel?.version || p.latest_version || "-"}
                </span>
                {p.installed_version && (
                  <span
                    className={`truncate ${
                      p.has_update ? "text-status-warn" : "text-muted"
                    }`}
                  >
                    / local {p.installed_version}
                  </span>
                )}
                {p.has_update && (
                  <span className="px-1.5 py-0.2 rounded bg-status-warn/15 border border-status-warn/50 text-[10px] text-status-warn shrink-0">
                    Có bản mới
                  </span>
                )}
              </div>
            )}
          </div>
        ) : (
          <span className="inline-flex items-center px-2 py-0.5 rounded bg-canvas border border-edge text-[11px] font-mono text-muted">
            Local
          </span>
        )}
      </td>

      {/* 4. Thao tác */}
      <td
        className={`sticky right-0 z-10 whitespace-nowrap py-2.5 pl-3 pr-3.5 text-right border-b border-l border-edge/60 transition-colors ${rowBg}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="inline-flex items-center justify-end gap-1.5 shrink-0">
          <ProjectPrimaryActionButton
            project={p}
            isBusy={isBusy}
            size="table"
            onLaunch={onLaunch}
            onHydrate={onHydrate}
            onSelect={onSelect}
          />

          {isRunning && firstPort && onOpenLocalPort && (
            <button
              type="button"
              onClick={() => onOpenLocalPort(firstPort)}
              title={`Mở http://localhost:${firstPort}`}
              className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-ready/50 bg-status-ready/10 hover:bg-status-ready/20 text-status-ready text-xs font-mono font-semibold inline-flex items-center transition-colors"
            >
              :{firstPort}
            </button>
          )}

          {isRunning && (
            <button
              type="button"
              onClick={() => onStopProcess(p)}
              title="Dừng tiến trình đang chạy"
              aria-label="Dừng tiến trình"
              className="h-8 w-8 shrink-0 rounded-md border border-status-danger/60 bg-status-danger/10 text-status-danger hover:bg-status-danger/20 inline-flex items-center justify-center transition-colors"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
            </button>
          )}

          {p.has_update && (
            <button
              type="button"
              onClick={() => onUpdateDesktop(p)}
              title="Cập nhật Desktop App ngay"
              aria-label="Cập nhật Desktop App"
              className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-status-warn/60 bg-status-warn/10 text-status-warn hover:bg-status-warn/20 text-xs font-medium inline-flex items-center gap-1.5 transition-colors"
            >
              <RefreshCw className="w-3.5 h-3.5 shrink-0" />
              <span className="hidden xl:inline">Update</span>
            </button>
          )}

          {p.live_url && (
            <button
              type="button"
              onClick={() => onOpenLive(p)}
              title={`Mở Live Web (${p.live_url})`}
              aria-label="Mở Live Web"
              className="h-8 w-8 shrink-0 rounded-md border border-edge bg-canvas hover:bg-elevated text-status-ready inline-flex items-center justify-center transition-colors"
            >
              <Globe className="w-3.5 h-3.5" />
            </button>
          )}

          {deskRel && (
            <button
              type="button"
              onClick={() => onCopyReleaseLink(deskRel.id)}
              title="Copy Download Link"
              aria-label="Copy Download Link"
              className="h-8 w-8 shrink-0 rounded-md border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary inline-flex items-center justify-center transition-colors"
            >
              <Link2 className="w-3.5 h-3.5" />
            </button>
          )}

          {p.local_dir && p.status !== "cloud_only" && (
            <>
              <button
                type="button"
                onClick={() => onOpenExplorer(p)}
                title="Mở thư mục trong Explorer"
                aria-label="Mở thư mục"
                className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary text-xs font-medium inline-flex items-center gap-1.5 transition-colors"
              >
                <FolderOpen className="w-3.5 h-3.5 shrink-0" />
                <span className="hidden 2xl:inline">Explorer</span>
              </button>
              <button
                type="button"
                onClick={() => onOpenVSCode(p)}
                title="Mở bằng VS Code"
                aria-label="Mở VS Code"
                className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary text-xs font-medium inline-flex items-center gap-1.5 transition-colors"
              >
                <Code2 className="w-3.5 h-3.5 shrink-0" />
                <span className="hidden 2xl:inline">Code</span>
              </button>
            </>
          )}

          <button
            type="button"
            onClick={() => onSelect(p)}
            title="Chi tiết & Cấu hình"
            aria-label="Chi tiết"
            className="h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary text-xs font-medium inline-flex items-center gap-1.5 transition-colors"
          >
            <SlidersHorizontal className="w-3.5 h-3.5 shrink-0" />
            <span className="hidden xl:inline">Chi tiết</span>
          </button>

          {onRequestDelete && (
            <button
              type="button"
              onClick={() => onRequestDelete(p)}
              title="Xóa dự án"
              aria-label={`Xóa ${p.name}`}
              className="h-8 w-8 shrink-0 rounded-md border border-edge bg-canvas hover:border-status-danger/60 hover:bg-status-danger/10 text-muted hover:text-status-danger inline-flex items-center justify-center transition-colors"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </td>
    </tr>
  );
}


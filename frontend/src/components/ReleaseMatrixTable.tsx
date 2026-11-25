import {
  Check,
  Code2,
  Download,
  ExternalLink,
  FolderOpen,
  Globe,
  Link2,
  Play,
  RefreshCw,
  SlidersHorizontal,
  Square,
  Trash2,
} from "lucide-react";
import type {
  ProjectView,
  RunningProcess,
  WebHealthResult,
} from "../types/project";

interface ReleaseMatrixTableProps {
  projects: ProjectView[];
  healthMap: Record<string, WebHealthResult>;
  runningMap: Record<string, RunningProcess>;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelect?: (p: ProjectView) => void;
  onToggleSelectAll?: () => void;
  onRequestDelete?: (p: ProjectView) => void;
  onSelect: (p: ProjectView) => void;
  onLaunch: (p: ProjectView) => void;
  onHydrate: (p: ProjectView) => void;
  onUpdateDesktop: (p: ProjectView) => void;
  onOpenLive: (p: ProjectView) => void;
  onStopProcess: (p: ProjectView) => void;
  onCopyReleaseLink: (releaseId: string) => void;
  onOpenExplorer: (p: ProjectView) => void;
  onOpenVSCode: (p: ProjectView) => void;
}

export function ReleaseMatrixTable({
  projects,
  healthMap,
  runningMap,
  selectionMode = false,
  selectedIds,
  onToggleSelect,
  onToggleSelectAll,
  onRequestDelete,
  onSelect,
  onLaunch,
  onHydrate,
  onUpdateDesktop,
  onOpenLive,
  onStopProcess,
  onCopyReleaseLink,
  onOpenExplorer,
  onOpenVSCode,
}: ReleaseMatrixTableProps) {
  const allSelected =
    projects.length > 0 &&
    Boolean(selectedIds) &&
    projects.every((p) => selectedIds?.has(p.id));

  return (
    <div className="rounded-md border border-edge bg-surface overflow-x-auto">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-edge bg-canvas/70 text-[11px] font-semibold uppercase tracking-wider text-muted">
            {selectionMode && onToggleSelectAll && (
              <th className="py-2.5 pl-3.5 pr-1 w-9">
                <button
                  type="button"
                  onClick={onToggleSelectAll}
                  aria-label={allSelected ? "Bỏ chọn tất cả" : "Chọn tất cả"}
                  className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
                    allSelected
                      ? "bg-accent border-accent text-canvas"
                      : "bg-canvas border-edge text-transparent hover:border-muted"
                  }`}
                >
                  <Check className="w-3 h-3 stroke-[3]" />
                </button>
              </th>
            )}
            <th className="py-2.5 px-3.5">Project</th>
            <th className="py-2.5 px-3">Web Live</th>
            <th className="py-2.5 px-3">Desktop (R2 / Máy)</th>
            <th className="py-2.5 px-3">Local &amp; Git</th>
            <th className="py-2.5 px-3.5 text-right">Thao tác</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-edge text-[13px]">
          {projects.map((p) => {
            const webTarget = (p.targets || []).find((t) => t.kind === "web");
            const deskTarget = (p.targets || []).find(
              (t) => t.kind === "desktop"
            );
            const webRel = webTarget ? p.latest_releases?.[webTarget.id] : undefined;
            const deskRel = deskTarget
              ? p.latest_releases?.[deskTarget.id]
              : undefined;
            const webHealth = webTarget ? healthMap[webTarget.id] : undefined;
            const running = runningMap[p.id];
            const firstPort = running?.ports?.[0];
            const isSelected = Boolean(selectedIds?.has(p.id));

            return (
              <tr
                key={p.id}
                onClick={() =>
                  selectionMode && onToggleSelect
                    ? onToggleSelect(p)
                    : onSelect(p)
                }
                className={`cursor-pointer transition-colors ${
                  isSelected
                    ? "bg-elevated/80"
                    : "hover:bg-elevated/60"
                }`}
              >
                {selectionMode && onToggleSelect && (
                  <td
                    className="py-2.5 pl-3.5 pr-1 w-9"
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
                <td className="py-2.5 px-3.5 min-w-[230px]">
                  <div className="flex items-center gap-2.5">
                    {p.thumbnail_url ? (
                      <img
                        src={p.thumbnail_url}
                        alt={p.name}
                        className="w-12 h-7 rounded object-cover border border-edge shrink-0 bg-canvas"
                      />
                    ) : (
                      <div className="w-12 h-7 rounded border border-edge bg-canvas shrink-0 flex items-center justify-center text-[10px] font-mono text-muted">
                        {p.type.toUpperCase()}
                      </div>
                    )}
                    <div className="min-w-0">
                      <div className="font-medium text-primary truncate flex items-center gap-1.5">
                        <span className="truncate">{p.name}</span>
                        <span className="px-1.5 py-0.2 rounded bg-canvas border border-edge text-[10px] font-mono text-muted shrink-0">
                          {p.type === "hybrid"
                            ? "HYBRID"
                            : p.type === "web"
                            ? "WEB"
                            : "EXE"}
                        </span>
                      </div>
                      <div className="text-[11px] text-muted truncate">
                        {p.category || "Chưa phân nhóm"}
                      </div>
                    </div>
                  </div>
                </td>

                {/* Web Live */}
                <td className="py-2.5 px-3 min-w-[180px]">
                  {webTarget ? (
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-1.5">
                        {webHealth ? (
                          <span
                            className={`text-[11px] font-mono font-semibold ${
                              webHealth.online
                                ? webHealth.drift
                                  ? "text-status-warn"
                                  : "text-status-ready"
                                : "text-status-danger"
                            }`}
                          >
                            {webHealth.online
                              ? `ON ${webHealth.latency_ms}ms`
                              : "OFFLINE"}
                          </span>
                        ) : (
                          <span className="text-[11px] font-mono text-muted">
                            {webRel?.version || (p.live_url ? "URL ready" : "-")}
                          </span>
                        )}
                        {webHealth?.live_version && (
                          <span className="text-[11px] font-mono text-primary">
                            {webHealth.live_version}
                          </span>
                        )}
                      </div>
                      {p.live_url && (
                        <div className="text-[11px] font-mono text-muted truncate max-w-[200px]">
                          {p.live_url.replace(/^https?:\/\//, "")}
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className="text-xs text-muted/60">-</span>
                  )}
                </td>

                {/* Desktop Release */}
                <td className="py-2.5 px-3 min-w-[170px]">
                  {deskTarget || p.type !== "web" ? (
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-1.5 font-mono text-xs">
                        <span className="text-primary font-semibold">
                          {deskRel?.version || p.latest_version || "-"}
                        </span>
                        {p.installed_version && (
                          <span
                            className={
                              p.has_update
                                ? "text-status-warn"
                                : "text-muted"
                            }
                          >
                            / local {p.installed_version}
                          </span>
                        )}
                      </div>
                      {p.has_update && (
                        <span className="inline-block px-1.5 py-0.2 rounded bg-status-warn/15 border border-status-warn/50 text-[10px] font-mono text-status-warn">
                          Có bản mới
                        </span>
                      )}
                    </div>
                  ) : (
                    <span className="text-xs text-muted/60">-</span>
                  )}
                </td>

                {/* Local & Git */}
                <td className="py-2.5 px-3 min-w-[150px]">
                  <div className="space-y-0.5">
                    {running && running.pid > 0 ? (
                      <span className="inline-flex items-center gap-1 text-[11px] font-mono font-semibold text-status-ready">
                        <span className="w-1.5 h-1.5 rounded-full bg-status-ready animate-pulse" />
                        RUNNING {firstPort ? `:${firstPort}` : ""}
                      </span>
                    ) : (
                      <span className="text-xs font-mono text-muted">
                        {p.status}
                      </span>
                    )}
                    {p.git_branch && (
                      <div className="text-[11px] font-mono text-muted/80 truncate">
                        {p.git_branch}
                        {p.git_commit ? ` @ ${p.git_commit}` : ""}
                      </div>
                    )}
                  </div>
                </td>

                {/* Actions */}
                <td
                  className="py-2.5 px-3.5 text-right"
                  onClick={(e) => e.stopPropagation()}
                >
                  <div className="inline-flex items-center gap-1">
                    {p.status === "cloud_only" ? (
                      <button
                        type="button"
                        onClick={() => onHydrate(p)}
                        title="Install / Clone về máy"
                        className="h-7 px-2 rounded bg-accent text-canvas text-xs font-semibold inline-flex items-center gap-1"
                      >
                        <Download className="w-3 h-3" />
                        <span>Tải</span>
                      </button>
                    ) : (
                      <button
                        type="button"
                        onClick={() => onLaunch(p)}
                        title="Khởi chạy"
                        className="h-7 px-2 rounded bg-accent text-canvas text-xs font-semibold inline-flex items-center gap-1"
                      >
                        <Play className="w-3 h-3 fill-current" />
                        <span>Run</span>
                      </button>
                    )}

                    {running && running.pid > 0 && (
                      <button
                        type="button"
                        onClick={() => onStopProcess(p)}
                        title="Dừng tiến trình đang chạy"
                        className="h-7 w-7 rounded border border-status-danger/60 text-status-danger hover:bg-status-danger/10 inline-flex items-center justify-center"
                      >
                        <Square className="w-3 h-3 fill-current" />
                      </button>
                    )}

                    {p.has_update && (
                      <button
                        type="button"
                        onClick={() => onUpdateDesktop(p)}
                        title="Cập nhật Desktop App ngay"
                        className="h-7 w-7 rounded border border-status-warn/60 text-status-warn hover:bg-status-warn/10 inline-flex items-center justify-center"
                      >
                        <RefreshCw className="w-3.5 h-3.5" />
                      </button>
                    )}

                    {p.live_url && (
                      <button
                        type="button"
                        onClick={() => onOpenLive(p)}
                        title={`Mở ${p.live_url}`}
                        className="h-7 w-7 rounded border border-edge bg-canvas hover:bg-elevated text-status-ready inline-flex items-center justify-center"
                      >
                        <Globe className="w-3.5 h-3.5" />
                      </button>
                    )}

                    {deskRel && (
                      <button
                        type="button"
                        onClick={() => onCopyReleaseLink(deskRel.id)}
                        title="Copy Download Link"
                        className="h-7 w-7 rounded border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary inline-flex items-center justify-center"
                      >
                        <Link2 className="w-3.5 h-3.5" />
                      </button>
                    )}

                    {p.local_dir && p.status !== "cloud_only" && (
                      <>
                        <button
                          type="button"
                          onClick={() => onOpenExplorer(p)}
                          title="Explorer"
                          className="h-7 w-7 rounded border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary inline-flex items-center justify-center"
                        >
                          <FolderOpen className="w-3.5 h-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => onOpenVSCode(p)}
                          title="VS Code"
                          className="h-7 w-7 rounded border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary inline-flex items-center justify-center"
                        >
                          <Code2 className="w-3.5 h-3.5" />
                        </button>
                      </>
                    )}

                    <button
                      type="button"
                      onClick={() => onSelect(p)}
                      title="Chi tiết"
                      className="h-7 w-7 rounded border border-edge bg-canvas hover:bg-elevated text-muted hover:text-primary inline-flex items-center justify-center"
                    >
                      {p.live_url ? (
                        <ExternalLink className="w-3.5 h-3.5 hidden" />
                      ) : null}
                      <SlidersHorizontal className="w-3.5 h-3.5" />
                    </button>

                    {onRequestDelete && (
                      <button
                        type="button"
                        onClick={() => onRequestDelete(p)}
                        title="Xóa dự án"
                        className="h-7 w-7 rounded border border-edge bg-canvas hover:border-status-danger/60 hover:bg-status-danger/10 text-muted hover:text-status-danger inline-flex items-center justify-center"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

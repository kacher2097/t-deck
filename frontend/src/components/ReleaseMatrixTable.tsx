import { Check } from "lucide-react";
import type {
  ProjectView,
  RunningProcess,
  WebHealthResult,
} from "../types/project";
import { ReleaseMatrixRow } from "./project/ReleaseMatrixRow";

interface ReleaseMatrixTableProps {
  projects: ProjectView[];
  healthMap: Record<string, WebHealthResult>;
  runningMap: Record<string, RunningProcess>;
  busyAction?: string | null;
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
  onOpenLocalPort?: (port: number) => void;
  onStopProcess: (p: ProjectView) => void;
  onCopyReleaseLink: (releaseId: string) => void;
  onOpenExplorer: (p: ProjectView) => void;
  onOpenVSCode: (p: ProjectView) => void;
}

export function ReleaseMatrixTable({
  projects,
  healthMap,
  runningMap,
  busyAction = null,
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
  onOpenLocalPort,
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
    <div
      tabIndex={0}
      role="region"
      aria-label="Bảng danh sách dự án"
      className="rounded-md border border-edge bg-surface overflow-x-auto focus:outline-none focus:ring-1 focus:ring-accent/50"
    >
      <table className="w-full text-left border-separate border-spacing-0">
        <thead className="sticky top-0 z-20 bg-canvas">
          <tr className="text-[11px] font-semibold uppercase tracking-wider text-muted">
            {selectionMode && onToggleSelectAll && (
              <th className="py-2.5 pl-3.5 pr-1 w-9 border-b border-edge bg-canvas">
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
            <th className="py-2.5 px-3.5 border-b border-edge bg-canvas">
              Dự án
            </th>
            <th className="py-2.5 px-3 border-b border-edge bg-canvas">
              Trạng thái &amp; Git
            </th>
            <th className="py-2.5 px-3 border-b border-edge bg-canvas">
              Phát hành &amp; Live
            </th>
            <th className="sticky right-0 z-30 py-2.5 pl-3 pr-3.5 text-right border-b border-l border-edge bg-canvas">
              Thao tác
            </th>
          </tr>
        </thead>
        <tbody className="text-[13px]">
          {projects.map((p) => (
            <ReleaseMatrixRow
              key={p.id}
              project={p}
              healthMap={healthMap}
              running={runningMap[p.id]}
              isBusy={busyAction === p.id}
              selectionMode={selectionMode}
              isSelected={Boolean(selectedIds?.has(p.id))}
              onToggleSelect={onToggleSelect}
              onRequestDelete={onRequestDelete}
              onSelect={onSelect}
              onLaunch={onLaunch}
              onHydrate={onHydrate}
              onUpdateDesktop={onUpdateDesktop}
              onOpenLive={onOpenLive}
              onOpenLocalPort={onOpenLocalPort}
              onStopProcess={onStopProcess}
              onCopyReleaseLink={onCopyReleaseLink}
              onOpenExplorer={onOpenExplorer}
              onOpenVSCode={onOpenVSCode}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

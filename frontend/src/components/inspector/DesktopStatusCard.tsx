import { Link2, Loader2, RefreshCw, RotateCcw } from "lucide-react";
import type { ProjectView, Release } from "../../types/project";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_CARD_BOX,
  UI_SECTION_TITLE_CLS,
} from "../ui/uiTokens";

interface DesktopStatusCardProps {
  project: ProjectView;
  latestRelease?: Release;
  updatingDesktop: boolean;
  onUpdateDesktop: (releaseId?: string) => void;
  onRollbackDesktop: () => void;
  onCopyDownloadLink: (releaseId: string) => void;
}

export function DesktopStatusCard({
  project,
  latestRelease,
  updatingDesktop,
  onUpdateDesktop,
  onRollbackDesktop,
  onCopyDownloadLink,
}: DesktopStatusCardProps) {
  return (
    <div className={`${UI_CARD_BOX} space-y-3`}>
      <div className={UI_SECTION_TITLE_CLS}>Trạng thái Desktop Release</div>
      <div className="flex items-center justify-between text-[13px]">
        <span className="text-muted">Mới nhất (R2)</span>
        <span className="font-mono font-semibold text-primary">
          {latestRelease?.version || project.latest_version || "Chưa phát hành"}
        </span>
      </div>
      <div className="flex items-center justify-between text-[13px]">
        <span className="text-muted">Đang cài trên máy</span>
        <span
          className={`font-mono font-semibold ${
            project.has_update ? "text-status-warn" : "text-status-ready"
          }`}
        >
          {project.installed_version || "(Chưa ghi nhận)"}
        </span>
      </div>
      {project.r2_release_key && (
        <div className="text-[11px] font-mono text-muted break-all">
          R2: {project.r2_release_key}
        </div>
      )}

      <div className="pt-2 border-t border-edge flex flex-wrap gap-2">
        <button
          type="button"
          disabled={updatingDesktop || !latestRelease}
          onClick={() => onUpdateDesktop(latestRelease?.id || "")}
          className={UI_BTN_PRIMARY}
        >
          {updatingDesktop ? (
            <Loader2 className="w-3.5 h-3.5 animate-spin" />
          ) : (
            <RefreshCw className="w-3.5 h-3.5" />
          )}
          <span>
            {project.has_update ? "Cập nhật ngay" : "Cài / Sync bản mới"}
          </span>
        </button>
        <button
          type="button"
          disabled={updatingDesktop}
          onClick={onRollbackDesktop}
          className={UI_BTN_SECONDARY}
        >
          <RotateCcw className="w-3.5 h-3.5" />
          <span>Rollback</span>
        </button>
        {latestRelease && (
          <button
            type="button"
            onClick={() => onCopyDownloadLink(latestRelease.id)}
            className={UI_BTN_SECONDARY}
          >
            <Link2 className="w-3.5 h-3.5" />
            <span>Copy Link</span>
          </button>
        )}
      </div>
    </div>
  );
}

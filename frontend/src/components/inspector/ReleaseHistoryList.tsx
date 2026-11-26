import { Check, Download, Link2, Loader2, Trash2 } from "lucide-react";
import type { Release } from "../../types/project";
import { ConfirmPopover } from "../ui/ConfirmPopover";
import { UI_CARD_BOX, UI_SECTION_TITLE_CLS } from "../ui/uiTokens";

interface ReleaseHistoryListProps {
  releases: Release[];
  loading: boolean;
  isWeb: boolean;
  copiedKey: string;
  onInstallRelease: (releaseId: string) => void;
  onCopyDownloadLink: (releaseId: string) => void;
  onDeleteRelease: (release: Release) => void;
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatTime(unixSec: number): string {
  if (!unixSec) return "";
  return new Date(unixSec * 1000).toLocaleString();
}

export function ReleaseHistoryList({
  releases,
  loading,
  isWeb,
  copiedKey,
  onInstallRelease,
  onCopyDownloadLink,
  onDeleteRelease,
}: ReleaseHistoryListProps) {
  return (
    <div className={`${UI_CARD_BOX} space-y-2.5`}>
      <div className="flex items-center justify-between">
        <span className={UI_SECTION_TITLE_CLS}>
          Lịch sử phát hành ({releases.length})
        </span>
        {loading && <Loader2 className="w-3.5 h-3.5 animate-spin text-muted" />}
      </div>

      {releases.length === 0 ? (
        <p className="text-xs text-muted">Chưa có bản phát hành nào.</p>
      ) : (
        <div className="space-y-2 max-h-[240px] overflow-y-auto pr-1">
          {releases.map((rel) => (
            <div
              key={rel.id}
              className="p-2.5 rounded bg-surface border border-edge space-y-1.5"
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className="font-mono text-xs font-semibold text-primary">
                    {rel.version}
                  </span>
                  <span className="px-1.5 py-0.5 rounded bg-canvas border border-edge text-[10px] font-mono text-muted">
                    {rel.channel}
                  </span>
                  <span className="text-[11px] font-mono text-muted">
                    {formatBytes(rel.file_size)}
                  </span>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  {!isWeb && rel.r2_key && (
                    <button
                      type="button"
                      title="Cài đặt phiên bản này"
                      onClick={() => onInstallRelease(rel.id)}
                      className="p-1 text-muted hover:text-primary"
                    >
                      <Download className="w-3.5 h-3.5" />
                    </button>
                  )}
                  {(rel.r2_key || rel.deploy_url) && (
                    <button
                      type="button"
                      title="Copy Link tải"
                      onClick={() => onCopyDownloadLink(rel.id)}
                      className="p-1 text-muted hover:text-primary"
                    >
                      {copiedKey === `link-${rel.id}` ? (
                        <Check className="w-3.5 h-3.5 text-status-ready" />
                      ) : (
                        <Link2 className="w-3.5 h-3.5" />
                      )}
                    </button>
                  )}
                  <ConfirmPopover
                    title={`Xóa bản phát hành ${rel.version}?`}
                    description={
                      rel.r2_key
                        ? "Bản ghi và tệp đóng gói trên R2 sẽ bị xóa."
                        : "Bản ghi phát hành này sẽ bị xóa khỏi lịch sử."
                    }
                    confirmLabel="Xóa release"
                    onConfirm={() => onDeleteRelease(rel)}
                  >
                    {({ open, toggle }) => (
                      <button
                        type="button"
                        onClick={toggle}
                        title="Xóa release"
                        aria-label={`Xóa release ${rel.version}`}
                        aria-expanded={open}
                        className={`p-1 rounded transition-colors ${
                          open
                            ? "text-status-danger bg-status-danger/10"
                            : "text-muted hover:text-status-danger"
                        }`}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </ConfirmPopover>
                </div>
              </div>
              {rel.sha256 && (
                <div className="text-[10px] font-mono text-muted truncate">
                  SHA256: {rel.sha256.slice(0, 16)}…
                </div>
              )}
              {rel.changelog && (
                <p className="text-xs text-muted line-clamp-2">
                  {rel.changelog}
                </p>
              )}
              <div className="text-[10px] text-muted/70">
                {formatTime(rel.published_at)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

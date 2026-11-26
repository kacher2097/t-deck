import { AlertTriangle, FolderX, HardDrive, Loader2, ShieldCheck, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import type { ProjectView } from "../types/project";
import { ModalDialog } from "./ui/ModalDialog";
import { UI_BTN_SECONDARY } from "./ui/uiTokens";

interface ConfirmDeleteModalProps {
  projects: ProjectView[];
  busy: boolean;
  onCancel: () => void;
  onConfirm: (deleteLocalFiles: boolean) => void;
}

export function ConfirmDeleteModal({
  projects,
  busy,
  onCancel,
  onConfirm,
}: ConfirmDeleteModalProps) {
  const [deleteLocalFiles, setDeleteLocalFiles] = useState(false);

  useEffect(() => {
    setDeleteLocalFiles(false);
  }, [projects]);

  const open = projects.length > 0;
  if (!open) return null;

  const isSingle = projects.length === 1;
  const hasAnyLocalDir = projects.some((p) => Boolean(p.local_dir));

  const confirmLabel = busy
    ? "Đang xóa..."
    : deleteLocalFiles
    ? isSingle
      ? "Xóa vĩnh viễn & xóa file"
      : `Xóa vĩnh viễn ${projects.length} dự án`
    : isSingle
    ? "Gỡ khỏi T-Deck"
    : `Gỡ ${projects.length} dự án khỏi T-Deck`;

  return (
    <ModalDialog
      open={open}
      onClose={busy ? () => {} : onCancel}
      size="md"
      title={
        <div className="flex items-center gap-2 text-status-danger">
          <Trash2 className="w-4 h-4 shrink-0" />
          <h1 className="font-heading text-base font-semibold">
            {isSingle
              ? `Xóa "${projects[0].name}"?`
              : `Xóa ${projects.length} dự án đã chọn?`}
          </h1>
        </div>
      }
      subtitle="Dự án đã xóa sẽ được đánh dấu bỏ qua ở lần Scan Local tiếp theo."
    >
      <div className="p-5 space-y-4">
        {isSingle ? (
          <div className="p-3 rounded-md bg-canvas border border-edge flex items-center gap-3 min-w-0">
            {projects[0].thumbnail_url ? (
              <img
                src={projects[0].thumbnail_url}
                alt={projects[0].name}
                className="w-16 h-9 rounded object-cover border border-edge shrink-0 bg-surface"
              />
            ) : (
              <div className="w-16 h-9 rounded border border-edge bg-surface shrink-0 flex items-center justify-center text-[10px] font-mono font-semibold text-muted">
                {projects[0].type === "hybrid"
                  ? "HYBRID"
                  : projects[0].type === "web"
                  ? "WEB"
                  : "EXE"}
              </div>
            )}
            <div className="min-w-0 flex-1 space-y-0.5">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-[13px] font-semibold text-primary truncate">
                  {projects[0].name}
                </span>
                <span className="px-1.5 py-0.2 rounded bg-surface border border-edge text-[10px] font-mono text-muted shrink-0">
                  {projects[0].type === "hybrid"
                    ? "HYBRID"
                    : projects[0].type === "web"
                    ? "WEB"
                    : "EXE"}
                </span>
                {projects[0].category && (
                  <span className="px-1.5 py-0.2 rounded bg-elevated border border-edge text-[10px] text-muted truncate max-w-[140px]">
                    {projects[0].category}
                  </span>
                )}
              </div>
              {projects[0].local_dir && (
                <div className="text-[11px] font-mono text-muted truncate">
                  {projects[0].local_dir}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs text-muted">
              <span>Danh sách dự án sẽ xử lý</span>
              <span className="px-2 py-0.5 rounded bg-canvas border border-edge font-mono text-[11px] text-primary">
                {projects.length} mục
              </span>
            </div>
            <div className="max-h-44 overflow-y-auto rounded-md bg-canvas border border-edge divide-y divide-edge">
              {projects.map((p) => (
                <div
                  key={p.id}
                  className="px-3 py-2 flex items-center justify-between gap-3 text-xs"
                >
                  <span className="font-medium text-primary truncate">
                    {p.name}
                  </span>
                  <span className="font-mono text-[11px] text-muted truncate max-w-[55%]">
                    {p.local_dir || p.category || p.type}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {hasAnyLocalDir && (
          <div
            role="radiogroup"
            aria-label="Phạm vi xóa dự án"
            className="space-y-2"
          >
            <button
              type="button"
              role="radio"
              aria-checked={!deleteLocalFiles}
              disabled={busy}
              onClick={() => setDeleteLocalFiles(false)}
              className={`w-full text-left p-3 rounded-md border transition-colors flex items-start gap-3 ${
                !deleteLocalFiles
                  ? "border-accent/60 bg-elevated/80"
                  : "border-edge bg-canvas hover:bg-elevated/40"
              }`}
            >
              <span
                className={`mt-0.5 w-4 h-4 rounded-full border flex items-center justify-center shrink-0 ${
                  !deleteLocalFiles ? "border-accent" : "border-muted/60"
                }`}
              >
                {!deleteLocalFiles && (
                  <span className="w-2 h-2 rounded-full bg-accent" />
                )}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5 text-[13px] font-medium text-primary">
                  <ShieldCheck className="w-3.5 h-3.5 text-status-ready shrink-0" />
                  <span>Chỉ gỡ khỏi danh sách T-Deck</span>
                </div>
                <p className="text-[11.5px] text-muted mt-0.5 leading-relaxed">
                  Giữ nguyên toàn bộ mã nguồn và file trên ổ cứng. Dự án sẽ
                  không tự hiện lại ở lần Scan Local tiếp theo.
                </p>
              </div>
            </button>

            <button
              type="button"
              role="radio"
              aria-checked={deleteLocalFiles}
              disabled={busy}
              onClick={() => setDeleteLocalFiles(true)}
              className={`w-full text-left p-3 rounded-md border transition-colors flex items-start gap-3 ${
                deleteLocalFiles
                  ? "border-status-danger/60 bg-status-danger/10"
                  : "border-edge bg-canvas hover:bg-elevated/40"
              }`}
            >
              <span
                className={`mt-0.5 w-4 h-4 rounded-full border flex items-center justify-center shrink-0 ${
                  deleteLocalFiles ? "border-status-danger" : "border-muted/60"
                }`}
              >
                {deleteLocalFiles && (
                  <span className="w-2 h-2 rounded-full bg-status-danger" />
                )}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5 text-[13px] font-medium text-status-danger">
                  <HardDrive className="w-3.5 h-3.5 shrink-0" />
                  <span>Xóa vĩnh viễn cả thư mục trên ổ cứng</span>
                </div>
                <p className="text-[11.5px] text-muted mt-0.5 leading-relaxed">
                  Xóa toàn bộ thư mục cục bộ khỏi máy tính. Hành động này không
                  thể hoàn tác.
                </p>
              </div>
            </button>

            {deleteLocalFiles && (
              <div className="p-2.5 rounded-md border border-status-danger/50 bg-status-danger/10 text-xs text-status-danger flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>
                  Cảnh báo: Toàn bộ thư mục cục bộ của{" "}
                  {isSingle ? "dự án này" : `${projects.length} dự án`} sẽ bị
                  xóa vĩnh viễn khỏi ổ cứng và không thể khôi phục.
                </span>
              </div>
            )}
          </div>
        )}

        <div className="pt-2 border-t border-edge flex items-center justify-end gap-2.5">
          <button
            type="button"
            autoFocus
            disabled={busy}
            onClick={onCancel}
            className={UI_BTN_SECONDARY}
          >
            Hủy
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => onConfirm(deleteLocalFiles)}
            className="h-9 px-4 rounded-md bg-status-danger hover:bg-status-danger/90 text-white text-[13px] font-semibold inline-flex items-center gap-2 disabled:opacity-50 transition-colors"
          >
            {busy ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : deleteLocalFiles ? (
              <FolderX className="w-4 h-4" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
            <span>{confirmLabel}</span>
          </button>
        </div>
      </div>
    </ModalDialog>
  );
}

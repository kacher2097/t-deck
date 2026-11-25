import { AlertTriangle, Loader2, Trash2 } from "lucide-react";
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

  return (
    <ModalDialog
      open={open}
      onClose={busy ? () => {} : onCancel}
      size="md"
      title={
        <div className="flex items-center gap-2 text-status-danger">
          <Trash2 className="w-4 h-4 shrink-0" />
          <h2 className="font-heading text-base font-semibold">
            {isSingle
              ? `Xóa "${projects[0].name}"?`
              : `Xóa ${projects.length} dự án đã chọn?`}
          </h2>
        </div>
      }
      subtitle="Dự án bị xóa sẽ không tự hiện lại ở lần Scan Local tiếp theo."
    >
      <div className="p-5 space-y-4">
        {isSingle ? (
          <div className="p-3 rounded-md bg-canvas border border-edge space-y-1">
            <div className="text-[13px] font-medium text-primary">
              {projects[0].name}
            </div>
            {projects[0].local_dir && (
              <div className="text-[11px] font-mono text-muted break-all">
                {projects[0].local_dir}
              </div>
            )}
          </div>
        ) : (
          <div className="max-h-48 overflow-y-auto rounded-md bg-canvas border border-edge divide-y divide-edge">
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
        )}

        {hasAnyLocalDir && (
          <div className="space-y-2">
            <label className="flex items-start gap-2.5 text-[13px] text-primary cursor-pointer select-none">
              <input
                type="checkbox"
                checked={deleteLocalFiles}
                disabled={busy}
                onChange={(e) => setDeleteLocalFiles(e.target.checked)}
                className="mt-0.5 rounded border-edge bg-canvas"
              />
              <span>
                Xóa luôn thư mục mã nguồn / app trên ổ cứng
                <span className="block text-[11px] text-muted mt-0.5">
                  Mặc định chỉ gỡ khỏi danh sách T-Deck và giữ nguyên file trên
                  máy.
                </span>
              </span>
            </label>

            {deleteLocalFiles && (
              <div className="p-2.5 rounded-md border border-status-danger/50 bg-status-danger/10 text-xs text-status-danger flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>
                  Hành động này sẽ xóa vĩnh viễn toàn bộ thư mục cục bộ của{" "}
                  {isSingle ? "dự án này" : `${projects.length} dự án`} khỏi ổ
                  cứng và không thể hoàn tác.
                </span>
              </div>
            )}
          </div>
        )}

        <div className="pt-2 border-t border-edge flex items-center justify-end gap-2.5">
          <button
            type="button"
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
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
            <span>
              {busy
                ? "Đang xóa..."
                : isSingle
                ? "Xác nhận xóa"
                : `Xóa ${projects.length} dự án`}
            </span>
          </button>
        </div>
      </div>
    </ModalDialog>
  );
}

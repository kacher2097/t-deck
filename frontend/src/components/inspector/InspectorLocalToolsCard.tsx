import {
  Check,
  Code2,
  Copy,
  FileCode2,
  FolderOpen,
  ShieldCheck,
  Terminal,
  Trash2,
} from "lucide-react";
import { useEffect, useState } from "react";
import { api, formatError } from "../../lib/api";
import type { BatTemplate, ProjectType, ProjectView } from "../../types/project";
import {
  UI_BTN_SECONDARY,
  UI_SECTION_TITLE_CLS,
  UI_SELECT_CLS,
} from "../ui/uiTokens";

interface InspectorLocalToolsCardProps {
  project: ProjectView;
  type: ProjectType;
  batFile: string;
  templates: BatTemplate[];
  selectedTpl: string;
  generatingBat: boolean;
  syncingEnv: boolean;
  onSelectTpl: (id: string) => void;
  onGenerateBat: () => void;
  onSyncEnv: () => void;
  onDeleted: (projectId: string) => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function InspectorLocalToolsCard({
  project,
  type,
  batFile,
  templates,
  selectedTpl,
  generatingBat,
  syncingEnv,
  onSelectTpl,
  onGenerateBat,
  onSyncEnv,
  onDeleted,
  onNotify,
}: InspectorLocalToolsCardProps) {
  const [deleteLocalFiles, setDeleteLocalFiles] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [copiedId, setCopiedId] = useState(false);

  useEffect(() => {
    setDeleteLocalFiles(false);
    setConfirmDelete(false);
    setCopiedId(false);
  }, [project.id]);

  const handleCopyUUID = async () => {
    try {
      await navigator.clipboard.writeText(project.id);
      setCopiedId(true);
      setTimeout(() => setCopiedId(false), 1500);
    } catch {
      // ignore clipboard errors
    }
  };

  const handleDelete = async () => {
    try {
      await api.deleteProject(project.id, deleteLocalFiles);
      onDeleted(project.id);
      onNotify("success", `Đã xóa "${project.name}".`);
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  const hasLocalDir = Boolean(project.local_dir && project.status !== "cloud_only");

  return (
    <div className="p-5 rounded-md bg-canvas border border-edge space-y-4 min-w-0">
      {hasLocalDir && (
        <>
          <div className="space-y-2.5">
            <div className={UI_SECTION_TITLE_CLS}>Tiện ích tại máy</div>

            {/* Mở nhanh */}
            <div className="grid grid-cols-3 gap-2.5">
              <button
                type="button"
                onClick={() =>
                  api
                    .openInExplorer(project.id)
                    .catch((e) => onNotify("error", formatError(e)))
                }
                className="box-border h-9 px-2.5 w-full min-w-0 rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center justify-center gap-1.5 transition-colors"
              >
                <FolderOpen className="w-3.5 h-3.5 text-muted shrink-0" />
                <span className="truncate">Explorer</span>
              </button>
              <button
                type="button"
                onClick={() =>
                  api
                    .openInEditor(project.id, "code")
                    .catch((e) => onNotify("error", formatError(e)))
                }
                className="box-border h-9 px-2.5 w-full min-w-0 rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center justify-center gap-1.5 transition-colors"
              >
                <Code2 className="w-3.5 h-3.5 text-muted shrink-0" />
                <span className="truncate">VS Code</span>
              </button>
              <button
                type="button"
                onClick={() =>
                  api
                    .openInEditor(project.id, "terminal")
                    .catch((e) => onNotify("error", formatError(e)))
                }
                className="box-border h-9 px-2.5 w-full min-w-0 rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center justify-center gap-1.5 transition-colors"
              >
                <Terminal className="w-3.5 h-3.5 text-muted shrink-0" />
                <span className="truncate">Terminal</span>
              </button>
            </div>

            {/* Compact file status pills */}
            <div className="grid grid-cols-3 gap-2.5">
              {[
                { name: ".env", ok: project.has_env },
                { name: ".env.example", ok: project.has_env_example },
                { name: "start.bat", ok: project.has_bat },
              ].map((item) => (
                <div
                  key={item.name}
                  className="px-2.5 py-2 rounded-md bg-surface border border-edge flex items-center justify-between gap-1.5 min-w-0"
                >
                  <span className="text-[11px] font-mono text-muted truncate">
                    {item.name}
                  </span>
                  <span
                    className={`w-2 h-2 rounded-full shrink-0 ${
                      item.ok ? "bg-status-ready" : "bg-edge"
                    }`}
                    title={item.ok ? "Có sẵn" : "Chưa có"}
                  />
                </div>
              ))}
            </div>
          </div>

          {/* Tạo start.bat mẫu */}
          {type !== "desktop" && (
            <div className="pt-3.5 border-t border-edge space-y-2 min-w-0">
              <div className="text-xs font-medium text-primary flex items-center gap-2 min-w-0">
                <FileCode2 className="w-3.5 h-3.5 text-muted shrink-0" />
                <span className="truncate">Tạo {batFile || "start.bat"} mẫu</span>
              </div>
              <div className="flex items-center gap-2 min-w-0">
                <select
                  value={selectedTpl}
                  onChange={(e) => onSelectTpl(e.target.value)}
                  className={`${UI_SELECT_CLS} flex-1 min-w-0 bg-surface`}
                >
                  {templates.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.label}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  disabled={generatingBat}
                  onClick={onGenerateBat}
                  className={UI_BTN_SECONDARY}
                >
                  {generatingBat ? "Đang tạo..." : "Tạo file"}
                </button>
              </div>
            </div>
          )}

          {/* Sync .env.example */}
          <div className="pt-3.5 border-t border-edge flex items-center justify-between gap-3 min-w-0">
            <div className="min-w-0">
              <div className="text-xs font-medium text-primary flex items-center gap-2 min-w-0">
                <ShieldCheck className="w-3.5 h-3.5 text-status-ready shrink-0" />
                <span className="truncate">Sync .env.example</span>
              </div>
              <p className="text-[11px] text-muted mt-0.5 truncate">
                Xóa sạch giá trị bí mật sau dấu =
              </p>
            </div>
            <button
              type="button"
              disabled={syncingEnv || !project.has_env}
              onClick={onSyncEnv}
              className={UI_BTN_SECONDARY}
            >
              {syncingEnv ? "Đang sync..." : "Sync .env"}
            </button>
          </div>
        </>
      )}

      {/* Định danh & Hệ thống — spacious inline bar */}
      <div
        className={`${
          hasLocalDir ? "pt-3.5 border-t border-edge" : ""
        } space-y-2.5 min-w-0`}
      >
        <div className={UI_SECTION_TITLE_CLS}>Định danh & Hệ thống</div>

        {!confirmDelete ? (
          <div className="flex items-center gap-2 min-w-0">
            <div className="h-9 px-3 rounded-md bg-surface border border-edge flex items-center justify-between gap-2 flex-1 min-w-0">
              <code
                title={project.id}
                className="font-mono text-xs text-primary truncate min-w-0"
              >
                {project.id}
              </code>
              <button
                type="button"
                onClick={handleCopyUUID}
                title="Sao chép Project UUID"
                className="text-xs text-muted hover:text-primary inline-flex items-center gap-1 shrink-0 transition-colors"
              >
                {copiedId ? (
                  <Check className="w-3.5 h-3.5 text-status-ready" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
                <span>{copiedId ? "Copied" : "Copy"}</span>
              </button>
            </div>

            <button
              type="button"
              onClick={() => setConfirmDelete(true)}
              className="box-border h-9 px-3 shrink-0 whitespace-nowrap rounded-md border border-status-danger/60 text-status-danger hover:bg-status-danger/15 text-xs font-medium inline-flex items-center gap-1.5 transition-colors"
            >
              <Trash2 className="w-3.5 h-3.5" />
              <span>Xóa dự án</span>
            </button>
          </div>
        ) : (
          <div className="p-3.5 rounded-md border border-status-danger/40 bg-status-danger/5 space-y-2.5">
            <div className="text-xs font-semibold text-status-danger">
              Xác nhận xóa "{project.name}" khỏi T-Deck?
            </div>
            {project.local_dir && (
              <label className="flex items-center gap-2 text-xs text-muted cursor-pointer">
                <input
                  type="checkbox"
                  checked={deleteLocalFiles}
                  onChange={(e) => setDeleteLocalFiles(e.target.checked)}
                  className="rounded border-edge bg-canvas"
                />
                <span>Xóa luôn thư mục mã nguồn/app trên ổ cứng</span>
              </label>
            )}
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleDelete}
                className="box-border h-8 px-3.5 rounded-md bg-status-danger hover:bg-status-danger/90 text-white text-xs font-semibold transition-colors"
              >
                Xác nhận xóa
              </button>
              <button
                type="button"
                onClick={() => setConfirmDelete(false)}
                className="box-border h-8 px-3 rounded-md border border-edge bg-surface hover:bg-elevated text-xs text-primary transition-colors"
              >
                Hủy
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

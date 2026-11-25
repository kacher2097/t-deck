import { FolderPlus, Trash2 } from "lucide-react";
import { useState } from "react";
import { api, formatError } from "../../lib/api";
import type { Settings } from "../../types/project";
import { FormField } from "../ui/FormField";
import {
  UI_BTN_SECONDARY,
  UI_INPUT_CLS,
  UI_MONO_INPUT_CLS,
} from "../ui/uiTokens";

interface WorkspaceSettingsSectionProps {
  form: Settings;
  onChange: (updater: (prev: Settings) => Settings) => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function WorkspaceSettingsSection({
  form,
  onChange,
  onNotify,
}: WorkspaceSettingsSectionProps) {
  const [newWorkspaceDir, setNewWorkspaceDir] = useState("");

  const addWorkspaceDir = (dir: string) => {
    const clean = dir.trim();
    if (!clean) return;
    if (
      form.workspace_dirs.some((d) => d.toLowerCase() === clean.toLowerCase())
    ) {
      return;
    }
    onChange((prev) => ({
      ...prev,
      workspace_dirs: [...prev.workspace_dirs, clean],
    }));
    setNewWorkspaceDir("");
  };

  const handleBrowseWorkspace = async () => {
    try {
      const picked = await api.selectDirectory("Chọn thư mục chứa mã nguồn");
      if (picked) addWorkspaceDir(picked);
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  const handleBrowseAppsDir = async () => {
    try {
      const picked = await api.selectDirectory(
        "Chọn thư mục cài đặt Desktop Apps"
      );
      if (picked) onChange((prev) => ({ ...prev, apps_dir: picked }));
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  return (
    <div className="space-y-4">
      <FormField label="Thư mục gốc chứa Code (WORKSPACE_DIRS)">
        <div className="space-y-2 mb-2.5">
          {form.workspace_dirs.length === 0 ? (
            <p className="text-[13px] text-muted italic">
              Chưa khai báo thư mục workspace nào.
            </p>
          ) : (
            form.workspace_dirs.map((dir) => (
              <div
                key={dir}
                className="h-9 px-3 rounded-md bg-canvas border border-edge flex items-center justify-between gap-2"
              >
                <span className="text-[13px] font-mono text-primary truncate">
                  {dir}
                </span>
                <button
                  type="button"
                  onClick={() =>
                    onChange((prev) => ({
                      ...prev,
                      workspace_dirs: prev.workspace_dirs.filter(
                        (d) => d !== dir
                      ),
                    }))
                  }
                  className="text-muted hover:text-status-danger p-1"
                  aria-label={`Xóa ${dir}`}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))
          )}
        </div>

        <div className="flex gap-2">
          <input
            type="text"
            value={newWorkspaceDir}
            onChange={(e) => setNewWorkspaceDir(e.target.value)}
            placeholder="VD: D:\Projects"
            className={UI_MONO_INPUT_CLS}
          />
          <button
            type="button"
            onClick={() => addWorkspaceDir(newWorkspaceDir)}
            className={UI_BTN_SECONDARY}
          >
            Thêm
          </button>
          <button
            type="button"
            onClick={handleBrowseWorkspace}
            className={UI_BTN_SECONDARY}
          >
            <FolderPlus className="w-4 h-4 text-muted" />
            <span>Browse</span>
          </button>
        </div>
      </FormField>

      <FormField label="Thư mục cài đặt Desktop Apps (APPS_DIR)">
        <div className="flex gap-2">
          <input
            type="text"
            value={form.apps_dir}
            onChange={(e) =>
              onChange((prev) => ({ ...prev, apps_dir: e.target.value }))
            }
            placeholder="VD: C:\TDeckApps"
            className={UI_MONO_INPUT_CLS}
          />
          <button
            type="button"
            onClick={handleBrowseAppsDir}
            className={UI_BTN_SECONDARY}
          >
            Browse
          </button>
        </div>
      </FormField>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <FormField label="Lọc dự án cập nhật (ngày, 0 = tất cả)">
          <input
            type="number"
            min={0}
            value={form.scan_recent_days}
            onChange={(e) =>
              onChange((prev) => ({
                ...prev,
                scan_recent_days: Math.max(0, Number(e.target.value) || 0),
              }))
            }
            className={UI_INPUT_CLS}
          />
        </FormField>
        <FormField label="Thư mục bỏ qua (phân cách dấu phẩy)">
          <input
            type="text"
            value={(form.scan_exclude_dirs || []).join(", ")}
            onChange={(e) =>
              onChange((prev) => ({
                ...prev,
                scan_exclude_dirs: e.target.value
                  .split(",")
                  .map((s) => s.trim())
                  .filter(Boolean),
              }))
            }
            placeholder="t-deck, agent-skills-master"
            className={UI_MONO_INPUT_CLS}
          />
        </FormField>
      </div>
    </div>
  );
}

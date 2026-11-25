import { Loader2, Save } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { api, formatError } from "../../lib/api";
import type { Settings } from "../../types/project";
import { ModalDialog } from "../ui/ModalDialog";
import { UI_BTN_PRIMARY, UI_BTN_SECONDARY } from "../ui/uiTokens";
import { R2SettingsSection } from "./R2SettingsSection";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";

interface SettingsModalProps {
  open: boolean;
  onClose: () => void;
  onSaved: (settings: Settings) => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

const EMPTY_SETTINGS: Settings = {
  workspace_dirs: [],
  apps_dir: "",
  scan_recent_days: 120,
  scan_exclude_dirs: [],
  r2: {
    enabled: false,
    account_id: "",
    access_key_id: "",
    secret_access_key: "",
    bucket_name: "my-storage",
    public_domain: "",
  },
};

export function SettingsModal({
  open,
  onClose,
  onSaved,
  onNotify,
}: SettingsModalProps) {
  const [form, setForm] = useState<Settings>(EMPTY_SETTINGS);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    api
      .getSettings()
      .then((s) => setForm(s))
      .catch((err) => onNotify("error", formatError(err)));
  }, [open, onNotify]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const saved = await api.saveSettings(form);
      onSaved(saved);
      onNotify("success", "Đã lưu cấu hình T-Deck.");
      onClose();
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ModalDialog
      open={open}
      onClose={onClose}
      size="lg"
      title={
        <h2 className="font-heading text-base font-semibold text-primary">
          Cấu hình Workspace & Cloudflare R2
        </h2>
      }
      subtitle="Quản lý thư mục quét, bộ lọc thời gian và thông số đồng bộ R2 (.env)"
    >
      <form onSubmit={handleSave} className="p-6 space-y-6">
        <WorkspaceSettingsSection
          form={form}
          onChange={setForm}
          onNotify={onNotify}
        />

        <R2SettingsSection
          r2={form.r2}
          onChange={(updater) =>
            setForm((prev) => ({ ...prev, r2: updater(prev.r2) }))
          }
        />

        <div className="pt-4 border-t border-edge flex justify-end gap-2.5">
          <button type="button" onClick={onClose} className={UI_BTN_SECONDARY}>
            Hủy
          </button>
          <button type="submit" disabled={saving} className={UI_BTN_PRIMARY}>
            {saving ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Save className="w-4 h-4" />
            )}
            <span>Lưu cài đặt</span>
          </button>
        </div>
      </form>
    </ModalDialog>
  );
}

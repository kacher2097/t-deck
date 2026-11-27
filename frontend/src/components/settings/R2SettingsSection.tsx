import { CheckCircle2, Eye, EyeOff, Loader2, PlugZap } from "lucide-react";
import { useState } from "react";
import { api, formatError } from "../../lib/api";
import type { R2Config } from "../../types/project";
import { FormField } from "../ui/FormField";
import { UI_BTN_SECONDARY, UI_MONO_INPUT_CLS } from "../ui/uiTokens";

interface R2SettingsSectionProps {
  r2: R2Config;
  onChange: (updater: (prev: R2Config) => R2Config) => void;
}

export function R2SettingsSection({ r2, onChange }: R2SettingsSectionProps) {
  const [showSecret, setShowSecret] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    msg: string;
  } | null>(null);

  const handleTestR2 = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      await api.testR2Connection(r2);
      setTestResult({ ok: true, msg: "Kết nối Cloudflare R2 thành công." });
    } catch (err) {
      setTestResult({ ok: false, msg: formatError(err) });
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="pt-4 border-t border-edge space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h3 className="font-heading text-sm font-semibold text-primary">
            Cloudflare R2 Storage
          </h3>
          <p className="text-xs text-muted mt-0.5">
            Đồng bộ catalog.json, thumbnails & releases (API key/secret lưu tại .env)
          </p>
        </div>
        <label
          htmlFor="r2-sync-enabled"
          className="flex items-center gap-2 text-[13px] font-medium text-primary cursor-pointer"
        >
          <input
            id="r2-sync-enabled"
            name="r2Enabled"
            type="checkbox"
            checked={r2.enabled}
            onChange={(e) =>
              onChange((prev) => ({ ...prev, enabled: e.target.checked }))
            }
          />
          <span>Kích hoạt</span>
        </label>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <FormField label="Account ID">
          <input
            type="text"
            value={r2.account_id}
            onChange={(e) =>
              onChange((prev) => ({ ...prev, account_id: e.target.value }))
            }
            className={UI_MONO_INPUT_CLS}
          />
        </FormField>
        <FormField label="Bucket Name">
          <input
            type="text"
            value={r2.bucket_name}
            onChange={(e) =>
              onChange((prev) => ({ ...prev, bucket_name: e.target.value }))
            }
            className={UI_MONO_INPUT_CLS}
          />
        </FormField>
      </div>

      <FormField label="Access Key ID">
        <input
          type="text"
          value={r2.access_key_id}
          onChange={(e) =>
            onChange((prev) => ({ ...prev, access_key_id: e.target.value }))
          }
          className={UI_MONO_INPUT_CLS}
        />
      </FormField>

      <FormField label="Public Domain (CDN / Custom Domain)" hint="Dùng cho link tải public & auto-update latest.json">
        <input
          type="text"
          value={r2.public_domain || ""}
          onChange={(e) =>
            onChange((prev) => ({ ...prev, public_domain: e.target.value }))
          }
          placeholder="https://cdn.example.com"
          className={UI_MONO_INPUT_CLS}
        />
      </FormField>

      <FormField label="Secret Access Key">
        <div className="relative">
          <input
            type={showSecret ? "text" : "password"}
            value={r2.secret_access_key}
            onChange={(e) =>
              onChange((prev) => ({
                ...prev,
                secret_access_key: e.target.value,
              }))
            }
            className={`${UI_MONO_INPUT_CLS} pr-10`}
          />
          <button
            type="button"
            onClick={() => setShowSecret((s) => !s)}
            aria-label="Hiện/ẩn Secret Key"
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-primary"
          >
            {showSecret ? (
              <EyeOff className="w-4 h-4" />
            ) : (
              <Eye className="w-4 h-4" />
            )}
          </button>
        </div>
      </FormField>

      <div className="flex items-center justify-between gap-3 pt-1">
        <button
          type="button"
          disabled={testing}
          onClick={handleTestR2}
          className={UI_BTN_SECONDARY}
        >
          {testing ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <PlugZap className="w-4 h-4 text-muted" />
          )}
          <span>Test Connection</span>
        </button>

        {testResult && (
          <span
            className={`text-[13px] flex items-center gap-1.5 truncate ${
              testResult.ok ? "text-status-ready" : "text-status-danger"
            }`}
          >
            {testResult.ok && (
              <CheckCircle2 className="w-4 h-4 shrink-0" />
            )}
            <span className="truncate">{testResult.msg}</span>
          </span>
        )}
      </div>
    </div>
  );
}

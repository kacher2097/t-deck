import { Activity, Check, Copy, ExternalLink, Loader2 } from "lucide-react";
import { api, formatError } from "../../lib/api";
import type { ProjectTarget, WebHealthResult } from "../../types/project";
import {
  UI_BTN_SECONDARY,
  UI_CARD_BOX,
  UI_SECTION_TITLE_CLS,
} from "../ui/uiTokens";

interface WebLiveStatusCardProps {
  activeTarget?: ProjectTarget;
  health: WebHealthResult | null;
  checkingHealth: boolean;
  copiedKey: string;
  onCopyText: (key: string, text: string, label: string) => void;
  onCheckHealth: () => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function WebLiveStatusCard({
  activeTarget,
  health,
  checkingHealth,
  copiedKey,
  onCopyText,
  onCheckHealth,
  onNotify,
}: WebLiveStatusCardProps) {
  return (
    <div className={`${UI_CARD_BOX} space-y-3`}>
      <div className="flex items-center justify-between gap-2">
        <span className={UI_SECTION_TITLE_CLS}>Web Live Environment</span>
        {activeTarget?.hosting_provider && (
          <span className="px-2 py-0.5 rounded bg-surface border border-edge text-[11px] font-mono text-muted">
            {activeTarget.hosting_provider}
          </span>
        )}
      </div>

      {activeTarget?.prod_url ? (
        <div className="space-y-2">
          <div className="p-2.5 rounded bg-surface border border-edge flex items-center justify-between gap-2">
            <span className="text-xs font-mono text-primary truncate">
              {activeTarget.prod_url}
            </span>
            <div className="flex items-center gap-1 shrink-0">
              <button
                type="button"
                onClick={() =>
                  onCopyText("prod", activeTarget.prod_url || "", "URL")
                }
                className="p-1.5 rounded hover:bg-elevated text-muted hover:text-primary"
                title="Copy URL"
              >
                {copiedKey === "prod" ? (
                  <Check className="w-3.5 h-3.5 text-status-ready" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
              <button
                type="button"
                onClick={() =>
                  api
                    .openTargetURL(activeTarget.id, "prod")
                    .catch((e) => onNotify("error", formatError(e)))
                }
                className="p-1.5 rounded hover:bg-elevated text-muted hover:text-primary"
                title="Mở trên trình duyệt"
              >
                <ExternalLink className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {activeTarget.staging_url && (
            <div className="p-2 rounded bg-surface border border-edge flex items-center justify-between gap-2">
              <span className="text-[11px] font-mono text-muted truncate">
                Staging: {activeTarget.staging_url}
              </span>
              <button
                type="button"
                onClick={() =>
                  api
                    .openTargetURL(activeTarget.id, "staging")
                    .catch((e) => onNotify("error", formatError(e)))
                }
                className="p-1 rounded hover:bg-elevated text-muted hover:text-primary"
              >
                <ExternalLink className="w-3 h-3" />
              </button>
            </div>
          )}

          <div className="flex items-center justify-between gap-2 pt-1">
            <button
              type="button"
              disabled={checkingHealth}
              onClick={onCheckHealth}
              className={UI_BTN_SECONDARY}
            >
              {checkingHealth ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Activity className="w-3.5 h-3.5 text-status-ready" />
              )}
              <span>Kiểm tra Health</span>
            </button>

            {health && (
              <div className="text-right">
                <div
                  className={`text-xs font-mono font-semibold ${
                    health.online
                      ? health.drift
                        ? "text-status-warn"
                        : "text-status-ready"
                      : "text-status-danger"
                  }`}
                >
                  {health.online
                    ? `ONLINE ${health.status_code} · ${health.latency_ms}ms`
                    : "OFFLINE"}
                </div>
                {health.live_version && (
                  <div className="text-[11px] font-mono text-muted">
                    Live: {health.live_version}
                    {health.drift ? " (Lệch version)" : ""}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      ) : (
        <p className="text-xs text-muted leading-relaxed">
          Chưa khai báo Production URL cho target này. Hãy dùng mục{" "}
          <strong className="text-primary">Targets Phát hành</strong> ở trên để
          nhập Production/Health URL hoặc lưu bản build tĩnh{" "}
          <span className="font-mono">dist.zip</span> lên R2.
        </p>
      )}
    </div>
  );
}

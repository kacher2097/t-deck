import { Loader2, PackageCheck, Rocket, Sparkles, Upload } from "lucide-react";
import type React from "react";
import { api } from "../../lib/api";
import type { ProjectTarget, ProjectView } from "../../types/project";
import { FormField } from "../ui/FormField";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_CARD_BOX,
  UI_MONO_INPUT_CLS,
  UI_SELECT_CLS,
  UI_TEXTAREA_CLS,
} from "../ui/uiTokens";

interface PublishReleaseFormProps {
  project: ProjectView;
  activeTarget?: ProjectTarget;
  isWeb: boolean;
  version: string;
  channel: string;
  assetKind: string;
  sourcePath: string;
  deployUrl: string;
  changelog: string;
  publishing: boolean;
  suggestingNotes: boolean;
  buildRunning: boolean;
  onVersionChange: (v: string) => void;
  onChannelChange: (v: string) => void;
  onAssetKindChange: (v: string) => void;
  onSourcePathChange: (v: string) => void;
  onDeployUrlChange: (v: string) => void;
  onChangelogChange: (v: string) => void;
  onSuggestChangelog: () => void;
  onPublish: (e: React.FormEvent) => void;
  onBuildAndPublish: () => void;
}

export function PublishReleaseForm({
  project,
  activeTarget,
  isWeb,
  version,
  channel,
  assetKind,
  sourcePath,
  deployUrl,
  changelog,
  publishing,
  suggestingNotes,
  buildRunning,
  onVersionChange,
  onChannelChange,
  onAssetKindChange,
  onSourcePathChange,
  onDeployUrlChange,
  onChangelogChange,
  onSuggestChangelog,
  onPublish,
  onBuildAndPublish,
}: PublishReleaseFormProps) {
  return (
    <form onSubmit={onPublish} className={`${UI_CARD_BOX} space-y-4`}>
      <div className="text-sm font-heading font-semibold text-primary flex items-center gap-2">
        {isWeb ? (
          <PackageCheck className="w-4 h-4 text-status-ready shrink-0" />
        ) : (
          <Rocket className="w-4 h-4 shrink-0" />
        )}
        <span>
          {isWeb
            ? "Phát hành / Lưu trữ bản Web Release"
            : "Publish Desktop Release lên Cloudflare R2"}
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-12 gap-3 items-start">
        <div className="sm:col-span-4 min-w-0">
          <FormField label="Version (VD: v1.2.0)">
            <input
              type="text"
              required
              value={version}
              onChange={(e) => onVersionChange(e.target.value)}
              placeholder="v1.0.0"
              className={UI_MONO_INPUT_CLS}
            />
          </FormField>
        </div>

        <div className="sm:col-span-3 min-w-0">
          <FormField label="Channel">
            <select
              value={channel}
              onChange={(e) => onChannelChange(e.target.value)}
              className={UI_SELECT_CLS}
            >
              <option value="stable">stable</option>
              <option value="beta">beta</option>
              <option value="canary">canary</option>
            </select>
          </FormField>
        </div>

        <div className="sm:col-span-5 min-w-0">
          <FormField label="Loại đóng gói">
            <select
              value={assetKind}
              onChange={(e) => onAssetKindChange(e.target.value)}
              className={UI_SELECT_CLS}
            >
              {isWeb ? (
                <>
                  <option value="static_zip">Đóng gói dist.zip lên R2</option>
                  <option value="external_url">Chỉ ghi log Deploy URL</option>
                </>
              ) : (
                <>
                  <option value="portable_zip">Portable (.zip)</option>
                  <option value="installer_exe">Installer (.exe)</option>
                </>
              )}
            </select>
          </FormField>
        </div>
      </div>

      {assetKind !== "external_url" ? (
        <FormField
          label={
            isWeb
              ? "Thư mục build tĩnh (dist / build)"
              : "File .exe hoặc thư mục build"
          }
        >
          <div className="flex items-center gap-2 min-w-0">
            <input
              type="text"
              required
              value={sourcePath}
              onChange={(e) => onSourcePathChange(e.target.value)}
              placeholder={
                isWeb
                  ? "D:\\Projects\\my-web\\dist"
                  : "D:\\Builds\\MyApp.exe hoặc thư mục dist"
              }
              className={`${UI_MONO_INPUT_CLS} flex-1 min-w-0`}
            />
            {!isWeb && (
              <button
                type="button"
                onClick={async () => {
                  const f = await api.selectExecutableFile(
                    "Chọn file .exe để đóng gói"
                  );
                  if (f) onSourcePathChange(f);
                }}
                className={UI_BTN_SECONDARY}
              >
                .exe
              </button>
            )}
            <button
              type="button"
              onClick={async () => {
                const d = await api.selectDirectory(
                  "Chọn thư mục build để nén .zip"
                );
                if (d) onSourcePathChange(d);
              }}
              className={UI_BTN_SECONDARY}
            >
              Thư mục
            </button>
          </div>
        </FormField>
      ) : (
        <FormField label="Deploy / Production URL">
          <input
            type="url"
            required
            value={deployUrl}
            onChange={(e) => onDeployUrlChange(e.target.value)}
            placeholder="https://app.example.com"
            className={UI_MONO_INPUT_CLS}
          />
        </FormField>
      )}

      <FormField
        label="Release Notes / Changelog"
        hint={
          project.local_dir ? "Có thể lấy tự động từ Git commits" : undefined
        }
      >
        <textarea
          rows={5}
          value={changelog}
          onChange={(e) => onChangelogChange(e.target.value)}
          placeholder="Thay đổi trong bản phát hành này..."
          className={UI_TEXTAREA_CLS}
        />
      </FormField>

      <div className="pt-3 border-t border-edge flex items-center justify-between gap-2.5 flex-wrap">
        <div>
          {project.local_dir && (
            <button
              type="button"
              disabled={suggestingNotes}
              onClick={onSuggestChangelog}
              className={UI_BTN_SECONDARY}
            >
              {suggestingNotes ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Sparkles className="w-3.5 h-3.5 text-status-ready" />
              )}
              <span>Gợi ý từ Git Log</span>
            </button>
          )}
        </div>

        <div className="flex items-center gap-2.5 flex-wrap ml-auto">
          {activeTarget?.build_cmd && (
            <button
              type="button"
              disabled={buildRunning}
              onClick={onBuildAndPublish}
              className={UI_BTN_SECONDARY}
            >
              <Rocket className="w-4 h-4 text-status-ready" />
              <span>1-Click Build & Publish</span>
            </button>
          )}

          <button
            type="submit"
            disabled={publishing}
            className={UI_BTN_PRIMARY}
          >
            {publishing ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Upload className="w-4 h-4" />
            )}
            <span>
              {publishing
                ? "Đang xử lý & Upload..."
                : assetKind === "external_url"
                ? "Ghi nhận bản Release Web"
                : "Đóng gói & Publish lên R2"}
            </span>
          </button>
        </div>
      </div>
    </form>
  );
}

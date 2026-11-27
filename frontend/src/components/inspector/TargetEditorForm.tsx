import type { ProjectTarget } from "../../types/project";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_INPUT_CLS,
  UI_MONO_INPUT_CLS,
  UI_SELECT_CLS,
} from "../ui/uiTokens";

interface TargetEditorFormProps {
  target: ProjectTarget;
  saving: boolean;
  onChange: (next: ProjectTarget) => void;
  onCancel: () => void;
  onSave: () => void;
}

export function TargetEditorForm({
  target,
  saving,
  onChange,
  onCancel,
  onSave,
}: TargetEditorFormProps) {
  return (
    <div className="pt-3 border-t border-edge space-y-2.5">
      <div className="text-xs font-semibold text-primary">
        {target.id ? "Sửa Target" : "Thêm Target mới"} ({target.kind.toUpperCase()})
      </div>
      <div className="grid grid-cols-2 gap-2 min-w-0">
        <input
          type="text"
          value={target.name}
          onChange={(e) => onChange({ ...target, name: e.target.value })}
          placeholder="Tên target"
          className={UI_INPUT_CLS}
        />
        <select
          value={target.distribution}
          onChange={(e) => onChange({ ...target, distribution: e.target.value })}
          className={UI_SELECT_CLS}
        >
          {target.kind === "web" ? (
            <>
              <option value="hosted">Hosted URL</option>
              <option value="static_zip">Static Zip (R2)</option>
            </>
          ) : (
            <>
              <option value="portable">Portable (.zip)</option>
              <option value="installer">Installer (.exe)</option>
            </>
          )}
        </select>
      </div>

      {target.kind === "web" && (
        <>
          <input
            type="url"
            value={target.prod_url || ""}
            onChange={(e) => onChange({ ...target, prod_url: e.target.value })}
            placeholder="Production URL (https://...)"
            className={UI_MONO_INPUT_CLS}
          />
          <div className="grid grid-cols-2 gap-2">
            <input
              type="url"
              value={target.staging_url || ""}
              onChange={(e) =>
                onChange({ ...target, staging_url: e.target.value })
              }
              placeholder="Staging URL"
              className={UI_MONO_INPUT_CLS}
            />
            <input
              type="text"
              value={target.hosting_provider || ""}
              onChange={(e) =>
                onChange({ ...target, hosting_provider: e.target.value })
              }
              placeholder="Hosting (CF Pages, Vercel...)"
              className={UI_INPUT_CLS}
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <input
              type="url"
              value={target.health_url || ""}
              onChange={(e) =>
                onChange({ ...target, health_url: e.target.value })
              }
              placeholder="Health URL (/health)"
              className={UI_MONO_INPUT_CLS}
            />
            <input
              type="url"
              value={target.version_url || ""}
              onChange={(e) =>
                onChange({ ...target, version_url: e.target.value })
              }
              placeholder="Version URL (/version.json)"
              className={UI_MONO_INPUT_CLS}
            />
          </div>
        </>
      )}

      <div className="grid grid-cols-2 gap-2">
        <input
          type="text"
          value={target.subpath || ""}
          onChange={(e) => onChange({ ...target, subpath: e.target.value })}
          placeholder="Subpath (VD: frontend)"
          className={UI_MONO_INPUT_CLS}
        />
        <input
          type="text"
          value={target.artifact_glob || ""}
          onChange={(e) =>
            onChange({ ...target, artifact_glob: e.target.value })
          }
          placeholder="Artifact (VD: dist hoặc build/bin/*.exe)"
          className={UI_MONO_INPUT_CLS}
        />
      </div>

      <input
        type="text"
        value={target.build_cmd || ""}
        onChange={(e) => onChange({ ...target, build_cmd: e.target.value })}
        placeholder="Build Cmd (VD: npm run build hoặc wails build -clean)"
        className={UI_MONO_INPUT_CLS}
      />

      <div className="flex justify-end gap-2">
        <button type="button" onClick={onCancel} className={UI_BTN_SECONDARY}>
          Hủy
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={onSave}
          className={UI_BTN_PRIMARY}
        >
          {saving ? "Đang lưu..." : "Lưu Target"}
        </button>
      </div>
    </div>
  );
}


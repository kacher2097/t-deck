import { Globe, Laptop, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { api, formatError } from "../../lib/api";
import type { ProjectTarget, ProjectView } from "../../types/project";
import { ConfirmPopover } from "../ui/ConfirmPopover";
import { UI_CARD_BOX, UI_SECTION_TITLE_CLS } from "../ui/uiTokens";
import { TargetEditorForm } from "./TargetEditorForm";

interface TargetListEditorProps {
  project: ProjectView;
  batFile: string;
  exeName: string;
  onUpdated: (updated: ProjectView) => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function TargetListEditor({
  project,
  batFile,
  exeName,
  onUpdated,
  onNotify,
}: TargetListEditorProps) {
  const [targets, setTargets] = useState<ProjectTarget[]>(
    project.targets || []
  );
  const [editingTarget, setEditingTarget] = useState<ProjectTarget | null>(
    null
  );
  const [savingTarget, setSavingTarget] = useState(false);

  useEffect(() => {
    setTargets(project.targets || []);
    setEditingTarget(null);
  }, [project]);

  const handleNewTarget = (kind: "web" | "desktop") => {
    setEditingTarget({
      id: "",
      project_id: project.id,
      kind,
      name: kind === "web" ? "Web Portal" : "Desktop Client",
      platform: kind === "web" ? "web" : "windows",
      distribution: kind === "web" ? "hosted" : "portable",
      subpath: "",
      build_cmd: "",
      artifact_glob: "",
      launch_entry: kind === "web" ? batFile || "start.bat" : exeName || "",
      prod_url: "",
      staging_url: "",
      health_url: "",
      version_url: "",
      hosting_provider: "",
      sort_order: targets.length,
      updated_at: 0,
    });
  };

  const handleSaveTarget = async () => {
    if (!editingTarget) return;
    setSavingTarget(true);
    try {
      await api.upsertTarget(editingTarget);
      const refreshed = await api.getProjectDetail(project.id);
      setTargets(refreshed.targets || []);
      setEditingTarget(null);
      onUpdated(refreshed);
      onNotify("success", `Đã lưu target "${editingTarget.name}".`);
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSavingTarget(false);
    }
  };

  const handleDeleteTarget = async (target: ProjectTarget) => {
    try {
      await api.deleteTarget(target.id);
      const refreshed = await api.getProjectDetail(project.id);
      setTargets(refreshed.targets || []);
      if (editingTarget?.id === target.id) setEditingTarget(null);
      onUpdated(refreshed);
      onNotify("success", `Đã xóa target "${target.name}".`);
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  return (
    <div className={`${UI_CARD_BOX} space-y-3`}>
      <div className="flex items-center justify-between gap-2">
        <span className={UI_SECTION_TITLE_CLS}>
          Targets Phát hành ({targets.length})
        </span>
        <div className="flex items-center gap-1.5 shrink-0">
          <button
            type="button"
            onClick={() => handleNewTarget("web")}
            className="h-7 px-2.5 shrink-0 whitespace-nowrap rounded border border-edge bg-surface hover:bg-elevated text-xs text-primary inline-flex items-center gap-1"
          >
            <Plus className="w-3 h-3" />
            <span>Web</span>
          </button>
          <button
            type="button"
            onClick={() => handleNewTarget("desktop")}
            className="h-7 px-2.5 shrink-0 whitespace-nowrap rounded border border-edge bg-surface hover:bg-elevated text-xs text-primary inline-flex items-center gap-1"
          >
            <Plus className="w-3 h-3" />
            <span>Desktop</span>
          </button>
        </div>
      </div>

      {targets.length === 0 ? (
        <p className="text-xs text-muted">
          Chưa khai báo target. Bấm + Web hoặc + Desktop để quản lý song song.
        </p>
      ) : (
        <div className="space-y-2">
          {targets.map((t) => (
            <div
              key={t.id}
              className="p-2.5 rounded-md bg-surface border border-edge flex items-center justify-between gap-2"
            >
              <button
                type="button"
                onClick={() => setEditingTarget({ ...t })}
                className="flex-1 min-w-0 text-left flex items-center gap-2"
              >
                {t.kind === "web" ? (
                  <Globe className="w-3.5 h-3.5 text-status-ready shrink-0" />
                ) : (
                  <Laptop className="w-3.5 h-3.5 text-primary shrink-0" />
                )}
                <div className="min-w-0">
                  <div className="text-xs font-medium text-primary truncate">
                    {t.name}
                  </div>
                  <div className="text-[11px] font-mono text-muted truncate">
                    {t.kind === "web"
                      ? t.prod_url || t.launch_entry || "hosted"
                      : t.launch_entry || t.distribution}
                  </div>
                </div>
              </button>
              <ConfirmPopover
                title={`Xóa target "${t.name}"?`}
                description="Cấu hình target phát hành này sẽ bị gỡ khỏi dự án."
                confirmLabel="Xóa target"
                onConfirm={() => handleDeleteTarget(t)}
              >
                {({ open, toggle }) => (
                  <button
                    type="button"
                    onClick={toggle}
                    title="Xóa target"
                    aria-label={`Xóa target ${t.name}`}
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
          ))}
        </div>
      )}

      {editingTarget && (
        <TargetEditorForm
          target={editingTarget}
          saving={savingTarget}
          onChange={setEditingTarget}
          onCancel={() => setEditingTarget(null)}
          onSave={handleSaveTarget}
        />
      )}
    </div>
  );
}

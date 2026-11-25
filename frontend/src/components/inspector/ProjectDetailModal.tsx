import { useEffect, useState } from "react";
import type { Category, ProjectView } from "../../types/project";
import { ErrorBoundary } from "../ui/ErrorBoundary";
import { ModalDialog } from "../ui/ModalDialog";
import { InspectorConfigTab } from "./InspectorConfigTab";
import { InspectorReleaseTab } from "./InspectorReleaseTab";

interface ProjectDetailModalProps {
  project: ProjectView | null;
  categories: Category[];
  onClose: () => void;
  onUpdated: (updated: ProjectView) => void;
  onDeleted: (projectId: string) => void;
  onCreateCategory: (name: string) => Promise<Category | null>;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

type ModalTab = "config" | "release";

const STATUS_BADGE: Record<ProjectView["status"], { label: string; cls: string }> = {
  ready: {
    label: "Sẵn sàng",
    cls: "border-status-ready/40 bg-status-ready/10 text-status-ready",
  },
  missing_env: {
    label: "Thiếu .env",
    cls: "border-status-warn/40 bg-status-warn/10 text-status-warn",
  },
  cloud_only: {
    label: "Cloud Only",
    cls: "border-edge bg-canvas text-muted",
  },
  missing_dir: {
    label: "Thiếu thư mục",
    cls: "border-status-danger/40 bg-status-danger/10 text-status-danger",
  },
};

export function ProjectDetailModal({
  project,
  categories,
  onClose,
  onUpdated,
  onDeleted,
  onCreateCategory,
  onNotify,
}: ProjectDetailModalProps) {
  const [tab, setTab] = useState<ModalTab>("config");

  useEffect(() => {
    if (project) setTab("config");
  }, [project?.id]);

  if (!project) return null;

  const badge = STATUS_BADGE[project.status] ?? STATUS_BADGE.ready;

  return (
    <ModalDialog
      open={Boolean(project)}
      onClose={onClose}
      size="2xl"
      fixedHeight
      title={
        <>
          <span className="px-2 py-0.5 rounded bg-canvas border border-edge text-xs font-mono font-semibold text-primary">
            {project.type === "web"
              ? "WEB"
              : project.type === "hybrid"
              ? "HYBRID"
              : "EXE"}
          </span>
          <h2 className="font-heading text-lg font-semibold text-primary truncate">
            {project.name}
          </h2>
          {project.category && (
            <span className="px-2.5 py-0.5 rounded bg-elevated border border-edge text-xs font-medium text-primary">
              {project.category}
            </span>
          )}
        </>
      }
      subtitle={<span className="font-mono">{project.slug}</span>}
      headerExtra={
        <span
          className={`px-2.5 py-1 rounded border text-xs font-medium ${badge.cls}`}
        >
          {badge.label}
        </span>
      }
      subHeader={
        <div className="px-7 border-b border-edge bg-surface flex items-center gap-2">
          {[
            { id: "config" as const, label: "Cấu hình & Hệ thống" },
            { id: "release" as const, label: "Phát hành & Targets (R2)" },
          ].map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setTab(item.id)}
              className={`px-4 py-3 text-[13px] font-medium border-b-2 transition-colors whitespace-nowrap ${
                tab === item.id
                  ? "border-accent text-primary"
                  : "border-transparent text-muted hover:text-primary"
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      }
    >
      <div className="px-7 py-6">
        <ErrorBoundary scope={`inspector-${tab}`} key={`${project.id}-${tab}`}>
          {tab === "config" && (
            <InspectorConfigTab
              project={project}
              categories={categories}
              onUpdated={onUpdated}
              onDeleted={onDeleted}
              onCreateCategory={onCreateCategory}
              onNotify={onNotify}
            />
          )}
          {tab === "release" && (
            <InspectorReleaseTab
              project={project}
              onUpdated={onUpdated}
              onNotify={onNotify}
            />
          )}
        </ErrorBoundary>
      </div>
    </ModalDialog>
  );
}

import { AppWindow, Globe, Hash, Layers } from "lucide-react";
import { useMemo } from "react";
import { computeSidebarCounts } from "../lib/facets";
import type { Category, ProjectView } from "../types/project";
import { SidebarCategorySection } from "./sidebar/SidebarCategorySection";
import { UI_SECTION_TITLE_CLS } from "./ui/uiTokens";

interface SidebarNavProps {
  queryMatchedProjects: ProjectView[];
  categories: Category[];
  selectedType: string;
  selectedCategory: string;
  selectedStatus: string;
  selectedTag: string;
  selectedQuickFilter: "all" | "has_release" | "needs_update" | "running";
  runningIds: Set<string>;
  onSelectType: (type: string) => void;
  onSelectCategory: (category: string) => void;
  onSelectStatus: (status: string) => void;
  onSelectTag: (tag: string) => void;
  onSelectQuickFilter: (
    qf: "all" | "has_release" | "needs_update" | "running"
  ) => void;
  onCreateCategory: (name: string) => Promise<void>;
  onDeleteCategory: (cat: Category) => Promise<void>;
}

const COUNT_SLOT_CLS =
  "w-7 text-right text-xs font-medium text-muted tabular-nums shrink-0";

export function SidebarNav({
  queryMatchedProjects,
  categories,
  selectedType,
  selectedCategory,
  selectedStatus,
  selectedTag,
  selectedQuickFilter,
  runningIds,
  onSelectType,
  onSelectCategory,
  onSelectStatus,
  onSelectTag,
  onSelectQuickFilter,
  onCreateCategory,
  onDeleteCategory,
}: SidebarNavProps) {
  const counts = useMemo(
    () => computeSidebarCounts(queryMatchedProjects, selectedTag),
    [queryMatchedProjects, selectedTag]
  );

  const typeItems = [
    { id: "all", label: "Tất cả", count: counts.total, icon: Layers },
    { id: "web", label: "Web Apps", count: counts.web, icon: Globe },
    {
      id: "desktop",
      label: "Desktop Apps",
      count: counts.desktop,
      icon: AppWindow,
    },
    {
      id: "hybrid",
      label: "Hybrid (Web + EXE)",
      count: counts.hybrid,
      icon: Layers,
    },
  ];

  const statusItems = [
    {
      id: "all",
      label: "Mọi trạng thái",
      count: counts.total,
      dotClass: "bg-muted",
    },
    {
      id: "ready",
      label: "Sẵn sàng",
      count: counts.ready,
      dotClass: "bg-status-ready",
    },
    {
      id: "cloud_only",
      label: "Chưa tải (Cloud)",
      count: counts.cloudOnly,
      dotClass: "bg-status-cloud",
    },
    {
      id: "missing_env",
      label: "Thiếu .env",
      count: counts.missingEnv,
      dotClass: "bg-status-warn",
    },
    {
      id: "missing_dir",
      label: "Thiếu thư mục",
      count: counts.missingDir,
      dotClass: "bg-status-danger",
    },
  ];

  return (
    <aside className="w-[248px] shrink-0 bg-surface border-r border-edge flex flex-col overflow-y-auto select-none">
      <SidebarCategorySection
        categories={categories}
        selectedCategory={selectedCategory}
        totalCount={counts.total}
        uncategorizedCount={counts.uncategorized}
        categoryCounts={counts.categoryMap}
        countSlotClass={COUNT_SLOT_CLS}
        onSelectCategory={onSelectCategory}
        onCreateCategory={onCreateCategory}
        onDeleteCategory={onDeleteCategory}
      />

      {/* Runtime Environment */}
      <div className="px-3.5 py-3 border-b border-edge space-y-1.5">
        <div className={`px-1 ${UI_SECTION_TITLE_CLS}`}>Môi trường</div>
        <nav className="space-y-0.5">
          {typeItems.map((item) => {
            const Icon = item.icon;
            const active = selectedType === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelectType(item.id)}
                className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${active
                    ? "bg-elevated text-primary font-medium border border-edge"
                    : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
                  }`}
              >
                <span className="flex items-center gap-2.5 min-w-0">
                  <Icon className="w-4 h-4 shrink-0" />
                  <span className="truncate">{item.label}</span>
                </span>
                <span className={COUNT_SLOT_CLS}>{item.count}</span>
              </button>
            );
          })}
        </nav>
      </div>

      {/* Status */}
      <div className="px-3.5 py-3 border-b border-edge space-y-1.5">
        <div className={`px-1 ${UI_SECTION_TITLE_CLS}`}>Trạng thái</div>
        <nav className="space-y-0.5">
          {statusItems.map((item) => {
            const active = selectedStatus === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelectStatus(item.id)}
                className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${active
                    ? "bg-elevated text-primary font-medium border border-edge"
                    : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
                  }`}
              >
                <span className="flex items-center gap-2.5 min-w-0">
                  <span className={`w-2 h-2 rounded-full shrink-0 ${item.dotClass}`} />
                  <span className="truncate">{item.label}</span>
                </span>
                <span className={COUNT_SLOT_CLS}>{item.count}</span>
              </button>
            );
          })}
        </nav>
      </div>

      {/* Release & Runtime Quick Filters */}
      <div className="px-3.5 py-3 border-b border-edge space-y-1.5">
        <div className={`px-1 ${UI_SECTION_TITLE_CLS}`}>Release &amp; Chạy</div>
        <div className="flex flex-wrap gap-1.5">
          {[
            { id: "all" as const, label: "Tất cả" },
            { id: "has_release" as const, label: "Có Release" },
            { id: "needs_update" as const, label: "Cần cập nhật" },
            {
              id: "running" as const,
              label: `Đang chạy (${runningIds.size})`,
            },
          ].map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() =>
                onSelectQuickFilter(
                  selectedQuickFilter === item.id && item.id !== "all"
                    ? "all"
                    : item.id
                )
              }
              className={`px-2 py-1 rounded text-xs border transition-colors ${selectedQuickFilter === item.id
                  ? "bg-elevated border-muted text-primary font-medium"
                  : "bg-canvas border-edge text-muted hover:text-primary"
                }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      </div>

      {/* Faceted #Tags */}
      <div className="px-3.5 py-3 flex-1 space-y-1.5">
        <div className="px-1 flex items-center justify-between">
          <span className={UI_SECTION_TITLE_CLS}>
            Tags ({counts.sortedTags.length})
          </span>
          {selectedTag && (
            <button
              type="button"
              onClick={() => onSelectTag("")}
              className="text-xs text-muted hover:text-primary underline"
            >
              Bỏ lọc
            </button>
          )}
        </div>

        {counts.sortedTags.length === 0 ? (
          <p className="px-1 text-[13px] text-muted/70">Chưa có tag nào</p>
        ) : (
          <div className="space-y-1">
            {counts.sortedTags.map(([tag, count]) => {
              const active = selectedTag.toLowerCase() === tag.toLowerCase();
              return (
                <button
                  key={tag}
                  type="button"
                  onClick={() => onSelectTag(active ? "" : tag)}
                  className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${active
                      ? "bg-elevated text-primary font-medium border border-edge"
                      : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
                    }`}
                >
                  <span className="flex items-center gap-2 min-w-0">
                    <Hash className="w-3.5 h-3.5 shrink-0 text-muted" />
                    <span className="truncate">{tag}</span>
                  </span>
                  <span className={COUNT_SLOT_CLS}>{count}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </aside>
  );
}

import { AppWindow, ChevronDown, Globe, Hash, Layers } from "lucide-react";
import { useMemo, useState } from "react";
import { computeSidebarCounts } from "../lib/facets";
import type { Category, ProjectView } from "../types/project";
import { SidebarCategorySection } from "./sidebar/SidebarCategorySection";
import { UI_SECTION_TITLE_CLS } from "./ui/uiTokens";

interface SidebarNavProps {
  collapsed: boolean;
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
  collapsed,
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
  const [openSections, setOpenSections] = useState({
    env: true,
    status: true,
    release: true,
    tags: true,
  });

  const toggleSection = (key: keyof typeof openSections) => {
    setOpenSections((prev) => ({ ...prev, [key]: !prev[key] }));
  };

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
    <aside
      aria-hidden={collapsed}
      className={`shrink-0 overflow-hidden bg-surface select-none transition-[width,border-color] duration-[250ms] ease-[cubic-bezier(0.22,1,0.36,1)] ${
        collapsed
          ? "w-0 border-r border-transparent pointer-events-none"
          : "w-[248px] border-r border-edge"
      }`}
    >
      <div
        className={`w-[248px] min-w-[248px] h-full flex flex-col overflow-y-auto overflow-x-hidden transition-[transform,opacity] duration-[250ms] ease-[cubic-bezier(0.22,1,0.36,1)] will-change-transform ${
          collapsed ? "-translate-x-5 opacity-0" : "translate-x-0 opacity-100"
        }`}
      >
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
        <div className="px-3.5 py-3 border-b border-edge">
          <button
            type="button"
            onClick={() => toggleSection("env")}
            aria-expanded={openSections.env}
            className="w-full px-1 flex items-center justify-between text-left group"
          >
            <span className={UI_SECTION_TITLE_CLS}>Môi trường</span>
            <ChevronDown
              className={`w-3.5 h-3.5 text-muted group-hover:text-primary transition-transform duration-150 ${
                openSections.env ? "" : "-rotate-90"
              }`}
            />
          </button>
          <div
            className={`grid transition-[grid-template-rows,opacity,margin] duration-200 ease-out ${
              openSections.env
                ? "grid-rows-[1fr] opacity-100 mt-1.5"
                : "grid-rows-[0fr] opacity-0 mt-0 pointer-events-none"
            }`}
          >
            <div className="overflow-hidden min-h-0">
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
                          ? "bg-white/[0.08] text-white font-medium border border-white/15"
                          : item.count === 0
                            ? "text-muted/45 hover:text-primary hover:bg-elevated/60 border border-transparent"
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
          </div>
        </div>

        {/* Status */}
        <div className="px-3.5 py-3 border-b border-edge">
          <button
            type="button"
            onClick={() => toggleSection("status")}
            aria-expanded={openSections.status}
            className="w-full px-1 flex items-center justify-between text-left group"
          >
            <span className={UI_SECTION_TITLE_CLS}>Trạng thái</span>
            <ChevronDown
              className={`w-3.5 h-3.5 text-muted group-hover:text-primary transition-transform duration-150 ${
                openSections.status ? "" : "-rotate-90"
              }`}
            />
          </button>
          <div
            className={`grid transition-[grid-template-rows,opacity,margin] duration-200 ease-out ${
              openSections.status
                ? "grid-rows-[1fr] opacity-100 mt-1.5"
                : "grid-rows-[0fr] opacity-0 mt-0 pointer-events-none"
            }`}
          >
            <div className="overflow-hidden min-h-0">
              <nav className="space-y-0.5">
                {statusItems.map((item) => {
                  const active = selectedStatus === item.id;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => onSelectStatus(item.id)}
                      className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${active
                          ? "bg-white/[0.08] text-white font-medium border border-white/15"
                          : item.count === 0
                            ? "text-muted/45 hover:text-primary hover:bg-elevated/60 border border-transparent"
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
          </div>
        </div>

        {/* Release & Runtime Quick Filters */}
        <div className="px-3.5 py-3 border-b border-edge">
          <button
            type="button"
            onClick={() => toggleSection("release")}
            aria-expanded={openSections.release}
            className="w-full px-1 flex items-center justify-between text-left group"
          >
            <span className={UI_SECTION_TITLE_CLS}>Release &amp; Chạy</span>
            <ChevronDown
              className={`w-3.5 h-3.5 text-muted group-hover:text-primary transition-transform duration-150 ${
                openSections.release ? "" : "-rotate-90"
              }`}
            />
          </button>
          <div
            className={`grid transition-[grid-template-rows,opacity,margin] duration-200 ease-out ${
              openSections.release
                ? "grid-rows-[1fr] opacity-100 mt-1.5"
                : "grid-rows-[0fr] opacity-0 mt-0 pointer-events-none"
            }`}
          >
            <div className="overflow-hidden min-h-0">
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
                        ? "bg-white/[0.08] border-white/20 text-white font-medium"
                        : "bg-canvas border-edge text-muted hover:text-primary"
                      }`}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Faceted #Tags */}
        <div className="px-3.5 py-3 flex-1">
          <div className="px-1 flex items-center justify-between gap-2">
            <button
              type="button"
              onClick={() => toggleSection("tags")}
              aria-expanded={openSections.tags}
              className="flex-1 flex items-center justify-between text-left group"
            >
              <span className={UI_SECTION_TITLE_CLS}>
                Tags ({counts.sortedTags.length})
              </span>
              <ChevronDown
                className={`w-3.5 h-3.5 text-muted group-hover:text-primary transition-transform duration-150 ${
                  openSections.tags ? "" : "-rotate-90"
                }`}
              />
            </button>
            {selectedTag && (
              <button
                type="button"
                onClick={() => onSelectTag("")}
                className="text-xs text-muted hover:text-primary underline shrink-0"
              >
                Bỏ lọc
              </button>
            )}
          </div>

          <div
            className={`grid transition-[grid-template-rows,opacity,margin] duration-200 ease-out ${
              openSections.tags
                ? "grid-rows-[1fr] opacity-100 mt-1.5"
                : "grid-rows-[0fr] opacity-0 mt-0 pointer-events-none"
            }`}
          >
            <div className="overflow-hidden min-h-0">
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
                            ? "bg-white/[0.08] text-white font-medium border border-white/15"
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
          </div>
        </div>
      </div>
    </aside>
  );
}

import {
  AppWindow,
  Folder,
  FolderPlus,
  Globe,
  Hash,
  Layers,
  Plus,
  Trash2,
  X,
} from "lucide-react";
import type React from "react";
import { useMemo, useState } from "react";
import type { Category, ProjectView } from "../types/project";
import { ConfirmPopover } from "./ui/ConfirmPopover";
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
  const [addingCat, setAddingCat] = useState(false);
  const [newCatName, setNewCatName] = useState("");
  const [submittingCat, setSubmittingCat] = useState(false);

  // Single stable pass over queryMatchedProjects so sidebar counts NEVER jump when clicking filters.
  const counts = useMemo(() => {
    const categoryMap = new Map<string, number>();
    const tagMap = new Map<string, number>();
    let uncategorized = 0;
    let web = 0;
    let desktop = 0;
    let hybrid = 0;
    let ready = 0;
    let cloudOnly = 0;
    let missingEnv = 0;
    let missingDir = 0;

    for (const p of queryMatchedProjects) {
      const c = (p.category || "").trim();
      if (!c) {
        uncategorized++;
      } else {
        const k = c.toLowerCase();
        categoryMap.set(k, (categoryMap.get(k) || 0) + 1);
      }

      if (p.type === "web") web++;
      else if (p.type === "desktop") desktop++;
      else if (p.type === "hybrid") hybrid++;

      if (p.status === "ready") ready++;
      else if (p.status === "cloud_only") cloudOnly++;
      else if (p.status === "missing_env") missingEnv++;
      else if (p.status === "missing_dir") missingDir++;

      for (const t of p.tags || []) {
        const clean = t.trim();
        if (!clean) continue;
        tagMap.set(clean, (tagMap.get(clean) || 0) + 1);
      }
    }

    if (
      selectedTag &&
      !Array.from(tagMap.keys()).some(
        (t) => t.toLowerCase() === selectedTag.toLowerCase()
      )
    ) {
      tagMap.set(selectedTag, 0);
    }

    const sortedTags = Array.from(tagMap.entries()).sort((a, b) => {
      if (b[1] !== a[1]) return b[1] - a[1];
      return a[0].localeCompare(b[0]);
    });

    return {
      total: queryMatchedProjects.length,
      categoryMap,
      uncategorized,
      web,
      desktop,
      hybrid,
      ready,
      cloudOnly,
      missingEnv,
      missingDir,
      sortedTags,
    };
  }, [queryMatchedProjects, selectedTag]);

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

  const handleAddCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    const clean = newCatName.trim();
    if (!clean) return;
    setSubmittingCat(true);
    try {
      await onCreateCategory(clean);
      setNewCatName("");
      setAddingCat(false);
    } finally {
      setSubmittingCat(false);
    }
  };

  return (
    <aside className="w-[248px] shrink-0 bg-surface border-r border-edge flex flex-col overflow-y-auto select-none">
      {/* Custom Categories / Groups */}
      <div className="p-4 border-b border-edge space-y-2">
        <div className="flex items-center justify-between px-1">
          <span className={UI_SECTION_TITLE_CLS}>Phân loại (Nhóm)</span>
          <button
            type="button"
            onClick={() => setAddingCat((v) => !v)}
            title="Thêm phân loại mới"
            className="text-xs font-medium text-muted hover:text-primary inline-flex items-center gap-1 px-1.5 py-0.5 rounded hover:bg-elevated transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Thêm</span>
          </button>
        </div>

        {addingCat && (
          <form onSubmit={handleAddCategory} className="flex items-center gap-1.5">
            <input
              type="text"
              autoFocus
              value={newCatName}
              onChange={(e) => setNewCatName(e.target.value)}
              placeholder="Tên nhóm mới..."
              className="flex-1 min-w-0 h-8 px-2.5 rounded-md bg-canvas border border-edge text-[13px] text-primary placeholder:text-muted/70 focus:outline-none focus:border-muted"
            />
            <button
              type="submit"
              disabled={submittingCat || !newCatName.trim()}
              className="h-8 px-2.5 rounded-md bg-accent text-canvas text-xs font-semibold disabled:opacity-40"
            >
              Lưu
            </button>
            <button
              type="button"
              onClick={() => {
                setAddingCat(false);
                setNewCatName("");
              }}
              aria-label="Hủy thêm nhóm"
              className="h-8 w-8 rounded-md border border-edge bg-canvas text-muted hover:text-primary flex items-center justify-center"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </form>
        )}

        <nav className="space-y-1">
          <button
            type="button"
            onClick={() => onSelectCategory("all")}
            className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${
              selectedCategory === "all"
                ? "bg-elevated text-primary font-medium border border-edge"
                : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
            }`}
          >
            <span className="flex items-center gap-2.5 min-w-0">
              <FolderPlus className="w-4 h-4 shrink-0" />
              <span className="truncate">Tất cả nhóm</span>
            </span>
            <span className={COUNT_SLOT_CLS}>{counts.total}</span>
          </button>

          {categories.map((cat) => {
            const active =
              selectedCategory.toLowerCase() === cat.name.toLowerCase();
            const count = counts.categoryMap.get(cat.name.toLowerCase()) || 0;
            return (
              <div
                key={cat.id}
                className={`group px-2.5 py-1.5 rounded-md flex items-center justify-between gap-2 transition-colors ${
                  active
                    ? "bg-elevated text-primary font-medium border border-edge"
                    : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
                }`}
              >
                <button
                  type="button"
                  onClick={() => onSelectCategory(active ? "all" : cat.name)}
                  className="flex-1 min-w-0 text-[13px] flex items-center gap-2.5 text-left"
                >
                  <Folder className="w-4 h-4 shrink-0" />
                  <span className="truncate">{cat.name}</span>
                </button>
                {/* Fixed w-7 right slot: swaps count <-> delete button on hover with ZERO horizontal shift */}
                <ConfirmPopover
                  title={`Xóa nhóm "${cat.name}"?`}
                  description={
                    count > 0
                      ? `${count} dự án trong nhóm sẽ chuyển về "Chưa phân nhóm".`
                      : "Nhóm trống sẽ bị gỡ khỏi danh sách."
                  }
                  confirmLabel="Xóa nhóm"
                  onConfirm={() => onDeleteCategory(cat)}
                >
                  {({ open, toggle }) => (
                    <>
                      <span
                        className={`${COUNT_SLOT_CLS} ${
                          open ? "hidden" : "group-hover:hidden"
                        }`}
                      >
                        {count}
                      </span>
                      <button
                        type="button"
                        onClick={toggle}
                        title={`Xóa nhóm "${cat.name}"`}
                        aria-label={`Xóa nhóm ${cat.name}`}
                        aria-expanded={open}
                        className={`${
                          open ? "inline-flex text-status-danger" : "hidden group-hover:inline-flex text-muted"
                        } w-7 h-4 items-center justify-end hover:text-status-danger shrink-0`}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </>
                  )}
                </ConfirmPopover>
              </div>
            );
          })}

          {categories.length > 0 && (
            <button
              type="button"
              onClick={() =>
                onSelectCategory(
                  selectedCategory === "__uncategorized__"
                    ? "all"
                    : "__uncategorized__"
                )
              }
              className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${
                selectedCategory === "__uncategorized__"
                  ? "bg-elevated text-primary font-medium border border-edge"
                  : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
              }`}
            >
              <span className="truncate pl-6">Chưa phân nhóm</span>
              <span className={COUNT_SLOT_CLS}>{counts.uncategorized}</span>
            </button>
          )}
        </nav>
      </div>

      {/* Runtime Environment */}
      <div className="p-4 border-b border-edge space-y-2">
        <div className={`px-1 ${UI_SECTION_TITLE_CLS}`}>Môi trường</div>
        <nav className="space-y-1">
          {typeItems.map((item) => {
            const Icon = item.icon;
            const active = selectedType === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelectType(item.id)}
                className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${
                  active
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
      <div className="p-4 border-b border-edge space-y-2">
        <div className={`px-1 ${UI_SECTION_TITLE_CLS}`}>Trạng thái</div>
        <nav className="space-y-1">
          {statusItems.map((item) => {
            const active = selectedStatus === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelectStatus(item.id)}
                className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${
                  active
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
      <div className="p-4 border-b border-edge space-y-2">
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
              className={`px-2 py-1 rounded text-xs border transition-colors ${
                selectedQuickFilter === item.id
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
      <div className="p-4 flex-1 space-y-2">
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
                  className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${
                    active
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

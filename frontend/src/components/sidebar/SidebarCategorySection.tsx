import { Folder, FolderPlus, Plus, Trash2, X } from "lucide-react";
import type React from "react";
import { useState } from "react";
import type { Category } from "../../types/project";
import { ConfirmPopover } from "../ui/ConfirmPopover";
import { UI_SECTION_TITLE_CLS } from "../ui/uiTokens";

interface SidebarCategorySectionProps {
  categories: Category[];
  selectedCategory: string;
  totalCount: number;
  uncategorizedCount: number;
  categoryCounts: Map<string, number>;
  countSlotClass: string;
  onSelectCategory: (category: string) => void;
  onCreateCategory: (name: string) => Promise<void>;
  onDeleteCategory: (cat: Category) => Promise<void>;
}

export function SidebarCategorySection({
  categories,
  selectedCategory,
  totalCount,
  uncategorizedCount,
  categoryCounts,
  countSlotClass,
  onSelectCategory,
  onCreateCategory,
  onDeleteCategory,
}: SidebarCategorySectionProps) {
  const [addingCat, setAddingCat] = useState(false);
  const [newCatName, setNewCatName] = useState("");
  const [submittingCat, setSubmittingCat] = useState(false);

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
    <div className="px-3.5 py-3 border-b border-edge space-y-1.5">
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
            name="newSidebarCategory"
            aria-label="Tên nhóm mới"
            autoFocus
            value={newCatName}
            onChange={(e) => setNewCatName(e.target.value)}
            placeholder="Tên nhóm mới..."
            className="flex-1 min-w-0 h-8 px-2.5 rounded-md bg-canvas border border-edge text-[13px] text-primary placeholder:text-muted/70 focus:outline-none focus:border-white/50 focus:ring-2 focus:ring-white/10"
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
          className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${selectedCategory === "all"
              ? "bg-white/[0.08] text-white font-medium border border-white/15"
              : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
            }`}
        >
          <span className="flex items-center gap-2.5 min-w-0">
            <FolderPlus className="w-4 h-4 shrink-0" />
            <span className="truncate">Tất cả nhóm</span>
          </span>
          <span className={countSlotClass}>{totalCount}</span>
        </button>

        {categories.map((cat) => {
          const active =
            selectedCategory.toLowerCase() === cat.name.toLowerCase();
          const count = categoryCounts.get(cat.name.toLowerCase()) || 0;
          return (
            <div
              key={cat.id}
              className={`group px-2.5 py-1.5 rounded-md flex items-center justify-between gap-2 transition-colors ${active
                  ? "bg-white/[0.08] text-white font-medium border border-white/15"
                  : count === 0
                    ? "text-muted/45 hover:text-primary hover:bg-elevated/60 border border-transparent"
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
                      className={`${countSlotClass} ${open ? "hidden" : "group-hover:hidden"
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
                      className={`${open
                          ? "inline-flex text-status-danger"
                          : "hidden group-hover:inline-flex text-muted"
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
            className={`w-full px-2.5 py-1.5 rounded-md text-[13px] flex items-center justify-between gap-2 transition-colors ${selectedCategory === "__uncategorized__"
                ? "bg-white/[0.08] text-white font-medium border border-white/15"
                : uncategorizedCount === 0
                  ? "text-muted/45 hover:text-primary hover:bg-elevated/60 border border-transparent"
                  : "text-muted hover:text-primary hover:bg-elevated/60 border border-transparent"
              }`}
          >
            <span className="truncate pl-6">Chưa phân nhóm</span>
            <span className={countSlotClass}>{uncategorizedCount}</span>
          </button>
        )}
      </nav>
    </div>
  );
}


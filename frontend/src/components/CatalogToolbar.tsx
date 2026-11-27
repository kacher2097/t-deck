import { ArrowUpDown, CheckSquare, Trash2, X } from "lucide-react";
import type { ProjectSortBy } from "../lib/facets";

interface CatalogToolbarProps {
  filteredCount: number;
  totalCount: number;
  hasActiveQuery: boolean;
  debouncedQuery: string;
  selectedCategory: string;
  selectedType: string;
  selectedStatus: string;
  selectedTag: string;
  sortBy: ProjectSortBy;
  selectionMode: boolean;
  selectedCount: number;
  allVisibleSelected: boolean;
  onClearQuery: () => void;
  onClearCategory: () => void;
  onClearType: () => void;
  onClearStatus: () => void;
  onClearTag: () => void;
  onSortByChange: (next: ProjectSortBy) => void;
  onEnterSelectionMode: () => void;
  onToggleSelectAllVisible: () => void;
  onRequestBulkDelete: () => void;
  onExitSelection: () => void;
}

export function CatalogToolbar({
  filteredCount,
  totalCount,
  hasActiveQuery,
  debouncedQuery,
  selectedCategory,
  selectedType,
  selectedStatus,
  selectedTag,
  sortBy,
  selectionMode,
  selectedCount,
  allVisibleSelected,
  onClearQuery,
  onClearCategory,
  onClearType,
  onClearStatus,
  onClearTag,
  onSortByChange,
  onEnterSelectionMode,
  onToggleSelectAllVisible,
  onRequestBulkDelete,
  onExitSelection,
}: CatalogToolbarProps) {
  return (
    <div className="mb-5 px-4 py-2.5 rounded-lg bg-surface border border-edge flex items-center justify-between gap-4 flex-wrap">
      <div className="flex items-center gap-3 flex-wrap min-w-0">
        <div className="flex items-center gap-2">
          <span className="inline-flex items-center justify-center min-w-[30px] h-7 px-2 rounded-md bg-elevated border border-edge font-heading text-[15px] font-bold text-primary tabular-nums">
            {filteredCount}
          </span>
          <span className="text-[14px] font-medium text-primary">
            dự án hiển thị
          </span>
          {filteredCount !== totalCount && (
            <span className="text-[13px] text-muted tabular-nums">
              / {totalCount} tổng
            </span>
          )}
        </div>

        {hasActiveQuery && (
          <FilterChip label={`Tìm: "${debouncedQuery}"`} onClear={onClearQuery} />
        )}
        {selectedCategory !== "all" && (
          <FilterChip
            label={`Nhóm: ${
              selectedCategory === "__uncategorized__"
                ? "Chưa phân nhóm"
                : selectedCategory
            }`}
            onClear={onClearCategory}
          />
        )}
        {selectedType !== "all" && (
          <FilterChip
            label={`Loại: ${selectedType.toUpperCase()}`}
            onClear={onClearType}
          />
        )}
        {selectedStatus !== "all" && (
          <FilterChip
            label={`Trạng thái: ${selectedStatus}`}
            onClear={onClearStatus}
          />
        )}
        {selectedTag && (
          <FilterChip label={`#${selectedTag}`} onClear={onClearTag} />
        )}
      </div>

      <div className="flex items-center gap-2.5 flex-wrap">
        <div className="flex items-center gap-2">
          <ArrowUpDown className="w-3.5 h-3.5 text-muted shrink-0" />
          <span className="text-[12.5px] text-muted hidden sm:inline">
            Sắp xếp:
          </span>
          <select
            aria-label="Sắp xếp dự án"
            value={sortBy}
            onChange={(e) => onSortByChange(e.target.value as ProjectSortBy)}
            className="h-8 px-2.5 rounded-md bg-canvas border border-edge text-[13px] font-medium text-primary focus:outline-none focus:border-muted transition-colors"
          >
            <option value="name_asc">Tên dự án (A → Z)</option>
            <option value="name_desc">Tên dự án (Z → A)</option>
            <option value="updated_desc">Cập nhật gần nhất</option>
            <option value="opened_desc">Mở chạy gần đây</option>
            <option value="status_ready">Sẵn sàng chạy trước</option>
            <option value="category_asc">Theo phân loại (Nhóm)</option>
            <option value="type_asc">Theo môi trường (Web / EXE)</option>
            <option value="relevance">Độ khớp tìm kiếm</option>
          </select>
        </div>

        {filteredCount > 0 && (
          <>
            {selectionMode ? (
              <>
                <span className="text-xs text-muted tabular-nums">
                  Đã chọn <strong className="text-primary">{selectedCount}</strong>
                </span>
                <button
                  type="button"
                  onClick={onToggleSelectAllVisible}
                  className="h-8 px-2.5 rounded-md border border-edge bg-canvas hover:bg-elevated text-xs font-medium text-primary transition-colors"
                >
                  {allVisibleSelected
                    ? "Bỏ chọn đang hiện"
                    : `Chọn tất cả (${filteredCount})`}
                </button>
                <button
                  type="button"
                  disabled={selectedCount === 0}
                  onClick={onRequestBulkDelete}
                  className="h-8 px-3 rounded-md bg-status-danger hover:bg-status-danger/90 text-white text-xs font-semibold inline-flex items-center gap-1.5 disabled:opacity-40 transition-colors"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Xóa ({selectedCount})</span>
                </button>
                <button
                  type="button"
                  onClick={onExitSelection}
                  className="h-8 px-2.5 rounded-md border border-edge bg-canvas hover:bg-elevated text-xs text-muted hover:text-primary inline-flex items-center gap-1 transition-colors"
                >
                  <X className="w-3.5 h-3.5" />
                  <span>Xong</span>
                </button>
              </>
            ) : (
              <button
                type="button"
                onClick={onEnterSelectionMode}
                className="h-8 px-3 rounded-md border border-edge bg-canvas hover:bg-elevated text-[13px] font-medium text-muted hover:text-primary inline-flex items-center gap-1.5 transition-colors"
              >
                <CheckSquare className="w-3.5 h-3.5" />
                <span>Chọn nhiều</span>
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function FilterChip({
  label,
  onClear,
}: {
  label: string;
  onClear: () => void;
}) {
  return (
    <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
      {label}
      <button
        type="button"
        onClick={onClear}
        className="text-muted hover:text-primary"
      >
        ×
      </button>
    </span>
  );
}


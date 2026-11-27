import {
  Activity,
  ChevronLeft,
  ChevronRight,
  CloudCog,
  FolderSearch,
  LayoutGrid,
  Loader2,
  Plus,
  Search,
  Settings,
  TableProperties,
  X,
} from "lucide-react";
import type React from "react";
import { UI_BTN_PRIMARY } from "./ui/uiTokens";

interface TopCommandBarProps {
  query: string;
  onQueryChange: (q: string) => void;
  onSearchCommit?: () => void;
  onCompositionChange?: (composing: boolean) => void;
  searchInputRef: React.RefObject<HTMLInputElement | null>;
  viewMode: "grid" | "matrix";
  onViewModeChange: (mode: "grid" | "matrix") => void;
  sidebarCollapsed: boolean;
  onToggleSidebar: () => void;
  scanning: boolean;
  syncing: boolean;
  checkingHealth: boolean;
  lastSyncAt: number | null;
  onScan: () => void;
  onSync: () => void;
  onCheckAllHealth: () => void;
  onAddProject: () => void;
  onOpenSettings: () => void;
}

function formatSyncTime(unixSec: number | null): string {
  if (!unixSec) return "Sync R2";
  const d = new Date(unixSec * 1000);
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  return `Sync ${hh}:${mm}`;
}

const HEADER_UTILITY_BTN =
  "box-border h-8 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-transparent bg-transparent hover:bg-elevated hover:border-edge text-[13px] font-medium text-muted hover:text-primary inline-flex items-center justify-center gap-1.5 disabled:opacity-50 transition-colors";

export function TopCommandBar({
  query,
  onQueryChange,
  onSearchCommit,
  onCompositionChange,
  searchInputRef,
  viewMode,
  onViewModeChange,
  sidebarCollapsed,
  onToggleSidebar,
  scanning,
  syncing,
  checkingHealth,
  lastSyncAt,
  onScan,
  onSync,
  onCheckAllHealth,
  onAddProject,
  onOpenSettings,
}: TopCommandBarProps) {
  return (
    <header className="h-[58px] shrink-0 bg-surface border-b border-edge px-4 xl:px-5 flex items-center justify-between gap-3 xl:gap-5 select-none">
      {/* Brand + Sidebar Toggle */}
      <div className="flex items-center justify-between gap-2 shrink-0 sm:w-[216px] xl:w-[228px]">
        <div className="flex items-center gap-2.5 min-w-0">
          <svg
            viewBox="0 0 32 32"
            className="w-8 h-8 rounded-md shrink-0"
            aria-hidden="true"
          >
            <rect width="32" height="32" rx="6" fill="#121317" />
            <rect
              x="1"
              y="1"
              width="30"
              height="30"
              rx="5"
              fill="none"
              stroke="#242830"
              strokeWidth="1.5"
            />
            <path d="M8 9h16v3.5h-6V23h-4V12.5H8V9z" fill="#E6E8EC" />
            <rect x="20" y="19" width="3.8" height="3.8" fill="#10B981" />
          </svg>
          <span className="font-heading font-bold text-[16px] tracking-tight text-primary whitespace-nowrap">
            T-Deck
          </span>
        </div>

        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label={sidebarCollapsed ? "Mở rộng Sidebar (Ctrl+B)" : "Thu gọn Sidebar (Ctrl+B)"}
          aria-pressed={sidebarCollapsed}
          title={sidebarCollapsed ? "Mở rộng Sidebar (Ctrl+B)" : "Thu gọn Sidebar (Ctrl+B)"}
          className="h-8 w-8 shrink-0 rounded-md border border-edge bg-canvas hover:bg-elevated hover:border-white/30 text-muted hover:text-primary flex items-center justify-center transition-colors"
        >
          {sidebarCollapsed ? (
            <ChevronRight className="w-4 h-4 stroke-[2.25]" />
          ) : (
            <ChevronLeft className="w-4 h-4 stroke-[2.25]" />
          )}
        </button>
      </div>

      {/* Search (Ctrl+K) */}
      <div className="flex-1 min-w-0 max-w-2xl relative">
        <Search className="w-4 h-4 text-muted absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
        <input
          id="global-project-search"
          name="projectSearch"
          ref={searchInputRef}
          type="text"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          onCompositionStart={() => onCompositionChange?.(true)}
          onCompositionEnd={() => onCompositionChange?.(false)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              onCompositionChange?.(false);
              onSearchCommit?.();
            }
          }}
          placeholder="Tìm theo tên, chức năng (VD: dịch video), tech stack hoặc #tag..."
          aria-label="Tìm kiếm dự án"
          className="w-full h-9 pl-10 pr-20 bg-canvas border border-edge rounded-md text-[13px] text-primary placeholder:text-muted/70 focus:outline-none focus:border-white/50 focus:ring-2 focus:ring-white/10 transition-[border-color,box-shadow]"
        />
        {query ? (
          <button
            type="button"
            onClick={() => onQueryChange("")}
            aria-label="Xóa tìm kiếm"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-primary p-1"
          >
            <X className="w-4 h-4" />
          </button>
        ) : (
          <kbd className="absolute right-3 top-1/2 -translate-y-1/2 px-2 py-0.5 text-[11px] font-mono text-muted bg-surface border border-edge rounded">
            Ctrl+K
          </kbd>
        )}
      </div>

      {/* Action Cluster */}
      <div className="flex items-center gap-1.5 xl:gap-2 shrink-0">
        <div className="h-9 p-0.5 rounded-md border border-edge bg-canvas flex items-center">
          <button
            type="button"
            title="Hiển thị dạng Lưới Card"
            onClick={() => onViewModeChange("grid")}
            className={`h-7 w-7 rounded flex items-center justify-center transition-colors ${
              viewMode === "grid"
                ? "bg-elevated text-primary"
                : "text-muted hover:text-primary"
            }`}
          >
            <LayoutGrid className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            title="Hiển thị dạng Release Matrix"
            onClick={() => onViewModeChange("matrix")}
            className={`h-7 w-7 rounded flex items-center justify-center transition-colors ${
              viewMode === "matrix"
                ? "bg-elevated text-primary"
                : "text-muted hover:text-primary"
            }`}
          >
            <TableProperties className="w-3.5 h-3.5" />
          </button>
        </div>

        <button
          type="button"
          onClick={onCheckAllHealth}
          disabled={checkingHealth}
          title="Kiểm tra trạng thái live của tất cả Web targets"
          className={HEADER_UTILITY_BTN}
        >
          {checkingHealth ? (
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
          ) : (
            <Activity className="w-4 h-4 text-status-ready" />
          )}
          <span className="hidden xl:inline">Health</span>
        </button>

        <button
          type="button"
          onClick={onScan}
          disabled={scanning}
          className={HEADER_UTILITY_BTN}
        >
          {scanning ? (
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
          ) : (
            <FolderSearch className="w-4 h-4 text-muted" />
          )}
          <span>{scanning ? "Đang quét..." : "Scan Local"}</span>
        </button>

        <button
          type="button"
          onClick={onSync}
          disabled={syncing}
          title={
            lastSyncAt
              ? `Đồng bộ gần nhất lúc ${new Date(lastSyncAt * 1000).toLocaleTimeString()}`
              : "Đồng bộ catalog và thumbnails với Cloudflare R2"
          }
          className={HEADER_UTILITY_BTN}
        >
          {syncing ? (
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
          ) : (
            <CloudCog className="w-4 h-4 text-muted" />
          )}
          <span>{syncing ? "Đang Sync..." : formatSyncTime(lastSyncAt)}</span>
        </button>

        <span className="h-5 w-px bg-edge mx-0.5 shrink-0" aria-hidden="true" />

        <button
          type="button"
          onClick={onAddProject}
          className={UI_BTN_PRIMARY}
        >
          <Plus className="w-4 h-4 stroke-[2.5]" />
          <span>Thêm Project</span>
        </button>

        <button
          type="button"
          onClick={onOpenSettings}
          title="Cấu hình Workspace & Cloudflare R2"
          aria-label="Mở cài đặt"
          className="h-9 w-9 shrink-0 rounded-md border border-edge bg-canvas hover:bg-elevated hover:border-white/25 text-muted hover:text-primary flex items-center justify-center transition-colors"
        >
          <Settings className="w-4 h-4" />
        </button>
      </div>
    </header>
  );
}

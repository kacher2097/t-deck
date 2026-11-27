import { FolderSearch, Settings } from "lucide-react";
import { UI_BTN_PRIMARY, UI_BTN_SECONDARY } from "./ui/uiTokens";

interface CatalogEmptyStateProps {
  hasAnyProjects: boolean;
  hasActiveQuery: boolean;
  onOpenSettings: () => void;
  onScan: () => void;
}

export function CatalogEmptyState({
  hasAnyProjects,
  hasActiveQuery,
  onOpenSettings,
  onScan,
}: CatalogEmptyStateProps) {
  return (
    <div className="h-[60vh] border border-dashed border-edge rounded-lg bg-surface/40 flex flex-col items-center justify-center text-center p-8">
      <FolderSearch className="w-9 h-9 text-muted mb-3" />
      <h2 className="font-heading text-base font-semibold text-primary">
        {!hasAnyProjects
          ? "Chưa có dự án nào phù hợp"
          : "Không tìm thấy dự án khớp bộ lọc"}
      </h2>
      <p className="text-[13px] text-muted max-w-md mt-2 leading-relaxed">
        {!hasAnyProjects
          ? "Hãy khai báo thư mục chứa code hoặc Apps trong Cài đặt rồi bấm Scan Local, hoặc cấu hình Cloudflare R2 để đồng bộ từ Cloud."
          : "Thử xóa bớt bộ lọc ở thanh bên trái hoặc điều chỉnh từ khóa tìm kiếm."}
      </p>
      {!hasAnyProjects && !hasActiveQuery && (
        <div className="mt-5 flex items-center gap-3">
          <button
            type="button"
            onClick={onOpenSettings}
            className={UI_BTN_SECONDARY}
          >
            <Settings className="w-4 h-4" />
            <span>Mở Cài đặt</span>
          </button>
          <button type="button" onClick={onScan} className={UI_BTN_PRIMARY}>
            Scan Local ngay
          </button>
        </div>
      )}
    </div>
  );
}


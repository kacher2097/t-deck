import { FolderSearch, Loader2 } from "lucide-react";

interface SkeletonItemProps {
  index: number;
}

export function ProjectCardSkeleton({ index }: SkeletonItemProps) {
  return (
    <div
      aria-hidden="true"
      style={{ animationDelay: `${index * 65}ms` }}
      className="skeleton-stagger-item bg-surface border border-edge rounded-md flex flex-col overflow-hidden min-w-0"
    >
      {/* Header strip */}
      <div className="h-9 px-3 border-b border-edge bg-canvas/50 flex items-center justify-between gap-2 shrink-0">
        <div className="flex items-center gap-1.5">
          <div className="h-5 w-11 rounded skeleton-shimmer" />
          <div className="h-5 w-12 rounded skeleton-shimmer" />
        </div>
        <div className="h-5 w-16 rounded skeleton-shimmer" />
      </div>

      {/* 16:9 preview area */}
      <div className="relative aspect-video bg-canvas border-b border-edge overflow-hidden flex items-center justify-center">
        <div className="absolute inset-0 skeleton-shimmer opacity-35" />
        <div className="w-10 h-10 rounded-lg border border-edge/80 bg-surface/80 flex items-center justify-center relative z-10">
          <FolderSearch className="w-4 h-4 text-muted/40" />
        </div>
      </div>

      {/* Body */}
      <div className="p-3.5 flex-1 flex flex-col gap-2.5">
        <div className="flex items-center justify-between gap-2">
          <div className="h-4 w-3/5 rounded skeleton-shimmer" />
          <div className="h-4 w-16 rounded skeleton-shimmer" />
        </div>
        <div className="space-y-1.5">
          <div className="h-3 w-full rounded skeleton-shimmer" />
          <div className="h-3 w-4/5 rounded skeleton-shimmer" />
        </div>
        <div className="mt-auto pt-1 flex items-center gap-1.5">
          <div className="h-5 w-14 rounded skeleton-shimmer" />
          <div className="h-5 w-16 rounded skeleton-shimmer" />
          <div className="h-5 w-12 rounded skeleton-shimmer" />
        </div>
      </div>

      {/* Footer */}
      <div className="h-[50px] px-3.5 border-t border-edge bg-canvas/40 flex items-center justify-between gap-2 shrink-0">
        <div className="h-8 w-24 rounded-md skeleton-shimmer" />
        <div className="flex items-center gap-1.5">
          <div className="h-7 w-7 rounded-md skeleton-shimmer" />
          <div className="h-7 w-7 rounded-md skeleton-shimmer" />
          <div className="h-7 w-7 rounded-md skeleton-shimmer" />
        </div>
      </div>
    </div>
  );
}

export function ReleaseMatrixSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <div
      aria-hidden="true"
      className="rounded-md border border-edge bg-surface overflow-hidden"
    >
      <div className="h-10 px-4 border-b border-edge bg-canvas flex items-center justify-between gap-4">
        <div className="h-3 w-24 rounded skeleton-shimmer" />
        <div className="h-3 w-28 rounded skeleton-shimmer" />
        <div className="h-3 w-28 rounded skeleton-shimmer" />
        <div className="h-3 w-20 rounded skeleton-shimmer" />
      </div>
      <div className="divide-y divide-edge">
        {Array.from({ length: rows }).map((_, idx) => (
          <div
            key={idx}
            style={{ animationDelay: `${idx * 55}ms` }}
            className="skeleton-stagger-item px-4 py-3 flex items-center justify-between gap-4"
          >
            <div className="flex items-center gap-3 min-w-[210px]">
              <div className="w-12 h-7 rounded skeleton-shimmer shrink-0" />
              <div className="space-y-1.5">
                <div className="h-3.5 w-36 rounded skeleton-shimmer" />
                <div className="h-2.5 w-20 rounded skeleton-shimmer" />
              </div>
            </div>
            <div className="h-5 w-20 rounded skeleton-shimmer" />
            <div className="h-4 w-28 rounded skeleton-shimmer" />
            <div className="flex items-center gap-1.5">
              <div className="h-8 w-20 rounded-md skeleton-shimmer" />
              <div className="h-8 w-8 rounded-md skeleton-shimmer" />
              <div className="h-8 w-8 rounded-md skeleton-shimmer" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ScanLoadingOverlay() {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label="Đang quét dự án cục bộ"
      className="fixed inset-0 z-40 pointer-events-none flex items-center justify-center p-4"
    >
      <div className="w-full max-w-sm rounded-xl border border-status-ready/40 bg-surface/95 backdrop-blur-md shadow-[0_24px_60px_-12px_rgba(0,0,0,0.85)] px-6 py-5 flex flex-col items-center text-center">
        <div className="relative w-14 h-14 flex items-center justify-center mb-3.5">
          <span className="scan-radar-ring absolute inset-0 rounded-full border border-status-ready/60" />
          <span className="w-12 h-12 rounded-full bg-status-ready/15 border border-status-ready/40 flex items-center justify-center">
            <Loader2 className="w-6 h-6 text-status-ready animate-spin" />
          </span>
        </div>
        <h3 className="font-heading text-[15px] font-semibold text-primary">
          Đang quét Workspace cục bộ...
        </h3>
        <p className="text-[12.5px] text-muted mt-1 leading-relaxed">
          Đang phân tích cấu trúc thư mục, nhận diện Tech Stack và lọc dự án thật
        </p>
        <div className="w-full h-1.5 rounded-full bg-canvas border border-edge overflow-hidden mt-4">
          <div className="scan-progress-bar h-full w-2/5 rounded-full bg-status-ready" />
        </div>
      </div>
    </div>
  );
}


import { ImagePlus, Loader2, Upload } from "lucide-react";
import type React from "react";
import { useRef, useState } from "react";
import { imageFileToPNGDataURL } from "../../lib/clipboard";
import type { ProjectView } from "../../types/project";

interface ProjectCardPreviewProps {
  project: ProjectView;
  onOpenDetail: () => void;
  onUploadThumbnail: (project: ProjectView, pngDataUrl: string) => Promise<void>;
  onUploadError: (msg: string) => void;
}

export function ProjectCardPreview({
  project,
  onOpenDetail,
  onUploadThumbnail,
  onUploadError,
}: ProjectCardPreviewProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [dragging, setDragging] = useState(false);

  const savePNG = async (dataUrl: string) => {
    setUploading(true);
    try {
      await onUploadThumbnail(project, dataUrl);
    } finally {
      setUploading(false);
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    try {
      const dataUrl = await imageFileToPNGDataURL(file);
      await savePNG(dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    try {
      const dataUrl = await imageFileToPNGDataURL(file);
      await savePNG(dataUrl);
    } catch (err) {
      onUploadError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,image/bmp"
        onChange={handleFileChange}
        className="hidden"
      />
      <div
        tabIndex={0}
        role="button"
        aria-label={`Ảnh preview ${project.name}. Kéo thả ảnh, chọn file hoặc click và nhấn Ctrl+V`}
        onClick={(e) => e.currentTarget.focus()}
        onDoubleClick={onOpenDetail}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={`relative aspect-video bg-canvas border-b overflow-hidden focus:outline-none focus:ring-1 focus:ring-inset focus:ring-accent/70 cursor-pointer shrink-0 ${
          dragging ? "border-accent bg-elevated/60" : "border-edge"
        }`}
      >
        {project.thumbnail_url ? (
          <img
            src={project.thumbnail_url}
            alt={project.name}
            loading="lazy"
            decoding="async"
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full flex flex-col items-center justify-center gap-2 text-muted/80 px-4 text-center">
            <ImagePlus className="w-5 h-5" />
            <span className="text-xs">
              Kéo thả,{" "}
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  fileInputRef.current?.click();
                }}
                className="underline text-primary hover:text-white"
              >
                tải ảnh
              </button>{" "}
              hoặc Ctrl+V
            </span>
          </div>
        )}

        <button
          type="button"
          title="Tải ảnh từ máy"
          onClick={(e) => {
            e.stopPropagation();
            fileInputRef.current?.click();
          }}
          className="absolute bottom-2 right-2 h-7 px-2.5 rounded bg-canvas/90 border border-edge text-xs font-medium text-primary hover:bg-elevated flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <Upload className="w-3 h-3" />
          <span>Tải ảnh</span>
        </button>

        {uploading && (
          <div className="absolute inset-0 bg-canvas/85 flex items-center justify-center gap-2 text-xs text-primary">
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
            <span>Đang lưu ảnh...</span>
          </div>
        )}
      </div>
    </>
  );
}


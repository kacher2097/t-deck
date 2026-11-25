import { ImagePlus, Loader2, Sparkles, Upload } from "lucide-react";
import type React from "react";
import { useRef, useState } from "react";
import {
  extractPastedImageAsPNG,
  imageFileToPNGDataURL,
} from "../../lib/clipboard";
import { UI_BTN_SECONDARY, UI_SECTION_TITLE_CLS } from "./uiTokens";

interface ThumbnailUploaderProps {
  title?: string;
  imageUrl?: string;
  alt: string;
  onUploadPNG: (pngDataUrl: string) => Promise<void> | void;
  onAutoGenerate?: () => Promise<void> | void;
  onError?: (msg: string) => void;
}

export function ThumbnailUploader({
  title,
  imageUrl,
  alt,
  onUploadPNG,
  onAutoGenerate,
  onError,
}: ThumbnailUploaderProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [dragging, setDragging] = useState(false);

  const processDataUrl = async (dataUrl: string) => {
    setBusy(true);
    try {
      await onUploadPNG(dataUrl);
    } catch (err) {
      onError?.(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const handleAutoGen = async () => {
    if (!onAutoGenerate) return;
    setBusy(true);
    try {
      await onAutoGenerate();
    } catch (err) {
      onError?.(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    try {
      const pngUrl = await imageFileToPNGDataURL(file);
      await processDataUrl(pngUrl);
    } catch (err) {
      onError?.(err instanceof Error ? err.message : String(err));
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    try {
      const pngUrl = await imageFileToPNGDataURL(file);
      await processDataUrl(pngUrl);
    } catch (err) {
      onError?.(err instanceof Error ? err.message : String(err));
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    try {
      const pngUrl = await extractPastedImageAsPNG(e);
      if (!pngUrl) return;
      e.preventDefault();
      e.stopPropagation();
      await processDataUrl(pngUrl);
    } catch (err) {
      onError?.(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        {title && <span className={UI_SECTION_TITLE_CLS}>{title}</span>}
        <div className="flex items-center gap-2 ml-auto">
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={busy}
            className="box-border h-8 px-3 shrink-0 whitespace-nowrap rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center gap-1.5 disabled:opacity-50 transition-colors"
          >
            <Upload className="w-3.5 h-3.5 text-muted" />
            <span>Tải ảnh lên</span>
          </button>

          {onAutoGenerate && (
            <button
              type="button"
              onClick={handleAutoGen}
              disabled={busy}
              className="box-border h-8 px-3 shrink-0 whitespace-nowrap rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center gap-1.5 disabled:opacity-50 transition-colors"
              title="Tạo ảnh đại diện tự động theo mô tả ứng dụng"
            >
              <Sparkles className="w-3.5 h-3.5 text-status-ready" />
              <span>Tạo theo mô tả</span>
            </button>
          )}
        </div>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,image/bmp"
        onChange={handleFileChange}
        className="hidden"
      />

      <div
        tabIndex={0}
        onPaste={handlePaste}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        onClick={(e) => e.currentTarget.focus()}
        className={`relative aspect-video max-h-[195px] w-full rounded-md border border-dashed bg-surface overflow-hidden flex items-center justify-center focus:outline-none focus:border-muted transition-colors cursor-pointer ${
          dragging ? "border-accent bg-elevated/60" : "border-edge"
        }`}
      >
        {imageUrl ? (
          <>
            <img
              src={imageUrl}
              alt={alt}
              className="w-full h-full object-contain bg-canvas"
            />
            <div className="absolute inset-x-0 bottom-0 bg-black/75 px-3 py-1.5 text-xs text-muted flex items-center justify-between opacity-0 hover:opacity-100 transition-opacity">
              <span>Kéo thả ảnh mới, nhấn Ctrl+V hoặc bấm "Tải ảnh lên"</span>
            </div>
          </>
        ) : (
          <div className="text-center px-5 py-4 space-y-2">
            <ImagePlus className="w-6 h-6 text-muted mx-auto" />
            <p className="text-[13px] font-medium text-primary">
              Kéo thả ảnh vào đây hoặc{" "}
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  fileInputRef.current?.click();
                }}
                className="underline hover:text-white"
              >
                chọn từ máy
              </button>
            </p>
            <p className="text-xs text-muted">
              Hỗ trợ PNG, JPG, WebP (tối đa 10 MB) hoặc dán Ctrl+V
            </p>
          </div>
        )}

        {busy && (
          <div className="absolute inset-0 bg-canvas/85 flex items-center justify-center gap-2 text-sm text-primary">
            <Loader2 className="w-4 h-4 animate-spin text-status-ready" />
            <span>Đang xử lý ảnh...</span>
          </div>
        )}
      </div>
    </div>
  );
}

import type React from "react";

const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
const FAST_PATH_PNG_BYTES = 500 * 1024;
const TARGET_MAX_W = 1280;
const TARGET_MAX_H = 720;

/**
 * Extracts an image from a paste event and normalizes it to a PNG data URL.
 */
export async function extractPastedImageAsPNG(
  e: React.ClipboardEvent | ClipboardEvent
): Promise<string | null> {
  const items = e.clipboardData?.items;
  if (!items) return null;

  let imageFile: File | null = null;
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.type.startsWith("image/")) {
      imageFile = item.getAsFile();
      if (imageFile) break;
    }
  }
  if (!imageFile) return null;
  return imageFileToPNGDataURL(imageFile);
}

/**
 * Validates an image file (File or dropped Blob) and normalizes it to a max 1280x720 PNG data URL.
 */
export async function imageFileToPNGDataURL(file: File): Promise<string> {
  if (!file.type.startsWith("image/")) {
    throw new Error("Vui lòng chọn tệp hình ảnh (PNG, JPG, WebP...).");
  }
  if (file.size > MAX_IMAGE_BYTES) {
    throw new Error("Ảnh vượt quá giới hạn 10 MB.");
  }
  return convertOrResizeToPNGDataURL(file);
}

function blobToDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(new Error("Không thể đọc tệp ảnh."));
    reader.readAsDataURL(blob);
  });
}

function convertOrResizeToPNGDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = async () => {
      try {
        const natW = img.naturalWidth || TARGET_MAX_W;
        const natH = img.naturalHeight || TARGET_MAX_H;
        const fits = natW <= TARGET_MAX_W && natH <= TARGET_MAX_H;

        if (file.type === "image/png" && fits && file.size <= FAST_PATH_PNG_BYTES) {
          URL.revokeObjectURL(url);
          resolve(await blobToDataURL(file));
          return;
        }

        const scale = Math.min(1, TARGET_MAX_W / natW, TARGET_MAX_H / natH);
        const outW = Math.max(1, Math.round(natW * scale));
        const outH = Math.max(1, Math.round(natH * scale));

        const canvas = document.createElement("canvas");
        canvas.width = outW;
        canvas.height = outH;
        const ctx = canvas.getContext("2d");
        if (!ctx) {
          reject(new Error("Canvas context unavailable"));
          return;
        }
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = "high";
        ctx.drawImage(img, 0, 0, outW, outH);
        resolve(canvas.toDataURL("image/png"));
      } catch (err) {
        reject(err);
      } finally {
        URL.revokeObjectURL(url);
      }
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("Ảnh không hợp lệ hoặc bị hỏng."));
    };
    img.src = url;
  });
}

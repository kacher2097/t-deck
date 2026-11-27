import { FolderPlus, Loader2, Save, Sparkles } from "lucide-react";
import type React from "react";
import { useState } from "react";
import type { Category, ProjectType } from "../../types/project";
import { FormField } from "../ui/FormField";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_INPUT_CLS,
  UI_MONO_INPUT_CLS,
  UI_SECTION_TITLE_CLS,
  UI_SELECT_CLS,
  UI_TEXTAREA_CLS,
} from "../ui/uiTokens";

interface InspectorMetadataFormProps {
  name: string;
  slug: string;
  type: ProjectType;
  category: string;
  categories: Category[];
  summary: string;
  techStack: string;
  tags: string[];
  gitRemote: string;
  localDir: string;
  batFile: string;
  exeName: string;
  saving: boolean;
  probingDir: boolean;
  onNameChange: (v: string) => void;
  onSlugChange: (v: string) => void;
  onTypeChange: (v: ProjectType) => void;
  onCategoryChange: (v: string) => void;
  onSummaryChange: (v: string) => void;
  onTechStackChange: (v: string) => void;
  onTagsChange: (v: string[]) => void;
  onGitRemoteChange: (v: string) => void;
  onLocalDirChange: (v: string) => void;
  onBatFileChange: (v: string) => void;
  onExeNameChange: (v: string) => void;
  onCreateCategory: (name: string) => Promise<Category | null>;
  onProbeDir: (dir: string, onlyFillEmpty: boolean) => Promise<void>;
  onBrowseDir: () => Promise<void>;
}

export function InspectorMetadataForm({
  name,
  slug,
  type,
  category,
  categories,
  summary,
  techStack,
  tags,
  gitRemote,
  localDir,
  batFile,
  exeName,
  saving,
  probingDir,
  onNameChange,
  onSlugChange,
  onTypeChange,
  onCategoryChange,
  onSummaryChange,
  onTechStackChange,
  onTagsChange,
  onGitRemoteChange,
  onLocalDirChange,
  onBatFileChange,
  onExeNameChange,
  onCreateCategory,
  onProbeDir,
  onBrowseDir,
}: InspectorMetadataFormProps) {
  const [creatingCat, setCreatingCat] = useState(false);
  const [newCatName, setNewCatName] = useState("");
  const [tagInput, setTagInput] = useState("");

  const handleAddTag = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    const clean = tagInput.trim().replace(/^#/, "");
    if (!clean) return;
    if (!tags.some((t) => t.toLowerCase() === clean.toLowerCase())) {
      onTagsChange([...tags, clean]);
    }
    setTagInput("");
  };

  const handleCreateCategoryInline = async () => {
    const clean = newCatName.trim();
    if (!clean) return;
    const created = await onCreateCategory(clean);
    if (created) {
      onCategoryChange(created.name);
      setNewCatName("");
      setCreatingCat(false);
    }
  };

  return (
    <div className="p-5 rounded-md bg-canvas border border-edge space-y-4">
      <div className="flex items-center justify-between gap-2">
        <div className={UI_SECTION_TITLE_CLS}>Thông tin &amp; Cấu hình chung</div>
        <button
          type="button"
          disabled={probingDir || !localDir.trim()}
          onClick={() => void onProbeDir(localDir, false)}
          title="Tự nhận diện lại Tech Stack, Git Remote, Tags, Mô tả từ thư mục dự án"
          className="h-7 px-2.5 rounded border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center gap-1.5 disabled:opacity-50 transition-colors"
        >
          {probingDir ? (
            <Loader2 className="w-3.5 h-3.5 animate-spin" />
          ) : (
            <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
          )}
          <span>Nhận diện từ thư mục</span>
        </button>
      </div>

      {/* Row 1: Tên dự án (5) + Môi trường chạy (3) + Phân loại (4) */}
      <div className="grid grid-cols-1 sm:grid-cols-12 gap-3.5 items-start">
        <div className="sm:col-span-5 min-w-0">
          <FormField label="Tên dự án">
            <input
              type="text"
              required
              value={name}
              onChange={(e) => onNameChange(e.target.value)}
              className={UI_INPUT_CLS}
            />
          </FormField>
        </div>

        <div className="sm:col-span-3 min-w-0">
          <FormField label="Môi trường chạy">
            <select
              value={type}
              onChange={(e) => onTypeChange(e.target.value as ProjectType)}
              className={UI_SELECT_CLS}
            >
              <option value="web">Web App (.bat)</option>
              <option value="desktop">Desktop App (.exe)</option>
              <option value="hybrid">Hybrid (Web + EXE)</option>
            </select>
          </FormField>
        </div>

        <div className="sm:col-span-4 min-w-0">
          <FormField label="Phân loại (Nhóm)">
            {!creatingCat ? (
              <div className="flex items-center gap-1.5 min-w-0">
                <select
                  value={category}
                  onChange={(e) => onCategoryChange(e.target.value)}
                  className={`${UI_SELECT_CLS} flex-1 min-w-0`}
                >
                  <option value="">Chưa phân nhóm</option>
                  {categories.map((c) => (
                    <option key={c.id} value={c.name}>
                      {c.name}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  title="Tạo nhóm dự án mới"
                  onClick={() => setCreatingCat(true)}
                  className="box-border h-9 px-2.5 shrink-0 whitespace-nowrap rounded-md border border-edge bg-surface hover:bg-elevated text-xs font-medium text-primary inline-flex items-center gap-1 transition-colors"
                >
                  <FolderPlus className="w-3.5 h-3.5 text-muted" />
                  <span>Mới</span>
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-1.5 min-w-0">
                <input
                  type="text"
                  name="newCategoryName"
                  aria-label="Tên nhóm mới"
                  autoFocus
                  value={newCatName}
                  onChange={(e) => setNewCatName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      void handleCreateCategoryInline();
                    }
                  }}
                  placeholder="Tên nhóm..."
                  className={`${UI_INPUT_CLS} flex-1 min-w-0`}
                />
                <button
                  type="button"
                  onClick={() => void handleCreateCategoryInline()}
                  className="box-border h-9 px-2.5 shrink-0 rounded-md bg-accent text-canvas text-xs font-semibold"
                >
                  Tạo
                </button>
                <button
                  type="button"
                  onClick={() => setCreatingCat(false)}
                  className="box-border h-9 px-2 shrink-0 rounded-md border border-edge bg-surface text-xs text-muted hover:text-primary"
                >
                  ×
                </button>
              </div>
            )}
          </FormField>
        </div>
      </div>

      {/* Row 2: Slug (4) + Tech Stack (4) + Git Remote URL (4) */}
      <div className="grid grid-cols-1 sm:grid-cols-12 gap-3.5 items-start">
        <div className="sm:col-span-4 min-w-0">
          <FormField label="Slug">
            <input
              type="text"
              value={slug}
              onChange={(e) => onSlugChange(e.target.value)}
              className={UI_MONO_INPUT_CLS}
            />
          </FormField>
        </div>

        <div className="sm:col-span-4 min-w-0">
          <FormField label="Tech Stack">
            <input
              type="text"
              value={techStack}
              onChange={(e) => onTechStackChange(e.target.value)}
              placeholder="Go, Wails, React"
              className={UI_INPUT_CLS}
            />
          </FormField>
        </div>

        <div className="sm:col-span-4 min-w-0">
          <FormField label="Git Remote URL">
            <input
              type="text"
              value={gitRemote}
              onChange={(e) => onGitRemoteChange(e.target.value)}
              placeholder="https://github.com/user/repo.git"
              className={UI_MONO_INPUT_CLS}
            />
          </FormField>
        </div>
      </div>

      {/* Row 3: Mô tả chức năng */}
      <FormField label="Mô tả chức năng" hint="Tìm kiếm toàn văn FTS5">
        <textarea
          rows={3}
          value={summary}
          onChange={(e) => onSummaryChange(e.target.value)}
          className={UI_TEXTAREA_CLS}
        />
      </FormField>

      {/* Row 4: Tags */}
      <FormField label="Tags" hint="Gõ tag rồi nhấn Enter">
        <div className="min-h-[38px] px-2.5 py-1.5 rounded-md bg-surface border border-edge flex flex-wrap items-center gap-1.5">
          {tags.map((t) => (
            <span
              key={t}
              className="px-2.5 py-0.5 rounded bg-canvas border border-edge text-xs text-primary flex items-center gap-1.5"
            >
              #{t}
              <button
                type="button"
                onClick={() => onTagsChange(tags.filter((x) => x !== t))}
                className="text-muted hover:text-primary"
              >
                ×
              </button>
            </span>
          ))}
          <input
            type="text"
            name="tagInput"
            aria-label="Thêm tag mới"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={handleAddTag}
            placeholder="Thêm tag..."
            className="flex-1 min-w-[110px] h-6 bg-transparent text-[13px] text-primary placeholder:text-muted/60 focus:outline-none"
          />
        </div>
      </FormField>

      {/* Row 5: Thư mục trên máy này (7) + File khởi chạy (5) */}
      <div className="grid grid-cols-1 sm:grid-cols-12 gap-3.5 items-start">
        <div className="sm:col-span-7 min-w-0">
          <FormField label="Thư mục trên máy này">
            <div className="flex items-center gap-2 min-w-0">
              <input
                type="text"
                value={localDir}
                onChange={(e) => onLocalDirChange(e.target.value)}
                className={`${UI_MONO_INPUT_CLS} flex-1 min-w-0`}
              />
              <button
                type="button"
                onClick={() => void onBrowseDir()}
                className={UI_BTN_SECONDARY}
              >
                Chọn
              </button>
            </div>
          </FormField>
        </div>

        <div className="sm:col-span-5 min-w-0">
          {type === "web" ? (
            <FormField label="Script khởi chạy (.bat)">
              <input
                type="text"
                value={batFile}
                onChange={(e) => onBatFileChange(e.target.value)}
                className={UI_MONO_INPUT_CLS}
              />
            </FormField>
          ) : type === "desktop" ? (
            <FormField label="Tên file thực thi (.exe)">
              <input
                type="text"
                value={exeName}
                onChange={(e) => onExeNameChange(e.target.value)}
                placeholder="MyApp.exe"
                className={UI_MONO_INPUT_CLS}
              />
            </FormField>
          ) : (
            <FormField label="Entry chạy (.bat / .exe)">
              <div className="grid grid-cols-2 gap-2">
                <input
                  type="text"
                  name="hybridBatFile"
                  aria-label="Script khởi chạy (.bat)"
                  value={batFile}
                  onChange={(e) => onBatFileChange(e.target.value)}
                  placeholder="start.bat"
                  className={UI_MONO_INPUT_CLS}
                />
                <input
                  type="text"
                  name="hybridExeName"
                  aria-label="File thực thi (.exe)"
                  value={exeName}
                  onChange={(e) => onExeNameChange(e.target.value)}
                  placeholder="MyApp.exe"
                  className={UI_MONO_INPUT_CLS}
                />
              </div>
            </FormField>
          )}
        </div>
      </div>

      <div className="pt-3.5 border-t border-edge flex items-center justify-between gap-4">
        <span className="text-xs text-muted">
          Tự động lưu định danh vào{" "}
          <code className="font-mono">.tdeck-meta.json</code>
        </span>
        <button type="submit" disabled={saving} className={UI_BTN_PRIMARY}>
          {saving ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Save className="w-4 h-4" />
          )}
          <span>Lưu cấu hình</span>
        </button>
      </div>
    </div>
  );
}


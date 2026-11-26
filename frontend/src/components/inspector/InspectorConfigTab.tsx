import { FolderPlus, Loader2, Save, Sparkles } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { api, formatError } from "../../lib/api";
import type {
  BatTemplate,
  Category,
  ProjectType,
  ProjectView,
} from "../../types/project";
import { FormField } from "../ui/FormField";
import { ThumbnailUploader } from "../ui/ThumbnailUploader";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_INPUT_CLS,
  UI_MONO_INPUT_CLS,
  UI_SECTION_TITLE_CLS,
  UI_SELECT_CLS,
  UI_TEXTAREA_CLS,
} from "../ui/uiTokens";
import { InspectorLocalToolsCard } from "./InspectorLocalToolsCard";

interface InspectorConfigTabProps {
  project: ProjectView;
  categories: Category[];
  onUpdated: (updated: ProjectView) => void;
  onRequestDelete: (project: ProjectView) => void;
  onCreateCategory: (name: string) => Promise<Category | null>;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function InspectorConfigTab({
  project,
  categories,
  onUpdated,
  onRequestDelete,
  onCreateCategory,
  onNotify,
}: InspectorConfigTabProps) {
  const [templates, setTemplates] = useState<BatTemplate[]>([]);
  const [selectedTpl, setSelectedTpl] = useState("node");

  const [name, setName] = useState(project.name);
  const [slug, setSlug] = useState(project.slug);
  const [type, setType] = useState<ProjectType>(project.type);
  const [category, setCategory] = useState(project.category || "");
  const [creatingCat, setCreatingCat] = useState(false);
  const [newCatName, setNewCatName] = useState("");

  const [summary, setSummary] = useState(project.summary);
  const [techStack, setTechStack] = useState(project.tech_stack);
  const [tags, setTags] = useState<string[]>(project.tags || []);
  const [tagInput, setTagInput] = useState("");
  const [gitRemote, setGitRemote] = useState(project.git_remote);
  const [localDir, setLocalDir] = useState(project.local_dir);
  const [batFile, setBatFile] = useState(project.bat_file || "start.bat");
  const [exeName, setExeName] = useState(project.exe_name);

  const [saving, setSaving] = useState(false);
  const [probingDir, setProbingDir] = useState(false);
  const [syncingEnv, setSyncingEnv] = useState(false);
  const [generatingBat, setGeneratingBat] = useState(false);

  useEffect(() => {
    api
      .getBatTemplates()
      .then((list) => {
        setTemplates(list);
        if (list.length > 0) setSelectedTpl(list[0].id);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    setName(project.name);
    setSlug(project.slug);
    setType(project.type);
    setCategory(project.category || "");
    setCreatingCat(false);
    setNewCatName("");
    setSummary(project.summary);
    setTechStack(project.tech_stack);
    setTags(project.tags || []);
    setTagInput("");
    setGitRemote(project.git_remote);
    setLocalDir(project.local_dir);
    setBatFile(project.bat_file || "start.bat");
    setExeName(project.exe_name);
  }, [project]);

  const handleAddTag = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    const clean = tagInput.trim().replace(/^#/, "");
    if (!clean) return;
    if (!tags.some((t) => t.toLowerCase() === clean.toLowerCase())) {
      setTags([...tags, clean]);
    }
    setTagInput("");
  };

  const handleCreateCategoryInline = async () => {
    const clean = newCatName.trim();
    if (!clean) return;
    const created = await onCreateCategory(clean);
    if (created) {
      setCategory(created.name);
      setNewCatName("");
      setCreatingCat(false);
    }
  };

  const handleProbeDir = async (dirToProbe: string, onlyFillEmpty: boolean) => {
    const trimmed = dirToProbe.trim();
    if (!trimmed) {
      onNotify("warn", "Chưa có đường dẫn thư mục cục bộ để quét.");
      return;
    }
    setProbingDir(true);
    try {
      const probe = await api.inspectLocalDirectory(trimmed);
      const updatedFields: string[] = [];

      if (probe.tech_stack && (!onlyFillEmpty || !techStack.trim()) && probe.tech_stack !== techStack) {
        setTechStack(probe.tech_stack);
        updatedFields.push("Tech Stack");
      }
      if (probe.git_remote && (!onlyFillEmpty || !gitRemote.trim()) && probe.git_remote !== gitRemote) {
        setGitRemote(probe.git_remote);
        updatedFields.push("Git Remote");
      }
      if (probe.summary && (!onlyFillEmpty ? !summary.trim() || probe.summary !== summary : !summary.trim())) {
        if (!summary.trim()) {
          setSummary(probe.summary);
          updatedFields.push("Mô tả");
        }
      }
      if (probe.category && !category.trim()) {
        setCategory(probe.category);
        updatedFields.push("Phân loại");
      }
      if (probe.bat_file && (!batFile.trim() || batFile.trim() === "start.bat") && probe.bat_file !== batFile) {
        setBatFile(probe.bat_file);
        updatedFields.push("Script .bat");
      }
      if (probe.exe_name && !exeName.trim()) {
        setExeName(probe.exe_name);
        updatedFields.push("File .exe");
      }
      if (probe.tags?.length > 0) {
        const merged = [...tags];
        let addedTags = 0;
        for (const t of probe.tags) {
          if (!merged.some((x) => x.toLowerCase() === t.toLowerCase())) {
            merged.push(t);
            addedTags++;
          }
        }
        if (addedTags > 0) {
          setTags(merged);
          updatedFields.push(`+${addedTags} tags`);
        }
      }

      if (updatedFields.length > 0) {
        onNotify(
          "success",
          `Đã nhận diện từ thư mục: ${updatedFields.join(", ")}. Nhấn Lưu cấu hình để áp dụng.`
        );
      } else {
        onNotify("warn", "Không tìm thấy thông tin mới bổ sung từ thư mục.");
      }
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setProbingDir(false);
    }
  };

  const handleSaveConfig = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const updated = await api.upsertProject({
        id: project.id,
        slug,
        name,
        type,
        category,
        summary,
        tags,
        tech_stack: techStack,
        git_remote: gitRemote,
        bat_file: batFile,
        exe_name: exeName,
        local_dir: localDir,
      });
      onUpdated(updated);
      onNotify("success", `Đã lưu cấu hình "${updated.name}".`);
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSaving(false);
    }
  };

  const handleUploadThumbnail = async (pngDataUrl: string) => {
    await api.saveClipboardThumbnail(project.id, pngDataUrl);
    const refreshed = await api.getProjectDetail(project.id);
    onUpdated(refreshed);
    onNotify("success", "Đã cập nhật ảnh preview cho dự án.");
  };

  const handleAutoGenerateThumbnail = async () => {
    await api.generateProjectThumbnail(project.id);
    const refreshed = await api.getProjectDetail(project.id);
    onUpdated(refreshed);
    onNotify("success", "Đã tạo ảnh đại diện mới cho dự án.");
  };

  const handleSyncEnv = async () => {
    setSyncingEnv(true);
    try {
      const res = await api.syncEnvExample(project.id);
      const refreshed = await api.getProjectDetail(project.id);
      onUpdated(refreshed);
      onNotify(
        "success",
        `Đã xuất .env.example (${res.vars_count} biến môi trường, đã xóa giá trị bí mật).`
      );
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSyncingEnv(false);
    }
  };

  const handleGenerateBat = async () => {
    setGeneratingBat(true);
    try {
      await api.generateStartBat(project.id, selectedTpl);
      const refreshed = await api.getProjectDetail(project.id);
      onUpdated(refreshed);
      onNotify("success", `Đã tạo file ${refreshed.bat_file || "start.bat"}.`);
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setGeneratingBat(false);
    }
  };

  return (
    <form
      onSubmit={handleSaveConfig}
      className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start"
    >
      {/* Left Column: Compact Thumbnail Card + Unified Local & System Card */}
      <div className="lg:col-span-5 space-y-5 min-w-0">
        <div className="p-5 rounded-md bg-canvas border border-edge">
          <ThumbnailUploader
            title="Ảnh đại diện ứng dụng"
            imageUrl={project.thumbnail_url}
            alt={project.name}
            onUploadPNG={handleUploadThumbnail}
            onAutoGenerate={handleAutoGenerateThumbnail}
            onError={(msg) => onNotify("error", msg)}
          />
        </div>

        <InspectorLocalToolsCard
          project={project}
          type={type}
          batFile={batFile}
          templates={templates}
          selectedTpl={selectedTpl}
          generatingBat={generatingBat}
          syncingEnv={syncingEnv}
          onSelectTpl={setSelectedTpl}
          onGenerateBat={handleGenerateBat}
          onSyncEnv={handleSyncEnv}
          onRequestDelete={onRequestDelete}
          onNotify={onNotify}
        />
      </div>

      {/* Right Column: Spacious 12-Col Form Card */}
      <div className="lg:col-span-7 min-w-0">
        <div className="p-5 rounded-md bg-canvas border border-edge space-y-4">
          <div className="flex items-center justify-between gap-2">
            <div className={UI_SECTION_TITLE_CLS}>Thông tin & Cấu hình chung</div>
            <button
              type="button"
              disabled={probingDir || !localDir.trim()}
              onClick={() => void handleProbeDir(localDir, false)}
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
                  onChange={(e) => setName(e.target.value)}
                  className={UI_INPUT_CLS}
                />
              </FormField>
            </div>

            <div className="sm:col-span-3 min-w-0">
              <FormField label="Môi trường chạy">
                <select
                  value={type}
                  onChange={(e) => setType(e.target.value as ProjectType)}
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
                      onChange={(e) => setCategory(e.target.value)}
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
                      autoFocus
                      value={newCatName}
                      onChange={(e) => setNewCatName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          handleCreateCategoryInline();
                        }
                      }}
                      placeholder="Tên nhóm..."
                      className={`${UI_INPUT_CLS} flex-1 min-w-0`}
                    />
                    <button
                      type="button"
                      onClick={handleCreateCategoryInline}
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
                  onChange={(e) => setSlug(e.target.value)}
                  className={UI_MONO_INPUT_CLS}
                />
              </FormField>
            </div>

            <div className="sm:col-span-4 min-w-0">
              <FormField label="Tech Stack">
                <input
                  type="text"
                  value={techStack}
                  onChange={(e) => setTechStack(e.target.value)}
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
                  onChange={(e) => setGitRemote(e.target.value)}
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
              onChange={(e) => setSummary(e.target.value)}
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
                    onClick={() => setTags(tags.filter((x) => x !== t))}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              ))}
              <input
                type="text"
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
                    onChange={(e) => setLocalDir(e.target.value)}
                    className={`${UI_MONO_INPUT_CLS} flex-1 min-w-0`}
                  />
                  <button
                    type="button"
                    onClick={async () => {
                      const picked = await api.selectDirectory(
                        "Chọn thư mục dự án"
                      );
                      if (picked) {
                        setLocalDir(picked);
                        await handleProbeDir(picked, true);
                      }
                    }}
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
                    onChange={(e) => setBatFile(e.target.value)}
                    className={UI_MONO_INPUT_CLS}
                  />
                </FormField>
              ) : type === "desktop" ? (
                <FormField label="Tên file thực thi (.exe)">
                  <input
                    type="text"
                    value={exeName}
                    onChange={(e) => setExeName(e.target.value)}
                    placeholder="MyApp.exe"
                    className={UI_MONO_INPUT_CLS}
                  />
                </FormField>
              ) : (
                <FormField label="Entry chạy (.bat / .exe)">
                  <div className="grid grid-cols-2 gap-2">
                    <input
                      type="text"
                      value={batFile}
                      onChange={(e) => setBatFile(e.target.value)}
                      placeholder="start.bat"
                      className={UI_MONO_INPUT_CLS}
                    />
                    <input
                      type="text"
                      value={exeName}
                      onChange={(e) => setExeName(e.target.value)}
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
              Tự động lưu định danh vào <code className="font-mono">.tdeck-meta.json</code>
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
      </div>
    </form>
  );
}

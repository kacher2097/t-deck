import { FolderOpen, Loader2, Plus, Sparkles } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { api, formatError } from "../lib/api";
import type { Category, ProjectType, ProjectView } from "../types/project";
import { FormField } from "./ui/FormField";
import { ModalDialog } from "./ui/ModalDialog";
import {
  UI_BTN_PRIMARY,
  UI_BTN_SECONDARY,
  UI_INPUT_CLS,
  UI_MONO_INPUT_CLS,
  UI_SELECT_CLS,
  UI_TEXTAREA_CLS,
} from "./ui/uiTokens";

interface AddProjectModalProps {
  open: boolean;
  categories: Category[];
  onClose: () => void;
  onCreated: (created: ProjectView) => void;
  onCategoryCreated: () => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function AddProjectModal({
  open,
  categories,
  onClose,
  onCreated,
  onCategoryCreated,
  onNotify,
}: AddProjectModalProps) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [type, setType] = useState<ProjectType>("web");
  const [category, setCategory] = useState("");
  const [localDir, setLocalDir] = useState("");
  const [prodUrl, setProdUrl] = useState("");
  const [gitRemote, setGitRemote] = useState("");
  const [techStack, setTechStack] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [batFile, setBatFile] = useState("start.bat");
  const [exeName, setExeName] = useState("");
  const [summary, setSummary] = useState("");

  const [probing, setProbing] = useState(false);
  const [lastProbedDir, setLastProbedDir] = useState("");
  const [probeSummaryNote, setProbeSummaryNote] = useState("");
  const [existingInDeck, setExistingInDeck] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName("");
    setSlug("");
    setType("web");
    setCategory("");
    setLocalDir("");
    setProdUrl("");
    setGitRemote("");
    setTechStack("");
    setTagsText("");
    setBatFile("start.bat");
    setExeName("");
    setSummary("");
    setProbing(false);
    setLastProbedDir("");
    setProbeSummaryNote("");
    setExistingInDeck(false);
  }, [open]);

  const runProbe = async (dir: string, silentError = false) => {
    const trimmed = dir.trim();
    if (!trimmed) return;
    setProbing(true);
    try {
      const probe = await api.inspectLocalDirectory(trimmed);
      setLastProbedDir(probe.local_dir || trimmed);
      setLocalDir(probe.local_dir || trimmed);
      if (probe.name) setName(probe.name);
      if (probe.slug) setSlug(probe.slug);
      if (probe.type) setType(probe.type);
      if (probe.category) setCategory(probe.category);
      if (probe.tech_stack) setTechStack(probe.tech_stack);
      if (probe.tags?.length > 0) setTagsText(probe.tags.join(", "));
      if (probe.git_remote) setGitRemote(probe.git_remote);
      if (probe.bat_file) setBatFile(probe.bat_file);
      if (probe.exe_name) setExeName(probe.exe_name);
      if (probe.prod_url) setProdUrl(probe.prod_url);
      if (probe.summary) setSummary(probe.summary);

      setExistingInDeck(Boolean(probe.existing_id));
      const parts = [
        probe.tech_stack,
        probe.type === "desktop" ? probe.exe_name : probe.bat_file,
        probe.version,
      ].filter(Boolean);
      setProbeSummaryNote(
        parts.length > 0
          ? `Đã tự nhận diện: ${parts.join(" • ")}`
          : "Đã đọc thư mục dự án."
      );
    } catch (err) {
      if (!silentError) {
        onNotify("error", formatError(err));
      }
    } finally {
      setProbing(false);
    }
  };

  const handleBrowse = async () => {
    try {
      const picked = await api.selectDirectory("Chọn thư mục dự án");
      if (picked) {
        setLocalDir(picked);
        await runProbe(picked);
      }
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  const handleDirBlur = () => {
    const trimmed = localDir.trim();
    if (trimmed && trimmed !== lastProbedDir) {
      void runProbe(trimmed, true);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const tags = tagsText
        .split(",")
        .map((t) => t.trim().replace(/^#/, ""))
        .filter(Boolean);

      const created = await api.upsertProject({
        id: "",
        slug: slug.trim(),
        name,
        type,
        category,
        summary,
        tags,
        tech_stack: techStack,
        git_remote: gitRemote,
        bat_file: batFile.trim() || "start.bat",
        exe_name: exeName.trim(),
        local_dir: localDir,
        prod_url: prodUrl.trim(),
      });
      if (category.trim()) onCategoryCreated();
      onCreated(created);
      onNotify("success", `Đã thêm dự án "${created.name}".`);
      onClose();
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ModalDialog
      open={open}
      onClose={onClose}
      size="lg"
      title={
        <h2 className="font-heading text-base font-semibold text-primary">
          Thêm Project mới
        </h2>
      }
      subtitle="Chọn thư mục trên máy để tự nhận diện stack, Git remote, mô tả và entry chạy"
    >
      <form onSubmit={handleSubmit} className="p-6 space-y-4">
        <FormField
          label="Thư mục cục bộ"
          hint={probing ? "Đang quét thông tin..." : "Tự điền khi chọn thư mục"}
        >
          <div className="flex gap-2">
            <input
              type="text"
              value={localDir}
              onChange={(e) => setLocalDir(e.target.value)}
              onBlur={handleDirBlur}
              placeholder="H:\app\my-project"
              className={`${UI_MONO_INPUT_CLS} flex-1 min-w-0`}
            />
            <button
              type="button"
              onClick={handleBrowse}
              disabled={probing}
              className={UI_BTN_SECONDARY}
            >
              <FolderOpen className="w-4 h-4 text-muted" />
              <span>Browse</span>
            </button>
            <button
              type="button"
              onClick={() => void runProbe(localDir)}
              disabled={probing || !localDir.trim()}
              title="Quét lại metadata từ thư mục"
              className={UI_BTN_SECONDARY}
            >
              {probing ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Sparkles className="w-4 h-4 text-emerald-400" />
              )}
              <span>Nhận diện</span>
            </button>
          </div>
          {(probeSummaryNote || existingInDeck) && (
            <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs">
              {probeSummaryNote && (
                <span className="text-emerald-400/90">{probeSummaryNote}</span>
              )}
              {existingInDeck && (
                <span className="px-2 py-0.5 rounded bg-amber-500/15 border border-amber-500/30 text-amber-300">
                  Thư mục này đã có .tdeck-meta.json trong T-Deck
                </span>
              )}
            </div>
          )}
        </FormField>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <FormField label="Tên dự án *">
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="VD: Invoice Portal"
              className={UI_INPUT_CLS}
            />
          </FormField>

          <FormField label="Môi trường">
            <select
              value={type}
              onChange={(e) => setType(e.target.value as ProjectType)}
              className={UI_SELECT_CLS}
            >
              <option value="web">Web App (.bat)</option>
              <option value="desktop">Desktop App (.exe)</option>
              <option value="hybrid">Hybrid (Web + Desktop)</option>
            </select>
          </FormField>

          <FormField label="Phân loại (Nhóm)">
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className={UI_SELECT_CLS}
            >
              <option value="">Chưa phân nhóm</option>
              {categories.map((c) => (
                <option key={c.id} value={c.name}>
                  {c.name}
                </option>
              ))}
            </select>
          </FormField>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <FormField label="Tech Stack">
            <input
              type="text"
              value={techStack}
              onChange={(e) => setTechStack(e.target.value)}
              placeholder="Go, Wails, React"
              className={UI_INPUT_CLS}
            />
          </FormField>
          <FormField label="Tags (phân cách bằng dấu phẩy)">
            <input
              type="text"
              value={tagsText}
              onChange={(e) => setTagsText(e.target.value)}
              placeholder="tool, video, ai"
              className={UI_INPUT_CLS}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-12 gap-4">
          <div className="sm:col-span-7 min-w-0">
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
          <div className="sm:col-span-5 min-w-0">
            {type === "web" ? (
              <FormField label="Script khởi chạy (.bat)">
                <input
                  type="text"
                  value={batFile}
                  onChange={(e) => setBatFile(e.target.value)}
                  placeholder="start.bat"
                  className={UI_MONO_INPUT_CLS}
                />
              </FormField>
            ) : type === "desktop" ? (
              <FormField label="File thực thi (.exe)">
                <input
                  type="text"
                  value={exeName}
                  onChange={(e) => setExeName(e.target.value)}
                  placeholder="MyApp.exe"
                  className={UI_MONO_INPUT_CLS}
                />
              </FormField>
            ) : (
              <FormField label="Entry (.bat / .exe)">
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

        {(type === "web" || type === "hybrid") && (
          <FormField label="Production Web URL (nếu đã deploy)">
            <input
              type="url"
              value={prodUrl}
              onChange={(e) => setProdUrl(e.target.value)}
              placeholder="https://app.example.com"
              className={UI_MONO_INPUT_CLS}
            />
          </FormField>
        )}

        <FormField label="Mô tả chức năng">
          <textarea
            rows={3}
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            placeholder="Để trống nếu muốn tự động lấy từ README.md"
            className={UI_TEXTAREA_CLS}
          />
        </FormField>

        <div className="pt-4 border-t border-edge flex justify-end gap-2.5">
          <button type="button" onClick={onClose} className={UI_BTN_SECONDARY}>
            Hủy
          </button>
          <button type="submit" disabled={saving} className={UI_BTN_PRIMARY}>
            {saving ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Plus className="w-4 h-4" />
            )}
            <span>Tạo Project</span>
          </button>
        </div>
      </form>
    </ModalDialog>
  );
}

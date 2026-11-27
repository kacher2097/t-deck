import type React from "react";
import { useEffect, useState } from "react";
import { api, formatError } from "../../lib/api";
import type {
  BatTemplate,
  Category,
  ProjectType,
  ProjectView,
} from "../../types/project";
import { ThumbnailUploader } from "../ui/ThumbnailUploader";
import { InspectorLocalToolsCard } from "./InspectorLocalToolsCard";
import { InspectorMetadataForm } from "./InspectorMetadataForm";

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
  const [summary, setSummary] = useState(project.summary);
  const [techStack, setTechStack] = useState(project.tech_stack);
  const [tags, setTags] = useState<string[]>(project.tags || []);
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
    setSummary(project.summary);
    setTechStack(project.tech_stack);
    setTags(project.tags || []);
    setGitRemote(project.git_remote);
    setLocalDir(project.local_dir);
    setBatFile(project.bat_file || "start.bat");
    setExeName(project.exe_name);
  }, [project]);

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
      if (probe.summary && !summary.trim()) {
        setSummary(probe.summary);
        updatedFields.push("Mô tả");
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

  const handleBrowseDir = async () => {
    const picked = await api.selectDirectory("Chọn thư mục dự án");
    if (picked) {
      setLocalDir(picked);
      await handleProbeDir(picked, true);
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

      <div className="lg:col-span-7 min-w-0">
        <InspectorMetadataForm
          name={name}
          slug={slug}
          type={type}
          category={category}
          categories={categories}
          summary={summary}
          techStack={techStack}
          tags={tags}
          gitRemote={gitRemote}
          localDir={localDir}
          batFile={batFile}
          exeName={exeName}
          saving={saving}
          probingDir={probingDir}
          onNameChange={setName}
          onSlugChange={setSlug}
          onTypeChange={setType}
          onCategoryChange={setCategory}
          onSummaryChange={setSummary}
          onTechStackChange={setTechStack}
          onTagsChange={setTags}
          onGitRemoteChange={setGitRemote}
          onLocalDirChange={setLocalDir}
          onBatFileChange={setBatFile}
          onExeNameChange={setExeName}
          onCreateCategory={onCreateCategory}
          onProbeDir={handleProbeDir}
          onBrowseDir={handleBrowseDir}
        />
      </div>
    </form>
  );
}

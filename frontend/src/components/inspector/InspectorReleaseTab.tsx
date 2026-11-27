import { Globe, Laptop } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useState } from "react";
import { useReleaseBuildJob } from "../../hooks/useReleaseBuildJob";
import { api, formatError } from "../../lib/api";
import type {
  ProjectTarget,
  ProjectView,
  Release,
  WebHealthResult,
} from "../../types/project";
import { BuildLogDrawer } from "./BuildLogDrawer";
import { DesktopStatusCard } from "./DesktopStatusCard";
import { PublishReleaseForm } from "./PublishReleaseForm";
import { ReleaseHistoryList } from "./ReleaseHistoryList";
import { TargetListEditor } from "./TargetListEditor";
import { WebLiveStatusCard } from "./WebLiveStatusCard";

interface InspectorReleaseTabProps {
  project: ProjectView;
  onUpdated: (updated: ProjectView) => void;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function InspectorReleaseTab({
  project,
  onUpdated,
  onNotify,
}: InspectorReleaseTabProps) {
  const targets = project.targets || [];
  const [selectedTargetId, setSelectedTargetId] = useState<string>(
    targets[0]?.id || ""
  );
  const [releases, setReleases] = useState<Release[]>([]);
  const [loadingRels, setLoadingRels] = useState(false);

  const [sourcePath, setSourcePath] = useState(project.local_dir || "");
  const [version, setVersion] = useState(project.latest_version || "v1.0.0");
  const [channel, setChannel] = useState("stable");
  const [assetKind, setAssetKind] = useState("portable_zip");
  const [deployUrl, setDeployUrl] = useState("");
  const [changelog, setChangelog] = useState(project.release_notes || "");
  const [publishing, setPublishing] = useState(false);
  const [updatingDesktop, setUpdatingDesktop] = useState(false);

  const [health, setHealth] = useState<WebHealthResult | null>(null);
  const [checkingHealth, setCheckingHealth] = useState(false);
  const [copiedKey, setCopiedKey] = useState("");
  const [suggestingNotes, setSuggestingNotes] = useState(false);

  const activeTarget: ProjectTarget | undefined =
    targets.find((t) => t.id === selectedTargetId) || targets[0];

  useEffect(() => {
    const first = project.targets?.[0];
    setSelectedTargetId(first?.id || "");
    setHealth(null);
  }, [project.id]);

  useEffect(() => {
    setSourcePath(project.local_dir || "");
    setVersion(project.latest_version || "v1.0.0");
    setChangelog(project.release_notes || "");
    if (activeTarget) {
      setDeployUrl(activeTarget.prod_url || "");
      if (activeTarget.kind === "web") {
        setAssetKind(
          activeTarget.distribution === "static_zip"
            ? "static_zip"
            : "external_url"
        );
      } else {
        setAssetKind(
          activeTarget.distribution === "installer"
            ? "installer_exe"
            : "portable_zip"
        );
      }
    }
  }, [project, activeTarget?.id]);

  const loadHistory = useCallback(async () => {
    setLoadingRels(true);
    try {
      const list = await api.listReleases(project.id, activeTarget?.id || "");
      setReleases(list || []);
    } catch {
      setReleases([]);
    } finally {
      setLoadingRels(false);
    }
  }, [project.id, activeTarget?.id]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  const buildJob = useReleaseBuildJob({
    projectId: project.id,
    projectName: project.name,
    onUpdated,
    onReloadHistory: loadHistory,
    onNotify,
  });

  const copyText = async (key: string, text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(""), 1500);
      onNotify("success", `Đã sao chép ${label}.`);
    } catch {
      onNotify("error", "Không thể truy cập clipboard.");
    }
  };

  const handleSuggestChangelog = async () => {
    setSuggestingNotes(true);
    try {
      const notes = await api.suggestReleaseChangelog(
        project.id,
        activeTarget?.id || ""
      );
      setChangelog(notes);
      onNotify("success", "Đã gợi ý Changelog từ lịch sử Git.");
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setSuggestingNotes(false);
    }
  };

  const handleBuildAndPublish = async () => {
    if (!activeTarget) return;
    await buildJob.startBuildAndPublish({
      targetId: activeTarget.id,
      version,
      channel,
      assetKind,
      changelog,
    });
  };

  const handlePublish = async (e: React.FormEvent) => {
    e.preventDefault();
    setPublishing(true);
    try {
      const saved = await api.publishRelease({
        project_id: project.id,
        target_id: activeTarget?.id || "",
        source_path: sourcePath,
        version,
        channel,
        asset_kind: assetKind,
        deploy_url: deployUrl,
        changelog,
      });
      const fresh = await api.getProjectDetail(project.id);
      onUpdated(fresh);
      await loadHistory();
      onNotify(
        "success",
        `Đã phát hành ${project.name} ${saved.version} (${saved.channel}).`
      );
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setPublishing(false);
    }
  };

  const handleCheckHealth = async () => {
    if (!activeTarget) return;
    setCheckingHealth(true);
    try {
      const res = await api.checkWebHealth(activeTarget.id);
      setHealth(res);
      if (res.online) {
        onNotify(
          res.drift ? "warn" : "success",
          res.drift
            ? `Web Online (${res.latency_ms}ms) nhưng lệch version live ${res.live_version}!`
            : `Web Online (${res.status_code} · ${res.latency_ms}ms).`
        );
      } else {
        onNotify("error", res.error || `HTTP ${res.status_code}`);
      }
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setCheckingHealth(false);
    }
  };

  const handleUpdateDesktop = async (releaseId = "") => {
    setUpdatingDesktop(true);
    try {
      const fresh = await api.updateDesktopRelease(
        project.id,
        activeTarget?.id || "",
        releaseId
      );
      onUpdated(fresh);
      onNotify(
        "success",
        `Đã cập nhật "${fresh.name}" lên ${fresh.installed_version || fresh.latest_version}.`
      );
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setUpdatingDesktop(false);
    }
  };

  const handleRollbackDesktop = async () => {
    setUpdatingDesktop(true);
    try {
      const fresh = await api.rollbackDesktopRelease(
        project.id,
        activeTarget?.id || ""
      );
      onUpdated(fresh);
      onNotify("success", `Đã rollback "${fresh.name}" về bản trước đó.`);
    } catch (err) {
      onNotify("error", formatError(err));
    } finally {
      setUpdatingDesktop(false);
    }
  };

  const handleCopyDownloadLink = async (releaseId: string) => {
    try {
      const url = await api.generateReleaseDownloadURL(releaseId, 24);
      await copyText(`link-${releaseId}`, url, "link tải bản phát hành");
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  const handleDeleteRelease = async (rel: Release) => {
    try {
      await api.deleteRelease(rel.id, Boolean(rel.r2_key));
      await loadHistory();
      const fresh = await api.getProjectDetail(project.id);
      onUpdated(fresh);
      onNotify("success", `Đã xóa release ${rel.version}.`);
    } catch (err) {
      onNotify("error", formatError(err));
    }
  };

  const isWeb =
    activeTarget?.kind === "web" || (!activeTarget && project.type === "web");

  return (
    <div className="space-y-5">
      {targets.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap border-b border-edge pb-3">
          {targets.map((t) => {
            const active = activeTarget?.id === t.id;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => {
                  setSelectedTargetId(t.id);
                  setHealth(null);
                }}
                className={`h-8 px-3 rounded-md border text-xs font-medium inline-flex items-center gap-1.5 transition-colors ${
                  active
                    ? "bg-elevated border-muted text-primary"
                    : "bg-canvas border-edge text-muted hover:text-primary"
                }`}
              >
                {t.kind === "web" ? (
                  <Globe className="w-3.5 h-3.5 text-status-ready" />
                ) : (
                  <Laptop className="w-3.5 h-3.5" />
                )}
                <span>{t.name}</span>
              </button>
            );
          })}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        <div className="lg:col-span-5 space-y-4 min-w-0">
          <TargetListEditor
            project={project}
            batFile={project.bat_file || "start.bat"}
            exeName={project.exe_name}
            onUpdated={onUpdated}
            onNotify={onNotify}
          />

          {isWeb ? (
            <WebLiveStatusCard
              activeTarget={activeTarget}
              health={health}
              checkingHealth={checkingHealth}
              copiedKey={copiedKey}
              onCopyText={copyText}
              onCheckHealth={handleCheckHealth}
              onNotify={onNotify}
            />
          ) : (
            <DesktopStatusCard
              project={project}
              latestRelease={releases[0]}
              updatingDesktop={updatingDesktop}
              onUpdateDesktop={handleUpdateDesktop}
              onRollbackDesktop={handleRollbackDesktop}
              onCopyDownloadLink={handleCopyDownloadLink}
            />
          )}

          <ReleaseHistoryList
            releases={releases}
            loading={loadingRels}
            isWeb={isWeb}
            copiedKey={copiedKey}
            onInstallRelease={handleUpdateDesktop}
            onCopyDownloadLink={handleCopyDownloadLink}
            onDeleteRelease={handleDeleteRelease}
          />
        </div>

        <div className="lg:col-span-7 min-w-0">
          <PublishReleaseForm
            project={project}
            activeTarget={activeTarget}
            isWeb={isWeb}
            version={version}
            channel={channel}
            assetKind={assetKind}
            sourcePath={sourcePath}
            deployUrl={deployUrl}
            changelog={changelog}
            publishing={publishing}
            suggestingNotes={suggestingNotes}
            buildRunning={buildJob.jobStatus === "running"}
            onVersionChange={setVersion}
            onChannelChange={setChannel}
            onAssetKindChange={setAssetKind}
            onSourcePathChange={setSourcePath}
            onDeployUrlChange={setDeployUrl}
            onChangelogChange={setChangelog}
            onSuggestChangelog={handleSuggestChangelog}
            onPublish={handlePublish}
            onBuildAndPublish={handleBuildAndPublish}
          />

          <BuildLogDrawer
            open={buildJob.drawerOpen || buildJob.jobStatus === "running"}
            jobId={buildJob.activeJobId}
            status={buildJob.jobStatus}
            errorMsg={buildJob.jobError}
            progress={buildJob.progress}
            logs={buildJob.buildLogs}
            onCancel={buildJob.cancelBuildJob}
            onClose={buildJob.closeDrawer}
          />
        </div>
      </div>
    </div>
  );
}

import { useCallback, useEffect, useState } from "react";
import { api, formatError } from "../lib/api";
import {
  type BuildLogEvent,
  type BuildStatusEvent,
  type ReleaseProgressEvent,
  subscribeEvent,
} from "../lib/events";
import type { ProjectView } from "../types/project";

interface UseReleaseBuildJobOptions {
  projectId: string;
  projectName: string;
  onUpdated: (updated: ProjectView) => void;
  onReloadHistory: () => Promise<void>;
  onNotify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function useReleaseBuildJob({
  projectId,
  projectName,
  onUpdated,
  onReloadHistory,
  onNotify,
}: UseReleaseBuildJobOptions) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [activeJobId, setActiveJobId] = useState("");
  const [jobStatus, setJobStatus] = useState<BuildStatusEvent["status"] | null>(
    null
  );
  const [jobError, setJobError] = useState("");
  const [progress, setProgress] = useState<ReleaseProgressEvent | null>(null);
  const [buildLogs, setBuildLogs] = useState<BuildLogEvent[]>([]);

  useEffect(() => {
    const unsubProg = subscribeEvent<ReleaseProgressEvent>(
      "release:progress",
      (ev) => {
        if (ev.project_id === projectId) setProgress(ev);
      }
    );
    const unsubLog = subscribeEvent<BuildLogEvent>("build:log", (ev) => {
      if (ev.project_id === projectId) {
        setBuildLogs((prev) => [...prev.slice(-200), ev]);
      }
    });
    const unsubStatus = subscribeEvent<BuildStatusEvent>(
      "build:status",
      async (ev) => {
        if (ev.project_id !== projectId) return;
        setJobStatus(ev.status);
        setJobError(ev.error || "");
        if (ev.status === "success") {
          const fresh = await api.getProjectDetail(projectId);
          onUpdated(fresh);
          await onReloadHistory();
          onNotify("success", `Build & Publish "${projectName}" hoàn tất.`);
        } else if (ev.status === "failed") {
          onNotify("error", ev.error || "Build & Publish thất bại.");
        }
      }
    );
    return () => {
      unsubProg();
      unsubLog();
      unsubStatus();
    };
  }, [projectId, projectName, onUpdated, onReloadHistory, onNotify]);

  const startBuildAndPublish = useCallback(
    async (params: {
      targetId: string;
      version: string;
      channel: string;
      assetKind: string;
      changelog: string;
    }) => {
      setDrawerOpen(true);
      setBuildLogs([]);
      setProgress(null);
      setJobError("");
      setJobStatus("running");
      try {
        const jid = await api.buildAndPublishTarget({
          project_id: projectId,
          target_id: params.targetId,
          version: params.version,
          channel: params.channel,
          asset_kind: params.assetKind,
          changelog: params.changelog,
        });
        setActiveJobId(jid);
      } catch (err) {
        setJobStatus("failed");
        setJobError(formatError(err));
        onNotify("error", formatError(err));
      }
    },
    [onNotify, projectId]
  );

  const cancelBuildJob = useCallback(() => {
    if (activeJobId) {
      api.cancelActiveJob(activeJobId).catch(() => {});
    }
  }, [activeJobId]);

  const closeDrawer = useCallback(() => {
    setDrawerOpen(false);
  }, []);

  return {
    drawerOpen,
    activeJobId,
    jobStatus,
    jobError,
    progress,
    buildLogs,
    startBuildAndPublish,
    cancelBuildJob,
    closeDrawer,
  };
}


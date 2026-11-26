import { useCallback, useEffect, useMemo, useState } from "react";
import { api, formatError } from "../lib/api";
import { type ProcessChangedEvent, subscribeEvent } from "../lib/events";
import type {
  Category,
  ProjectView,
  RunningProcess,
  WebHealthResult,
} from "../types/project";

interface UseProjectsHubOptions {
  debouncedQuery: string;
  selectedCategory: string;
  onResetCategory: () => void;
  onSelectProject: (project: ProjectView | null) => void;
  onRefreshSelectedThumbnail: (projectId: string) => void;
  notify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function useProjectsHub({
  debouncedQuery,
  selectedCategory,
  onResetCategory,
  onSelectProject,
  onRefreshSelectedThumbnail,
  notify,
}: UseProjectsHubOptions) {
  const [queryMatchedProjects, setQueryMatchedProjects] = useState<
    ProjectView[]
  >([]);
  const [categories, setCategories] = useState<Category[]>([]);

  const [scanning, setScanning] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [checkingHealth, setCheckingHealth] = useState(false);
  const [deletingProjects, setDeletingProjects] = useState(false);
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null);
  const [busyCardId, setBusyCardId] = useState<string | null>(null);
  const [healthMap, setHealthMap] = useState<Record<string, WebHealthResult>>(
    {}
  );
  const [runningMap, setRunningMap] = useState<Record<string, RunningProcess>>(
    {}
  );

  const loadCategories = useCallback(async () => {
    try {
      const list = await api.listCategories();
      setCategories(list || []);
    } catch (err) {
      notify("error", formatError(err));
    }
  }, [notify]);

  const loadQueryMatched = useCallback(async () => {
    try {
      const list = await api.listProjects({
        type: "",
        category: "",
        status: "",
        tag: "",
        query: debouncedQuery,
      });
      setQueryMatchedProjects(list || []);
    } catch (err) {
      notify("error", formatError(err));
    }
  }, [debouncedQuery, notify]);

  const refreshAll = useCallback(async () => {
    await Promise.all([loadCategories(), loadQueryMatched()]);
  }, [loadCategories, loadQueryMatched]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    loadQueryMatched();
  }, [loadQueryMatched]);

  // Global unhandledrejection guard so async Wails/DOM errors never vanish silently.
  useEffect(() => {
    const onUnhandled = (ev: PromiseRejectionEvent) => {
      ev.preventDefault();
      notify("error", formatError(ev.reason));
    };
    window.addEventListener("unhandledrejection", onUnhandled);
    return () => window.removeEventListener("unhandledrejection", onUnhandled);
  }, [notify]);

  // Subscribe to backend runtime events (web health & process monitor).
  useEffect(() => {
    api
      .listRunningProcesses()
      .then((list) => {
        const next: Record<string, RunningProcess> = {};
        for (const p of list || []) {
          if (p.pid > 0) next[p.project_id] = p;
        }
        setRunningMap(next);
      })
      .catch(() => {});

    const unsubHealth = subscribeEvent<WebHealthResult>(
      "health:result",
      (res) => {
        setHealthMap((prev) => ({ ...prev, [res.target_id]: res }));
      }
    );
    const unsubProc = subscribeEvent<ProcessChangedEvent>(
      "process:changed",
      (ev) => {
        setRunningMap((prev) => {
          const next = { ...prev };
          if (!ev.running || ev.pid <= 0) {
            delete next[ev.project_id];
          } else {
            next[ev.project_id] = {
              project_id: ev.project_id,
              pid: ev.pid,
              ports: ev.ports || [],
              started_at: prev[ev.project_id]?.started_at || Date.now(),
              command: prev[ev.project_id]?.command || "",
            };
          }
          return next;
        });
      }
    );
    return () => {
      unsubHealth();
      unsubProc();
    };
  }, []);

  const runningIds = useMemo(
    () =>
      new Set(
        Object.values(runningMap)
          .filter((r) => r.pid > 0)
          .map((r) => r.project_id)
      ),
    [runningMap]
  );

  const handleScan = useCallback(async () => {
    setScanning(true);
    try {
      const rep = await api.scanLocalWorkspaces();
      await refreshAll();
      notify(
        "success",
        `Quét xong ${rep.scanned_dirs} thư mục: +${rep.added} mới, ${rep.relinked} relink, ${rep.updated} cập nhật.`
      );
      if (rep.errors?.length > 0) {
        notify("warn", rep.errors[0]);
      }
    } catch (err) {
      notify("error", formatError(err));
    } finally {
      setScanning(false);
    }
  }, [notify, refreshAll]);

  const handleSync = useCallback(async () => {
    setSyncing(true);
    try {
      const rep = await api.syncCatalogWithR2();
      setLastSyncAt(rep.synced_at || Math.floor(Date.now() / 1000));
      await refreshAll();
      notify("success", rep.message);
    } catch (err) {
      notify("error", formatError(err));
    } finally {
      setSyncing(false);
    }
  }, [notify, refreshAll]);

  const handleLaunch = useCallback(
    async (project: ProjectView) => {
      setBusyCardId(project.id);
      try {
        const res = await api.launchProject(project.id);
        if (res.status === "ENV_CREATED_NEEDS_CONFIG") {
          notify("warn", res.message);
          await refreshAll();
        } else if (
          res.status === "MISSING_BAT" ||
          res.status === "MISSING_EXE"
        ) {
          notify("warn", res.message);
          onSelectProject(project);
          await refreshAll();
        } else {
          notify("success", res.message);
          // Skip unnecessary grid re-fetch on normal LAUNCHED when card is already ready
          // so the list stays rock-steady with zero layout shift or thumbnail flash.
          if (project.status !== "ready") {
            await refreshAll();
          }
        }
      } catch (err) {
        notify("error", formatError(err));
      } finally {
        setBusyCardId(null);
      }
    },
    [notify, onSelectProject, refreshAll]
  );

  const handleHydrate = useCallback(
    async (project: ProjectView) => {
      setBusyCardId(project.id);
      try {
        const updated = await api.hydrateProjectOnNewPC(project.id);
        await refreshAll();
        notify(
          updated.status === "missing_env" ? "warn" : "success",
          updated.status === "missing_env"
            ? `Đã clone "${updated.name}". Cần cấu hình .env trước khi chạy.`
            : `Đã cài đặt "${updated.name}" sẵn sàng khởi chạy.`
        );
      } catch (err) {
        notify("error", formatError(err));
      } finally {
        setBusyCardId(null);
      }
    },
    [notify, refreshAll]
  );

  const handleUploadThumbnail = useCallback(
    async (project: ProjectView, pngDataUrl: string) => {
      try {
        await api.saveClipboardThumbnail(project.id, pngDataUrl);
        await refreshAll();
        onRefreshSelectedThumbnail(project.id);
        notify("success", `Đã lưu ảnh preview cho "${project.name}".`);
      } catch (err) {
        notify("error", formatError(err));
      }
    },
    [notify, onRefreshSelectedThumbnail, refreshAll]
  );

  const handleOpenLive = useCallback(
    (p: ProjectView) => {
      const webTarget = (p.targets || []).find(
        (t) => t.kind === "web" && t.prod_url
      );
      if (!webTarget) return;
      api
        .openTargetURL(webTarget.id, "prod")
        .catch((e) => notify("error", formatError(e)));
    },
    [notify]
  );

  const handleCheckAllHealth = useCallback(async () => {
    setCheckingHealth(true);
    try {
      const results = await api.checkAllWebHealth();
      const map: Record<string, WebHealthResult> = {};
      let online = 0;
      for (const r of results || []) {
        map[r.target_id] = r;
        if (r.online) online++;
      }
      setHealthMap((prev) => ({ ...prev, ...map }));
      notify(
        "success",
        `Kiểm tra xong ${results?.length || 0} Web targets (${online} online).`
      );
    } catch (err) {
      notify("error", formatError(err));
    } finally {
      setCheckingHealth(false);
    }
  }, [notify]);

  const handleStopProcess = useCallback(
    async (p: ProjectView) => {
      try {
        await api.stopProjectProcess(p.id);
        notify("success", `Đã dừng tiến trình "${p.name}".`);
      } catch (err) {
        notify("error", formatError(err));
      }
    },
    [notify]
  );

  const handleOpenLocalPort = useCallback(
    (port: number) => {
      api
        .openLocalhostPort(port)
        .catch((e) => notify("error", formatError(e)));
    },
    [notify]
  );

  const handleUpdateDesktopFromList = useCallback(
    async (p: ProjectView) => {
      setBusyCardId(p.id);
      try {
        const fresh = await api.updateDesktopRelease(p.id, "", "");
        await refreshAll();
        notify(
          "success",
          `Đã cập nhật "${fresh.name}" lên ${fresh.installed_version || fresh.latest_version}.`
        );
      } catch (err) {
        notify("error", formatError(err));
      } finally {
        setBusyCardId(null);
      }
    },
    [notify, refreshAll]
  );

  const handleCopyReleaseLink = useCallback(
    async (releaseId: string) => {
      try {
        const url = await api.generateReleaseDownloadURL(releaseId, 24);
        await navigator.clipboard.writeText(url);
        notify("success", "Đã sao chép link tải release.");
      } catch (err) {
        notify("error", formatError(err));
      }
    },
    [notify]
  );

  const handleOpenExplorer = useCallback(
    (p: ProjectView) => {
      api.openInExplorer(p.id).catch((e) => notify("error", formatError(e)));
    },
    [notify]
  );

  const handleOpenVSCode = useCallback(
    (p: ProjectView) => {
      api
        .openInEditor(p.id, "code")
        .catch((e) => notify("error", formatError(e)));
    },
    [notify]
  );

  const handleCreateCategory = useCallback(
    async (name: string): Promise<Category | null> => {
      try {
        const created = await api.createCategory(name);
        await loadCategories();
        notify("success", `Đã tạo phân loại "${created.name}".`);
        return created;
      } catch (err) {
        notify("error", formatError(err));
        return null;
      }
    },
    [loadCategories, notify]
  );

  const handleDeleteCategory = useCallback(
    async (cat: Category) => {
      try {
        await api.deleteCategory(cat.id);
        if (selectedCategory.toLowerCase() === cat.name.toLowerCase()) {
          onResetCategory();
        }
        await refreshAll();
        notify("success", `Đã xóa phân loại "${cat.name}".`);
      } catch (err) {
        notify("error", formatError(err));
      }
    },
    [notify, onResetCategory, refreshAll, selectedCategory]
  );

  const handleDeleteProjects = useCallback(
    async (targets: ProjectView[], deleteLocalFiles: boolean): Promise<boolean> => {
      if (targets.length === 0) return false;
      setDeletingProjects(true);
      try {
        if (targets.length === 1) {
          await api.deleteProject(targets[0].id, deleteLocalFiles);
          await refreshAll();
          notify(
            "success",
            deleteLocalFiles
              ? `Đã xóa vĩnh viễn "${targets[0].name}" và toàn bộ thư mục cục bộ.`
              : `Đã gỡ "${targets[0].name}" khỏi danh sách T-Deck.`
          );
        } else {
          const ids = targets.map((p) => p.id);
          const res = await api.deleteProjects(ids, deleteLocalFiles);
          await refreshAll();
          if (res.failed && res.failed.length > 0) {
            notify(
              "warn",
              `Đã xóa ${res.deleted}/${targets.length} dự án (${res.failed.length} lỗi: ${res.failed[0]}).`
            );
          } else {
            notify(
              "success",
              deleteLocalFiles
                ? `Đã xóa vĩnh viễn ${res.deleted} dự án và toàn bộ thư mục cục bộ.`
                : `Đã gỡ ${res.deleted} dự án khỏi danh sách T-Deck.`
            );
          }
        }
        return true;
      } catch (err) {
        await refreshAll();
        notify("error", formatError(err));
        return false;
      } finally {
        setDeletingProjects(false);
      }
    },
    [notify, refreshAll]
  );

  return {
    queryMatchedProjects,
    categories,
    scanning,
    syncing,
    checkingHealth,
    deletingProjects,
    lastSyncAt,
    busyCardId,
    healthMap,
    runningMap,
    runningIds,
    loadCategories,
    refreshAll,
    handleScan,
    handleSync,
    handleLaunch,
    handleHydrate,
    handleUploadThumbnail,
    handleOpenLive,
    handleCheckAllHealth,
    handleStopProcess,
    handleOpenLocalPort,
    handleUpdateDesktopFromList,
    handleCopyReleaseLink,
    handleOpenExplorer,
    handleOpenVSCode,
    handleCreateCategory,
    handleDeleteCategory,
    handleDeleteProjects,
  };
}

import { useCallback, useEffect, useMemo, useState } from "react";
import { api, formatError } from "../lib/api";
import { type ProcessChangedEvent, subscribeEvent } from "../lib/events";
import type {
  ProjectView,
  RunningProcess,
  WebHealthResult,
} from "../types/project";

interface UseRuntimeEventsOptions {
  notify: (kind: "success" | "error" | "warn", msg: string) => void;
}

export function useRuntimeEvents({ notify }: UseRuntimeEventsOptions) {
  const [checkingHealth, setCheckingHealth] = useState(false);
  const [healthMap, setHealthMap] = useState<Record<string, WebHealthResult>>(
    {}
  );
  const [runningMap, setRunningMap] = useState<Record<string, RunningProcess>>(
    {}
  );

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

  return {
    checkingHealth,
    healthMap,
    runningMap,
    runningIds,
    handleCheckAllHealth,
    handleStopProcess,
    handleOpenLocalPort,
  };
}


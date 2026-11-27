import { useCallback, useEffect, useState } from "react";
import type { ProjectView } from "../types/project";

interface UseBulkSelectionOptions {
  allProjects: ProjectView[];
  visibleProjects: ProjectView[];
  onDeleteProjects: (
    targets: ProjectView[],
    deleteLocalFiles: boolean
  ) => Promise<boolean>;
  onAfterDelete?: (removedIds: Set<string>) => void;
}

export function useBulkSelection({
  allProjects,
  visibleProjects,
  onDeleteProjects,
  onAfterDelete,
}: UseBulkSelectionOptions) {
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [pendingDelete, setPendingDelete] = useState<ProjectView[]>([]);

  // Prune any selected IDs that no longer exist in the catalog.
  useEffect(() => {
    setSelectedIds((prev) => {
      if (prev.size === 0) return prev;
      const valid = new Set(allProjects.map((p) => p.id));
      let changed = false;
      const next = new Set<string>();
      for (const id of prev) {
        if (valid.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [allProjects]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && selectionMode && pendingDelete.length === 0) {
        setSelectionMode(false);
        setSelectedIds(new Set());
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [selectionMode, pendingDelete.length]);

  const handleToggleSelect = useCallback((project: ProjectView) => {
    setSelectionMode(true);
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(project.id)) {
        next.delete(project.id);
      } else {
        next.add(project.id);
      }
      return next;
    });
  }, []);

  const handleToggleSelectAllVisible = useCallback(() => {
    setSelectedIds((prev) => {
      const allSelected =
        visibleProjects.length > 0 &&
        visibleProjects.every((p) => prev.has(p.id));
      const next = new Set(prev);
      if (allSelected) {
        for (const p of visibleProjects) next.delete(p.id);
      } else {
        for (const p of visibleProjects) next.add(p.id);
      }
      return next;
    });
  }, [visibleProjects]);

  const handleExitSelection = useCallback(() => {
    setSelectionMode(false);
    setSelectedIds(new Set());
  }, []);

  const handleRequestSingleDelete = useCallback((project: ProjectView) => {
    setPendingDelete([project]);
  }, []);

  const handleRequestBulkDelete = useCallback(() => {
    const targets = allProjects.filter((p) => selectedIds.has(p.id));
    if (targets.length > 0) {
      setPendingDelete(targets);
    }
  }, [allProjects, selectedIds]);

  const handleConfirmDelete = useCallback(
    async (deleteLocalFiles: boolean) => {
      const targets = pendingDelete;
      const ok = await onDeleteProjects(targets, deleteLocalFiles);
      if (ok) {
        const removedIds = new Set(targets.map((t) => t.id));
        onAfterDelete?.(removedIds);
        setSelectedIds((prev) => {
          const next = new Set(prev);
          for (const id of removedIds) next.delete(id);
          return next;
        });
        setPendingDelete([]);
        if (selectionMode && targets.length >= selectedIds.size) {
          setSelectionMode(false);
        }
      }
    },
    [
      onAfterDelete,
      onDeleteProjects,
      pendingDelete,
      selectedIds.size,
      selectionMode,
    ]
  );

  const cancelPendingDelete = useCallback(() => {
    setPendingDelete([]);
  }, []);

  const allVisibleSelected =
    visibleProjects.length > 0 &&
    visibleProjects.every((p) => selectedIds.has(p.id));

  return {
    selectionMode,
    selectedIds,
    pendingDelete,
    allVisibleSelected,
    enterSelectionMode: () => setSelectionMode(true),
    handleToggleSelect,
    handleToggleSelectAllVisible,
    handleExitSelection,
    handleRequestSingleDelete,
    handleRequestBulkDelete,
    handleConfirmDelete,
    cancelPendingDelete,
  };
}


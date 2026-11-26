import {
  ArrowUpDown,
  CheckSquare,
  FolderSearch,
  Settings,
  Trash2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AddProjectModal } from "./components/AddProjectModal";
import { ConfirmDeleteModal } from "./components/ConfirmDeleteModal";
import { ProjectDetailModal } from "./components/inspector/ProjectDetailModal";
import { ProjectCard } from "./components/ProjectCard";
import { ReleaseMatrixTable } from "./components/ReleaseMatrixTable";
import { SettingsModal } from "./components/settings/SettingsModal";
import { SidebarNav } from "./components/SidebarNav";
import { ToastStack } from "./components/ToastStack";
import { TopCommandBar } from "./components/TopCommandBar";
import { UI_BTN_PRIMARY, UI_BTN_SECONDARY } from "./components/ui/uiTokens";
import { useProjectsHub } from "./hooks/useProjectsHub";
import { useToast } from "./hooks/useToast";
import { applyFacets, type ProjectSortBy } from "./lib/facets";
import type { ProjectView } from "./types/project";

export function App() {
  const [viewMode, setViewMode] = useState<"grid" | "matrix">(() => {
    try {
      return localStorage.getItem("tdeck_view_mode") === "matrix"
        ? "matrix"
        : "grid";
    } catch {
      return "grid";
    }
  });
  const [selectedType, setSelectedType] = useState("all");
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [selectedStatus, setSelectedStatus] = useState("all");
  const [selectedTag, setSelectedTag] = useState("");
  const [selectedQuickFilter, setSelectedQuickFilter] = useState<
    "all" | "has_release" | "needs_update" | "running"
  >("all");
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [isComposing, setIsComposing] = useState(false);
  const [sortBy, setSortBy] = useState<ProjectSortBy>(() => {
    try {
      const saved = localStorage.getItem("tdeck_sort_by") as ProjectSortBy | null;
      return saved || "name_asc";
    } catch {
      return "name_asc";
    }
  });

  const [selectedProject, setSelectedProject] = useState<ProjectView | null>(
    null
  );
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [pendingDelete, setPendingDelete] = useState<ProjectView[]>([]);

  const { toasts, notify, dismiss } = useToast();
  const searchInputRef = useRef<HTMLInputElement>(null);

  const handleResetCategory = useCallback(() => {
    setSelectedCategory("all");
  }, []);

  const handleRefreshSelectedThumbnail = useCallback((projectId: string) => {
    setSelectedProject((prev) =>
      prev?.id === projectId ? { ...prev } : prev
    );
  }, []);

  const hub = useProjectsHub({
    debouncedQuery,
    selectedCategory,
    onResetCategory: handleResetCategory,
    onSelectProject: setSelectedProject,
    onRefreshSelectedThumbnail: handleRefreshSelectedThumbnail,
    notify,
  });

  const hasActiveQuery = Boolean(debouncedQuery.trim());

  const filteredProjects = useMemo(
    () =>
      applyFacets(hub.queryMatchedProjects, {
        type: selectedType,
        category: selectedCategory,
        status: selectedStatus,
        tag: selectedTag,
        quickFilter: selectedQuickFilter,
        runningIds: hub.runningIds,
        preserveOrder: hasActiveQuery,
        hasQuery: hasActiveQuery,
        sortBy,
      }),
    [
      hub.queryMatchedProjects,
      selectedType,
      selectedCategory,
      selectedStatus,
      selectedTag,
      selectedQuickFilter,
      hub.runningIds,
      hasActiveQuery,
      sortBy,
    ]
  );

  // Prune any selected IDs that no longer exist in the catalog.
  useEffect(() => {
    setSelectedIds((prev) => {
      if (prev.size === 0) return prev;
      const valid = new Set(hub.queryMatchedProjects.map((p) => p.id));
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
  }, [hub.queryMatchedProjects]);

  const handleViewModeChange = useCallback((mode: "grid" | "matrix") => {
    setViewMode(mode);
    try {
      localStorage.setItem("tdeck_view_mode", mode);
    } catch {
      // ignore storage quota errors
    }
  }, []);

  const handleSortByChange = useCallback((nextSort: ProjectSortBy) => {
    setSortBy(nextSort);
    try {
      localStorage.setItem("tdeck_sort_by", nextSort);
    } catch {
      // ignore storage quota errors
    }
  }, []);

  const handleSearchCommit = useCallback(() => {
    const trimmed = query.trim();
    if (trimmed === "" || (trimmed.length < 2 && !trimmed.startsWith("#"))) {
      setDebouncedQuery("");
      return;
    }
    setDebouncedQuery(trimmed);
  }, [query]);

  useEffect(() => {
    if (isComposing) return;
    const trimmed = query.trim();
    if (trimmed === "" || (trimmed.length < 2 && !trimmed.startsWith("#"))) {
      setDebouncedQuery("");
      return;
    }
    const timer = setTimeout(() => {
      setDebouncedQuery(trimmed);
    }, 320);
    return () => clearTimeout(timer);
  }, [query, isComposing]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      } else if (e.key === "Escape" && selectionMode && pendingDelete.length === 0) {
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
      const allVisibleSelected =
        filteredProjects.length > 0 &&
        filteredProjects.every((p) => prev.has(p.id));
      const next = new Set(prev);
      if (allVisibleSelected) {
        for (const p of filteredProjects) next.delete(p.id);
      } else {
        for (const p of filteredProjects) next.add(p.id);
      }
      return next;
    });
  }, [filteredProjects]);

  const handleExitSelection = useCallback(() => {
    setSelectionMode(false);
    setSelectedIds(new Set());
  }, []);

  const handleRequestSingleDelete = useCallback((project: ProjectView) => {
    setPendingDelete([project]);
  }, []);

  const handleRequestBulkDelete = useCallback(() => {
    const targets = hub.queryMatchedProjects.filter((p) =>
      selectedIds.has(p.id)
    );
    if (targets.length > 0) {
      setPendingDelete(targets);
    }
  }, [hub.queryMatchedProjects, selectedIds]);

  const handleConfirmDelete = useCallback(
    async (deleteLocalFiles: boolean) => {
      const targets = pendingDelete;
      const ok = await hub.handleDeleteProjects(targets, deleteLocalFiles);
      if (ok) {
        const removedIds = new Set(targets.map((t) => t.id));
        setSelectedProject((prev) =>
          prev && removedIds.has(prev.id) ? null : prev
        );
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
    [hub.handleDeleteProjects, pendingDelete, selectedIds.size, selectionMode]
  );

  const handleUploadError = useCallback(
    (msg: string) => {
      notify("error", msg);
    },
    [notify]
  );

  const handleTagClick = useCallback((tag: string) => {
    setSelectedTag((prev) =>
      prev.toLowerCase() === tag.toLowerCase() ? "" : tag
    );
  }, []);

  const handleCategoryClick = useCallback((cat: string) => {
    setSelectedCategory((prev) =>
      prev.toLowerCase() === cat.toLowerCase() ? "all" : cat
    );
  }, []);

  const handleProjectUpdated = useCallback(
    (updated: ProjectView) => {
      setSelectedProject(updated);
      hub.refreshAll();
    },
    [hub.refreshAll]
  );

  const allVisibleSelected =
    filteredProjects.length > 0 &&
    filteredProjects.every((p) => selectedIds.has(p.id));

  return (
    <div className="app-enter h-screen w-screen bg-canvas text-primary flex flex-col overflow-hidden">
      <TopCommandBar
        query={query}
        onQueryChange={setQuery}
        onSearchCommit={handleSearchCommit}
        onCompositionChange={setIsComposing}
        searchInputRef={searchInputRef}
        viewMode={viewMode}
        onViewModeChange={handleViewModeChange}
        scanning={hub.scanning}
        syncing={hub.syncing}
        checkingHealth={hub.checkingHealth}
        lastSyncAt={hub.lastSyncAt}
        onScan={hub.handleScan}
        onSync={hub.handleSync}
        onCheckAllHealth={hub.handleCheckAllHealth}
        onAddProject={() => setAddOpen(true)}
        onOpenSettings={() => setSettingsOpen(true)}
      />

      <div className="flex-1 flex min-h-0 overflow-hidden">
        <SidebarNav
          queryMatchedProjects={hub.queryMatchedProjects}
          categories={hub.categories}
          selectedType={selectedType}
          selectedCategory={selectedCategory}
          selectedStatus={selectedStatus}
          selectedTag={selectedTag}
          selectedQuickFilter={selectedQuickFilter}
          runningIds={hub.runningIds}
          onSelectType={setSelectedType}
          onSelectCategory={setSelectedCategory}
          onSelectStatus={setSelectedStatus}
          onSelectTag={setSelectedTag}
          onSelectQuickFilter={setSelectedQuickFilter}
          onCreateCategory={async (name) => {
            await hub.handleCreateCategory(name);
          }}
          onDeleteCategory={hub.handleDeleteCategory}
        />

        <main className="flex-1 overflow-y-auto p-6">
          <div className="mb-5 px-4 py-2.5 rounded-lg bg-surface border border-edge flex items-center justify-between gap-4 flex-wrap">
            <div className="flex items-center gap-3 flex-wrap min-w-0">
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center justify-center min-w-[30px] h-7 px-2 rounded-md bg-elevated border border-edge font-heading text-[15px] font-bold text-primary tabular-nums">
                  {filteredProjects.length}
                </span>
                <span className="text-[14px] font-medium text-primary">
                  dự án hiển thị
                </span>
                {filteredProjects.length !== hub.queryMatchedProjects.length && (
                  <span className="text-[13px] text-muted tabular-nums">
                    / {hub.queryMatchedProjects.length} tổng
                  </span>
                )}
              </div>

              {hasActiveQuery && (
                <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
                  Tìm: "{debouncedQuery}"
                  <button
                    type="button"
                    onClick={() => {
                      setQuery("");
                      setDebouncedQuery("");
                    }}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              )}

              {selectedCategory !== "all" && (
                <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
                  Nhóm:{" "}
                  {selectedCategory === "__uncategorized__"
                    ? "Chưa phân nhóm"
                    : selectedCategory}
                  <button
                    type="button"
                    onClick={() => setSelectedCategory("all")}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              )}

              {selectedType !== "all" && (
                <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
                  Loại: {selectedType.toUpperCase()}
                  <button
                    type="button"
                    onClick={() => setSelectedType("all")}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              )}

              {selectedStatus !== "all" && (
                <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
                  Trạng thái: {selectedStatus}
                  <button
                    type="button"
                    onClick={() => setSelectedStatus("all")}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              )}

              {selectedTag && (
                <span className="h-7 px-2.5 rounded-md bg-canvas border border-edge text-primary text-[12.5px] font-medium inline-flex items-center gap-1.5">
                  #{selectedTag}
                  <button
                    type="button"
                    onClick={() => setSelectedTag("")}
                    className="text-muted hover:text-primary"
                  >
                    ×
                  </button>
                </span>
              )}
            </div>

            <div className="flex items-center gap-2.5 flex-wrap">
              <div className="flex items-center gap-2">
                <ArrowUpDown className="w-3.5 h-3.5 text-muted shrink-0" />
                <span className="text-[12.5px] text-muted hidden sm:inline">
                  Sắp xếp:
                </span>
                <select
                  aria-label="Sắp xếp dự án"
                  value={sortBy}
                  onChange={(e) =>
                    handleSortByChange(e.target.value as ProjectSortBy)
                  }
                  className="h-8 px-2.5 rounded-md bg-canvas border border-edge text-[13px] font-medium text-primary focus:outline-none focus:border-muted transition-colors"
                >
                  <option value="name_asc">Tên dự án (A → Z)</option>
                  <option value="name_desc">Tên dự án (Z → A)</option>
                  <option value="updated_desc">Cập nhật gần nhất</option>
                  <option value="opened_desc">Mở chạy gần đây</option>
                  <option value="status_ready">Sẵn sàng chạy trước</option>
                  <option value="category_asc">Theo phân loại (Nhóm)</option>
                  <option value="type_asc">Theo môi trường (Web / EXE)</option>
                  <option value="relevance">Độ khớp tìm kiếm</option>
                </select>
              </div>

              {filteredProjects.length > 0 && (
                <>
                  {selectionMode ? (
                    <>
                      <span className="text-xs text-muted tabular-nums">
                        Đã chọn{" "}
                        <strong className="text-primary">
                          {selectedIds.size}
                        </strong>
                      </span>
                      <button
                        type="button"
                        onClick={handleToggleSelectAllVisible}
                        className="h-8 px-2.5 rounded-md border border-edge bg-canvas hover:bg-elevated text-xs font-medium text-primary transition-colors"
                      >
                        {allVisibleSelected
                          ? "Bỏ chọn đang hiện"
                          : `Chọn tất cả (${filteredProjects.length})`}
                      </button>
                      <button
                        type="button"
                        disabled={selectedIds.size === 0}
                        onClick={handleRequestBulkDelete}
                        className="h-8 px-3 rounded-md bg-status-danger hover:bg-status-danger/90 text-white text-xs font-semibold inline-flex items-center gap-1.5 disabled:opacity-40 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                        <span>Xóa ({selectedIds.size})</span>
                      </button>
                      <button
                        type="button"
                        onClick={handleExitSelection}
                        className="h-8 px-2.5 rounded-md border border-edge bg-canvas hover:bg-elevated text-xs text-muted hover:text-primary inline-flex items-center gap-1 transition-colors"
                      >
                        <X className="w-3.5 h-3.5" />
                        <span>Xong</span>
                      </button>
                    </>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setSelectionMode(true)}
                      className="h-8 px-3 rounded-md border border-edge bg-canvas hover:bg-elevated text-[13px] font-medium text-muted hover:text-primary inline-flex items-center gap-1.5 transition-colors"
                    >
                      <CheckSquare className="w-3.5 h-3.5" />
                      <span>Chọn nhiều</span>
                    </button>
                  )}
                </>
              )}
            </div>
          </div>

          {filteredProjects.length === 0 ? (
            <div className="h-[60vh] border border-dashed border-edge rounded-lg bg-surface/40 flex flex-col items-center justify-center text-center p-8">
              <FolderSearch className="w-9 h-9 text-muted mb-3" />
              <h2 className="font-heading text-base font-semibold text-primary">
                {hub.queryMatchedProjects.length === 0
                  ? "Chưa có dự án nào phù hợp"
                  : "Không tìm thấy dự án khớp bộ lọc"}
              </h2>
              <p className="text-[13px] text-muted max-w-md mt-2 leading-relaxed">
                {hub.queryMatchedProjects.length === 0
                  ? "Hãy khai báo thư mục chứa code hoặc Apps trong Cài đặt rồi bấm Scan Local, hoặc cấu hình Cloudflare R2 để đồng bộ từ Cloud."
                  : "Thử xóa bớt bộ lọc ở thanh bên trái hoặc điều chỉnh từ khóa tìm kiếm."}
              </p>
              {hub.queryMatchedProjects.length === 0 && !debouncedQuery && (
                <div className="mt-5 flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => setSettingsOpen(true)}
                    className={UI_BTN_SECONDARY}
                  >
                    <Settings className="w-4 h-4" />
                    <span>Mở Cài đặt</span>
                  </button>
                  <button
                    type="button"
                    onClick={hub.handleScan}
                    className={UI_BTN_PRIMARY}
                  >
                    Scan Local ngay
                  </button>
                </div>
              )}
            </div>
          ) : viewMode === "matrix" ? (
            <ReleaseMatrixTable
              projects={filteredProjects}
              healthMap={hub.healthMap}
              runningMap={hub.runningMap}
              busyAction={hub.busyCardId}
              selectionMode={selectionMode}
              selectedIds={selectedIds}
              onToggleSelect={handleToggleSelect}
              onToggleSelectAll={handleToggleSelectAllVisible}
              onRequestDelete={handleRequestSingleDelete}
              onSelect={setSelectedProject}
              onLaunch={hub.handleLaunch}
              onHydrate={hub.handleHydrate}
              onUpdateDesktop={hub.handleUpdateDesktopFromList}
              onOpenLive={hub.handleOpenLive}
              onOpenLocalPort={hub.handleOpenLocalPort}
              onStopProcess={hub.handleStopProcess}
              onCopyReleaseLink={hub.handleCopyReleaseLink}
              onOpenExplorer={hub.handleOpenExplorer}
              onOpenVSCode={hub.handleOpenVSCode}
            />
          ) : (
            <div className="grid grid-cols-[repeat(auto-fill,minmax(290px,1fr))] gap-5">
              {filteredProjects.map((proj) => (
                <ProjectCard
                  key={proj.id}
                  project={proj}
                  runningProc={hub.runningMap[proj.id]}
                  busyAction={hub.busyCardId}
                  selectionMode={selectionMode}
                  selected={selectedIds.has(proj.id)}
                  onToggleSelect={handleToggleSelect}
                  onRequestDelete={handleRequestSingleDelete}
                  onSelect={setSelectedProject}
                  onLaunch={hub.handleLaunch}
                  onHydrate={hub.handleHydrate}
                  onOpenLive={hub.handleOpenLive}
                  onStopProcess={hub.handleStopProcess}
                  onOpenLocalPort={hub.handleOpenLocalPort}
                  onOpenExplorer={hub.handleOpenExplorer}
                  onOpenVSCode={hub.handleOpenVSCode}
                  onUploadThumbnail={hub.handleUploadThumbnail}
                  onUploadError={handleUploadError}
                  onTagClick={handleTagClick}
                  onCategoryClick={handleCategoryClick}
                />
              ))}
            </div>
          )}
        </main>
      </div>

      <ProjectDetailModal
        project={selectedProject}
        categories={hub.categories}
        onClose={() => setSelectedProject(null)}
        onUpdated={handleProjectUpdated}
        onRequestDelete={handleRequestSingleDelete}
        onCreateCategory={hub.handleCreateCategory}
        onNotify={notify}
      />

      <ConfirmDeleteModal
        projects={pendingDelete}
        busy={hub.deletingProjects}
        onCancel={() => setPendingDelete([])}
        onConfirm={handleConfirmDelete}
      />

      <SettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        onSaved={() => hub.refreshAll()}
        onNotify={notify}
      />

      <AddProjectModal
        open={addOpen}
        categories={hub.categories}
        onClose={() => setAddOpen(false)}
        onCreated={() => hub.refreshAll()}
        onCategoryCreated={hub.loadCategories}
        onNotify={notify}
      />

      <ToastStack toasts={toasts} onDismiss={dismiss} />
    </div>
  );
}

export default App;

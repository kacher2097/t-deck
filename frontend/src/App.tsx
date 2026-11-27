import { useCallback, useMemo, useState } from "react";
import { AddProjectModal } from "./components/AddProjectModal";
import { CatalogEmptyState } from "./components/CatalogEmptyState";
import { CatalogToolbar } from "./components/CatalogToolbar";
import { ConfirmDeleteModal } from "./components/ConfirmDeleteModal";
import { ProjectDetailModal } from "./components/inspector/ProjectDetailModal";
import { ProjectCard } from "./components/ProjectCard";
import { ReleaseMatrixTable } from "./components/ReleaseMatrixTable";
import {
  ProjectCardSkeleton,
  ReleaseMatrixSkeleton,
  ScanLoadingOverlay,
} from "./components/ScanLoadingOverlay";
import { SettingsModal } from "./components/settings/SettingsModal";
import { SidebarNav } from "./components/SidebarNav";
import { ToastStack } from "./components/ToastStack";
import { TopCommandBar } from "./components/TopCommandBar";
import { useBulkSelection } from "./hooks/useBulkSelection";
import { useProjectFilters } from "./hooks/useProjectFilters";
import { useProjectsHub } from "./hooks/useProjectsHub";
import { useToast } from "./hooks/useToast";
import { applyFacets } from "./lib/facets";
import type { ProjectView } from "./types/project";

export function App() {
  const [selectedProject, setSelectedProject] = useState<ProjectView | null>(
    null
  );
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  const { toasts, notify, dismiss } = useToast();
  const filters = useProjectFilters();

  const handleRefreshSelectedThumbnail = useCallback((projectId: string) => {
    setSelectedProject((prev) =>
      prev?.id === projectId ? { ...prev } : prev
    );
  }, []);

  const hub = useProjectsHub({
    debouncedQuery: filters.debouncedQuery,
    selectedCategory: filters.selectedCategory,
    onResetCategory: filters.handleResetCategory,
    onSelectProject: setSelectedProject,
    onRefreshSelectedThumbnail: handleRefreshSelectedThumbnail,
    notify,
  });

  const filteredProjects = useMemo(
    () =>
      applyFacets(hub.queryMatchedProjects, {
        type: filters.selectedType,
        category: filters.selectedCategory,
        status: filters.selectedStatus,
        tag: filters.selectedTag,
        quickFilter: filters.selectedQuickFilter,
        runningIds: hub.runningIds,
        preserveOrder: filters.hasActiveQuery,
        hasQuery: filters.hasActiveQuery,
        sortBy: filters.sortBy,
      }),
    [
      hub.queryMatchedProjects,
      filters.selectedType,
      filters.selectedCategory,
      filters.selectedStatus,
      filters.selectedTag,
      filters.selectedQuickFilter,
      hub.runningIds,
      filters.hasActiveQuery,
      filters.sortBy,
    ]
  );

  const handleAfterDelete = useCallback((removedIds: Set<string>) => {
    setSelectedProject((prev) =>
      prev && removedIds.has(prev.id) ? null : prev
    );
  }, []);

  const selection = useBulkSelection({
    allProjects: hub.queryMatchedProjects,
    visibleProjects: filteredProjects,
    onDeleteProjects: hub.handleDeleteProjects,
    onAfterDelete: handleAfterDelete,
  });

  const handleUploadError = useCallback(
    (msg: string) => notify("error", msg),
    [notify]
  );

  const handleProjectUpdated = useCallback(
    (updated: ProjectView) => {
      setSelectedProject(updated);
      hub.refreshAll();
    },
    [hub.refreshAll]
  );

  return (
    <div className="app-enter h-screen w-screen bg-canvas text-primary flex flex-col overflow-hidden">
      <TopCommandBar
        query={filters.query}
        onQueryChange={filters.setQuery}
        onSearchCommit={filters.handleSearchCommit}
        onCompositionChange={filters.setIsComposing}
        searchInputRef={filters.searchInputRef}
        viewMode={filters.viewMode}
        onViewModeChange={filters.handleViewModeChange}
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
          selectedType={filters.selectedType}
          selectedCategory={filters.selectedCategory}
          selectedStatus={filters.selectedStatus}
          selectedTag={filters.selectedTag}
          selectedQuickFilter={filters.selectedQuickFilter}
          runningIds={hub.runningIds}
          onSelectType={filters.setSelectedType}
          onSelectCategory={filters.setSelectedCategory}
          onSelectStatus={filters.setSelectedStatus}
          onSelectTag={filters.setSelectedTag}
          onSelectQuickFilter={filters.setSelectedQuickFilter}
          onCreateCategory={async (name) => {
            await hub.handleCreateCategory(name);
          }}
          onDeleteCategory={hub.handleDeleteCategory}
        />

        <main className="flex-1 overflow-y-auto p-6">
          <CatalogToolbar
            filteredCount={filteredProjects.length}
            totalCount={hub.queryMatchedProjects.length}
            hasActiveQuery={filters.hasActiveQuery}
            debouncedQuery={filters.debouncedQuery}
            selectedCategory={filters.selectedCategory}
            selectedType={filters.selectedType}
            selectedStatus={filters.selectedStatus}
            selectedTag={filters.selectedTag}
            sortBy={filters.sortBy}
            selectionMode={selection.selectionMode}
            selectedCount={selection.selectedIds.size}
            allVisibleSelected={selection.allVisibleSelected}
            onClearQuery={filters.clearQuery}
            onClearCategory={() => filters.setSelectedCategory("all")}
            onClearType={() => filters.setSelectedType("all")}
            onClearStatus={() => filters.setSelectedStatus("all")}
            onClearTag={() => filters.setSelectedTag("")}
            onSortByChange={filters.handleSortByChange}
            onEnterSelectionMode={selection.enterSelectionMode}
            onToggleSelectAllVisible={selection.handleToggleSelectAllVisible}
            onRequestBulkDelete={selection.handleRequestBulkDelete}
            onExitSelection={selection.handleExitSelection}
          />

          {hub.scanning ? (
            filters.viewMode === "matrix" ? (
              <ReleaseMatrixSkeleton rows={8} />
            ) : (
              <div className="grid grid-cols-[repeat(auto-fill,minmax(290px,1fr))] gap-5">
                {Array.from({ length: 12 }).map((_, idx) => (
                  <ProjectCardSkeleton key={idx} index={idx} />
                ))}
              </div>
            )
          ) : filteredProjects.length === 0 ? (
            <CatalogEmptyState
              hasAnyProjects={hub.queryMatchedProjects.length > 0}
              hasActiveQuery={Boolean(filters.debouncedQuery)}
              onOpenSettings={() => setSettingsOpen(true)}
              onScan={hub.handleScan}
            />
          ) : filters.viewMode === "matrix" ? (
            <ReleaseMatrixTable
              projects={filteredProjects}
              healthMap={hub.healthMap}
              runningMap={hub.runningMap}
              busyAction={hub.busyCardId}
              selectionMode={selection.selectionMode}
              selectedIds={selection.selectedIds}
              onToggleSelect={selection.handleToggleSelect}
              onToggleSelectAll={selection.handleToggleSelectAllVisible}
              onRequestDelete={selection.handleRequestSingleDelete}
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
              {filteredProjects.map((proj, idx) => (
                <div
                  key={
                    hub.scanRevealTick > 0
                      ? `${proj.id}-scan-${hub.scanRevealTick}`
                      : proj.id
                  }
                  style={
                    hub.scanRevealTick > 0
                      ? { animationDelay: `${Math.min(idx * 35, 560)}ms` }
                      : undefined
                  }
                  className={
                    hub.scanRevealTick > 0
                      ? "card-stagger-reveal min-w-0"
                      : "min-w-0"
                  }
                >
                  <ProjectCard
                    project={proj}
                    runningProc={hub.runningMap[proj.id]}
                    busyAction={hub.busyCardId}
                    selectionMode={selection.selectionMode}
                    selected={selection.selectedIds.has(proj.id)}
                    onToggleSelect={selection.handleToggleSelect}
                    onRequestDelete={selection.handleRequestSingleDelete}
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
                    onTagClick={filters.handleTagClick}
                    onCategoryClick={filters.handleCategoryClick}
                  />
                </div>
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
        onRequestDelete={selection.handleRequestSingleDelete}
        onCreateCategory={hub.handleCreateCategory}
        onNotify={notify}
      />

      <ConfirmDeleteModal
        projects={selection.pendingDelete}
        busy={hub.deletingProjects}
        onCancel={selection.cancelPendingDelete}
        onConfirm={selection.handleConfirmDelete}
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

      {hub.scanning && <ScanLoadingOverlay />}

      <ToastStack toasts={toasts} onDismiss={dismiss} />
    </div>
  );
}

export default App;

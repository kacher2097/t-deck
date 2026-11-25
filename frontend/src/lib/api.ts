import type {
  BatTemplate,
  BuildPublishRequest,
  BulkDeleteResult,
  Category,
  EnvSyncResult,
  LaunchResult,
  ProjectFilter,
  ProjectInput,
  ProjectProbe,
  ProjectTarget,
  ProjectView,
  PublishReleaseRequest,
  PublishRequest,
  R2Config,
  Release,
  RunningProcess,
  ScanReport,
  Settings,
  SyncReport,
  WebHealthResult,
} from "../types/project";

interface WailsAppBridge {
  ListProjects(filter: ProjectFilter): Promise<ProjectView[]>;
  GetProjectDetail(id: string): Promise<ProjectView>;
  UpsertProject(input: ProjectInput): Promise<ProjectView>;
  DeleteProject(id: string, deleteLocalFiles: boolean): Promise<void>;
  DeleteProjects(
    ids: string[],
    deleteLocalFiles: boolean
  ): Promise<BulkDeleteResult>;
  ListCategories(): Promise<Category[]>;
  CreateCategory(name: string): Promise<Category>;
  RenameCategory(id: string, newName: string): Promise<Category>;
  DeleteCategory(id: string): Promise<void>;
  ScanLocalWorkspaces(): Promise<ScanReport>;
  SaveClipboardThumbnail(projectID: string, base64PNG: string): Promise<string>;
  GenerateProjectThumbnail(projectID: string): Promise<string>;
  SyncEnvExample(projectID: string): Promise<EnvSyncResult>;
  GenerateStartBat(projectID: string, template: string): Promise<void>;
  GetBatTemplates(): Promise<BatTemplate[]>;
  LaunchProject(projectID: string): Promise<LaunchResult>;
  OpenInExplorer(projectID: string): Promise<void>;
  OpenInEditor(projectID: string, editor: string): Promise<void>;
  SyncCatalogWithR2(): Promise<SyncReport>;
  PublishDesktopRelease(req: PublishRequest): Promise<ProjectView>;
  HydrateProjectOnNewPC(projectID: string): Promise<ProjectView>;
  GetSettings(): Promise<Settings>;
  SaveSettings(s: Settings): Promise<Settings>;
  TestR2Connection(cfg: R2Config): Promise<void>;
  SelectDirectory(title: string): Promise<string>;
  SelectExecutableFile(title: string): Promise<string>;
  InspectLocalDirectory(dir: string): Promise<ProjectProbe>;

  ListProjectTargets(projectID: string): Promise<ProjectTarget[]>;
  UpsertTarget(target: ProjectTarget): Promise<ProjectTarget>;
  DeleteTarget(targetID: string): Promise<void>;
  ListReleases(projectID: string, targetID: string): Promise<Release[]>;
  PublishRelease(req: PublishReleaseRequest): Promise<Release>;
  DeleteRelease(releaseID: string, deleteR2Object: boolean): Promise<void>;
  UpdateDesktopRelease(
    projectID: string,
    targetID: string,
    releaseID: string
  ): Promise<ProjectView>;
  RollbackDesktopRelease(
    projectID: string,
    targetID: string
  ): Promise<ProjectView>;
  GenerateReleaseDownloadURL(
    releaseID: string,
    expiryHours: number
  ): Promise<string>;
  CheckWebHealth(targetID: string): Promise<WebHealthResult>;
  CheckAllWebHealth(): Promise<WebHealthResult[]>;
  OpenTargetURL(targetID: string, env: string): Promise<void>;

  SuggestReleaseChangelog(projectID: string, targetID: string): Promise<string>;
  BuildAndPublishTarget(req: BuildPublishRequest): Promise<string>;
  CancelActiveJob(jobID: string): Promise<void>;
  ListRunningProcesses(): Promise<RunningProcess[]>;
  StopProjectProcess(projectID: string): Promise<void>;
  OpenLocalhostPort(port: number): Promise<void>;
  ImportProjectManifest(projectID: string): Promise<ProjectView>;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: WailsAppBridge;
      };
    };
  }
}

function bridge(): WailsAppBridge {
  const app = window.go?.main?.App;
  if (!app) {
    throw new Error("Wails backend bridge chưa sẵn sàng.");
  }
  return app;
}

export const api = {
  listProjects: (filter: ProjectFilter) => bridge().ListProjects(filter),
  getProjectDetail: (id: string) => bridge().GetProjectDetail(id),
  upsertProject: (input: ProjectInput) => bridge().UpsertProject(input),
  deleteProject: (id: string, deleteLocalFiles: boolean) =>
    bridge().DeleteProject(id, deleteLocalFiles),
  deleteProjects: (ids: string[], deleteLocalFiles: boolean) =>
    bridge().DeleteProjects(ids, deleteLocalFiles),
  listCategories: () => bridge().ListCategories(),
  createCategory: (name: string) => bridge().CreateCategory(name),
  renameCategory: (id: string, newName: string) =>
    bridge().RenameCategory(id, newName),
  deleteCategory: (id: string) => bridge().DeleteCategory(id),
  scanLocalWorkspaces: () => bridge().ScanLocalWorkspaces(),
  saveClipboardThumbnail: (projectID: string, base64PNG: string) =>
    bridge().SaveClipboardThumbnail(projectID, base64PNG),
  generateProjectThumbnail: (projectID: string) =>
    bridge().GenerateProjectThumbnail(projectID),
  syncEnvExample: (projectID: string) => bridge().SyncEnvExample(projectID),
  generateStartBat: (projectID: string, template: string) =>
    bridge().GenerateStartBat(projectID, template),
  getBatTemplates: () => bridge().GetBatTemplates(),
  launchProject: (projectID: string) => bridge().LaunchProject(projectID),
  openInExplorer: (projectID: string) => bridge().OpenInExplorer(projectID),
  openInEditor: (projectID: string, editor: string) =>
    bridge().OpenInEditor(projectID, editor),
  syncCatalogWithR2: () => bridge().SyncCatalogWithR2(),
  publishDesktopRelease: (req: PublishRequest) =>
    bridge().PublishDesktopRelease(req),
  hydrateProjectOnNewPC: (projectID: string) =>
    bridge().HydrateProjectOnNewPC(projectID),
  getSettings: () => bridge().GetSettings(),
  saveSettings: (s: Settings) => bridge().SaveSettings(s),
  testR2Connection: (cfg: R2Config) => bridge().TestR2Connection(cfg),
  selectDirectory: (title: string) => bridge().SelectDirectory(title),
  selectExecutableFile: (title: string) => bridge().SelectExecutableFile(title),
  inspectLocalDirectory: (dir: string) => bridge().InspectLocalDirectory(dir),

  listProjectTargets: (projectID: string) =>
    bridge().ListProjectTargets(projectID),
  upsertTarget: (target: ProjectTarget) => bridge().UpsertTarget(target),
  deleteTarget: (targetID: string) => bridge().DeleteTarget(targetID),
  listReleases: (projectID: string, targetID = "") =>
    bridge().ListReleases(projectID, targetID),
  publishRelease: (req: PublishReleaseRequest) => bridge().PublishRelease(req),
  deleteRelease: (releaseID: string, deleteR2Object: boolean) =>
    bridge().DeleteRelease(releaseID, deleteR2Object),
  updateDesktopRelease: (
    projectID: string,
    targetID = "",
    releaseID = ""
  ) => bridge().UpdateDesktopRelease(projectID, targetID, releaseID),
  rollbackDesktopRelease: (projectID: string, targetID = "") =>
    bridge().RollbackDesktopRelease(projectID, targetID),
  generateReleaseDownloadURL: (releaseID: string, expiryHours = 24) =>
    bridge().GenerateReleaseDownloadURL(releaseID, expiryHours),
  checkWebHealth: (targetID: string) => bridge().CheckWebHealth(targetID),
  checkAllWebHealth: () => bridge().CheckAllWebHealth(),
  openTargetURL: (targetID: string, env = "prod") =>
    bridge().OpenTargetURL(targetID, env),

  suggestReleaseChangelog: (projectID: string, targetID = "") =>
    bridge().SuggestReleaseChangelog(projectID, targetID),
  buildAndPublishTarget: (req: BuildPublishRequest) =>
    bridge().BuildAndPublishTarget(req),
  cancelActiveJob: (jobID: string) => bridge().CancelActiveJob(jobID),
  listRunningProcesses: () => bridge().ListRunningProcesses(),
  stopProjectProcess: (projectID: string) =>
    bridge().StopProjectProcess(projectID),
  openLocalhostPort: (port: number) => bridge().OpenLocalhostPort(port),
  importProjectManifest: (projectID: string) =>
    bridge().ImportProjectManifest(projectID),
};

export function formatError(err: unknown): string {
  let raw = "";
  if (err instanceof Error) {
    raw = err.message;
  } else if (typeof err === "string") {
    raw = err;
  } else if (err && typeof err === "object" && "message" in err) {
    raw = String((err as { message?: unknown }).message ?? "");
  }
  const cleaned = raw
    .replace(/^\[(VALIDATION|NOT_FOUND|NOT_READY|CONFLICT|CLOUD|IO|INTERNAL)\]\s*/i, "")
    .trim();
  return cleaned || "Đã xảy ra lỗi không xác định";
}

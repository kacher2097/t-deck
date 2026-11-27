export type ProjectType = "web" | "desktop" | "hybrid";

export type ProjectStatus = "ready" | "cloud_only" | "missing_env" | "missing_dir";

export type ReleaseChannel = "stable" | "beta" | "canary";

export interface Category {
  id: string;
  name: string;
  slug: string;
  sort_order: number;
  created_at: number;
  updated_at?: number;
}

export interface ProjectTarget {
  id: string;
  project_id: string;
  kind: "web" | "desktop";
  name: string;
  platform: string;
  distribution: string;
  subpath?: string;
  build_cmd?: string;
  artifact_glob?: string;
  launch_entry?: string;
  prod_url?: string;
  staging_url?: string;
  health_url?: string;
  version_url?: string;
  hosting_provider?: string;
  sort_order: number;
  updated_at: number;
}

export interface Release {
  id: string;
  project_id: string;
  target_id: string;
  version: string;
  channel: ReleaseChannel | string;
  git_tag?: string;
  commit_sha?: string;
  changelog?: string;
  r2_key?: string;
  file_name?: string;
  file_size: number;
  sha256?: string;
  deploy_url?: string;
  asset_kind: string;
  status: string;
  published_at: number;
}

export interface WebHealthResult {
  target_id: string;
  project_id: string;
  online: boolean;
  status_code: number;
  latency_ms: number;
  live_version: string;
  drift: boolean;
  checked_at: number;
  error?: string;
}

export interface RunningProcess {
  project_id: string;
  pid: number;
  ports: number[];
  started_at: number;
  command: string;
}

export interface ProjectView {
  id: string;
  slug: string;
  name: string;
  type: ProjectType;
  category: string;
  summary: string;
  tags: string[];
  tech_stack: string;
  git_remote: string;
  bat_file: string;
  exe_name: string;
  latest_version: string;
  r2_release_key: string;
  release_notes: string;
  thumbnail_updated_at: number;
  updated_at: number;

  local_dir: string;
  status: ProjectStatus;
  last_opened_at: number;
  has_env: boolean;
  has_env_example: boolean;
  has_bat: boolean;
  thumbnail_url: string;
  icon_url?: string;
  installed_version: string;
  git_branch: string;
  git_commit: string;
  git_dirty: boolean;
  targets: ProjectTarget[];
  latest_releases: Record<string, Release>;
  has_update: boolean;
  live_url: string;
}

export interface ProjectFilter {
  type: string;
  category: string;
  status: string;
  tag: string;
  query: string;
}

export interface ProjectInput {
  id: string;
  slug: string;
  name: string;
  type: ProjectType;
  category: string;
  summary: string;
  tags: string[];
  tech_stack: string;
  git_remote: string;
  bat_file: string;
  exe_name: string;
  local_dir: string;
  prod_url?: string;
}

export interface ProjectProbe {
  local_dir: string;
  name: string;
  slug: string;
  type: ProjectType;
  category: string;
  summary: string;
  tags: string[];
  tech_stack: string;
  git_remote: string;
  bat_file: string;
  exe_name: string;
  prod_url: string;
  version: string;
  has_env: boolean;
  has_bat: boolean;
  has_meta: boolean;
  existing_id?: string;
}

export interface PublishReleaseRequest {
  project_id: string;
  target_id: string;
  source_path: string;
  version: string;
  channel: string;
  asset_kind: string;
  deploy_url: string;
  changelog: string;
}

export interface BuildPublishRequest {
  project_id: string;
  target_id: string;
  version: string;
  channel: string;
  asset_kind: string;
  changelog: string;
}

export interface ScanReport {
  scanned_dirs: number;
  discovered: number;
  added: number;
  relinked: number;
  updated: number;
  missing: number;
  errors: string[];
}

export interface EnvSyncResult {
  project_id: string;
  env_path: string;
  example_path: string;
  vars_count: number;
}

export interface LaunchResult {
  project_id: string;
  status: string;
  message: string;
  opened_file: string;
}

export interface BulkDeleteResult {
  deleted: number;
  failed: string[];
}

export interface SyncReport {
  pulled: number;
  pushed: number;
  updated_local: number;
  updated_remote: number;
  thumbnails_down: number;
  thumbnails_up: number;
  synced_at: number;
  message: string;
}

export interface PublishRequest {
  project_id: string;
  source_path: string;
  version: string;
  release_notes: string;
}

export interface BatTemplate {
  id: string;
  label: string;
  command: string;
}

export interface R2Config {
  enabled: boolean;
  account_id: string;
  access_key_id: string;
  secret_access_key: string;
  bucket_name: string;
  public_domain: string;
}

export interface Settings {
  workspace_dirs: string[];
  apps_dir: string;
  scan_recent_days: number;
  scan_exclude_dirs: string[];
  r2: R2Config;
}

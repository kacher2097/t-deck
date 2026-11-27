package model

// Category represents a user-defined project group/classification.
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SortOrder int    `json:"sort_order"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
}

// Project represents the cloud-synced metadata stored in the projects table and catalog.json.
type Project struct {
	ID                 string   `json:"id"`
	Slug               string   `json:"slug"`
	Name               string   `json:"name"`
	Type               string   `json:"type"` // "web" | "desktop" | "hybrid"
	Category           string   `json:"category"`
	Summary            string   `json:"summary"`
	Tags               []string `json:"tags"`
	TechStack          string   `json:"tech_stack"`
	GitRemote          string   `json:"git_remote"`
	BatFile            string   `json:"bat_file"`
	ExeName            string   `json:"exe_name"`
	LatestVersion      string   `json:"latest_version"`
	R2ReleaseKey       string   `json:"r2_release_key"`
	ReleaseNotes       string   `json:"release_notes"`
	ThumbnailUpdatedAt int64    `json:"thumbnail_updated_at"`
	UpdatedAt          int64    `json:"updated_at"`
	DeletedAt          int64    `json:"deleted_at,omitempty"`
}

// LocalState represents machine-specific path and status stored in local_state.
type LocalState struct {
	ProjectID          string `json:"project_id"`
	LocalDir           string `json:"local_dir"`
	Status             string `json:"status"` // ready | cloud_only | missing_env | missing_dir
	LastOpenedAt       int64  `json:"last_opened_at"`
	HasEnv             bool   `json:"has_env"`
	HasEnvExample      bool   `json:"has_env_example"`
	HasBat             bool   `json:"has_bat"`
	FlagsCheckedAt     int64  `json:"flags_checked_at"`
	InstalledVersion   string `json:"installed_version,omitempty"`
	InstalledReleaseID string `json:"installed_release_id,omitempty"`
	InstalledAt        int64  `json:"installed_at,omitempty"`
	GitBranch          string `json:"git_branch,omitempty"`
	GitCommit          string `json:"git_commit,omitempty"`
	GitDirty           bool   `json:"git_dirty,omitempty"`
}

// ProjectType values include "web", "desktop", and "hybrid".

// ProjectTarget represents one deployable/launchable target (web or desktop) owned by a Project.
type ProjectTarget struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	Kind            string `json:"kind"` // "web" | "desktop"
	Name            string `json:"name"`
	Platform        string `json:"platform"`     // "web" | "windows" | "macos" | "linux"
	Distribution    string `json:"distribution"` // "hosted" | "static_zip" | "portable" | "installer"
	Subpath         string `json:"subpath,omitempty"`
	BuildCmd        string `json:"build_cmd,omitempty"`
	ArtifactGlob    string `json:"artifact_glob,omitempty"`
	LaunchEntry     string `json:"launch_entry,omitempty"`
	ProdURL         string `json:"prod_url,omitempty"`
	StagingURL      string `json:"staging_url,omitempty"`
	HealthURL       string `json:"health_url,omitempty"`
	VersionURL      string `json:"version_url,omitempty"`
	HostingProvider string `json:"hosting_provider,omitempty"`
	SortOrder       int    `json:"sort_order"`
	UpdatedAt       int64  `json:"updated_at"`
	DeletedAt       int64  `json:"deleted_at,omitempty"`
}

// Release represents a published version record for either a Web or Desktop target.
type Release struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	TargetID    string `json:"target_id"`
	Version     string `json:"version"`
	Channel     string `json:"channel"` // "stable" | "beta" | "canary"
	GitTag      string `json:"git_tag,omitempty"`
	CommitSHA   string `json:"commit_sha,omitempty"`
	Changelog   string `json:"changelog,omitempty"`
	R2Key       string `json:"r2_key,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	FileSize    int64  `json:"file_size"`
	SHA256      string `json:"sha256,omitempty"`
	DeployURL   string `json:"deploy_url,omitempty"`
	AssetKind   string `json:"asset_kind"` // "portable_zip" | "installer_exe" | "static_zip" | "external_url"
	Status      string `json:"status"`     // "published" | "draft" | "yanked"
	PublishedAt int64  `json:"published_at"`
	DeletedAt   int64  `json:"deleted_at,omitempty"`
}

// ProjectReleaseIndex stores the per-project release history JSON on R2 at releases/<slug>/index.json.
type ProjectReleaseIndex struct {
	Slug      string    `json:"slug"`
	UpdatedAt int64     `json:"updated_at"`
	Releases  []Release `json:"releases"`
}

// PublishReleaseRequest describes publishing a release for a specific Web or Desktop target.
type PublishReleaseRequest struct {
	ProjectID  string `json:"project_id"`
	TargetID   string `json:"target_id"`
	SourcePath string `json:"source_path"`
	Version    string `json:"version"`
	Channel    string `json:"channel"`
	AssetKind  string `json:"asset_kind"`
	DeployURL  string `json:"deploy_url"`
	Changelog  string `json:"changelog"`
}

// WebHealthResult reports HTTP liveness, latency, and live version drift for a Web target.
type WebHealthResult struct {
	TargetID    string `json:"target_id"`
	ProjectID   string `json:"project_id"`
	Online      bool   `json:"online"`
	StatusCode  int    `json:"status_code"`
	LatencyMs   int64  `json:"latency_ms"`
	LiveVersion string `json:"live_version"`
	Drift       bool   `json:"drift"`
	CheckedAt   int64  `json:"checked_at"`
	Error       string `json:"error,omitempty"`
}

// ProjectView combines cloud metadata and local machine state for UI rendering.
type ProjectView struct {
	ID                 string   `json:"id"`
	Slug               string   `json:"slug"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Category           string   `json:"category"`
	Summary            string   `json:"summary"`
	Tags               []string `json:"tags"`
	TechStack          string   `json:"tech_stack"`
	GitRemote          string   `json:"git_remote"`
	BatFile            string   `json:"bat_file"`
	ExeName            string   `json:"exe_name"`
	LatestVersion      string   `json:"latest_version"`
	R2ReleaseKey       string   `json:"r2_release_key"`
	ReleaseNotes       string   `json:"release_notes"`
	ThumbnailUpdatedAt int64    `json:"thumbnail_updated_at"`
	UpdatedAt          int64    `json:"updated_at"`

	LocalDir         string             `json:"local_dir"`
	Status           string             `json:"status"`
	LastOpenedAt     int64              `json:"last_opened_at"`
	HasEnv           bool               `json:"has_env"`
	HasEnvExample    bool               `json:"has_env_example"`
	HasBat           bool               `json:"has_bat"`
	ThumbnailURL     string             `json:"thumbnail_url"`
	IconURL          string             `json:"icon_url,omitempty"`
	InstalledVersion string             `json:"installed_version"`
	GitBranch        string             `json:"git_branch"`
	GitCommit        string             `json:"git_commit"`
	GitDirty         bool               `json:"git_dirty"`
	Targets          []ProjectTarget    `json:"targets"`
	LatestReleases   map[string]Release `json:"latest_releases"`
	HasUpdate        bool               `json:"has_update"`
	LiveURL          string             `json:"live_url"`
}

// ProjectFilter defines search and sidebar filter parameters.
type ProjectFilter struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Tag      string `json:"tag"`
	Query    string `json:"query"`
}

// ProjectInput represents user-editable fields for creating or updating a project.
type ProjectInput struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Category  string   `json:"category"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	TechStack string   `json:"tech_stack"`
	GitRemote string   `json:"git_remote"`
	BatFile   string   `json:"bat_file"`
	ExeName   string   `json:"exe_name"`
	LocalDir  string   `json:"local_dir"`
	ProdURL   string   `json:"prod_url,omitempty"`
}

// ProjectProbe holds auto-detected metadata extracted from a local project directory.
type ProjectProbe struct {
	LocalDir   string   `json:"local_dir"`
	Name       string   `json:"name"`
	Slug       string   `json:"slug"`
	Type       string   `json:"type"`
	Category   string   `json:"category"`
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	TechStack  string   `json:"tech_stack"`
	GitRemote  string   `json:"git_remote"`
	BatFile    string   `json:"bat_file"`
	ExeName    string   `json:"exe_name"`
	ProdURL    string   `json:"prod_url"`
	Version    string   `json:"version"`
	HasEnv     bool     `json:"has_env"`
	HasBat     bool     `json:"has_bat"`
	HasMeta    bool     `json:"has_meta"`
	ExistingID string   `json:"existing_id,omitempty"`
}

// ProjectMeta is the anchor file (.tdeck-meta.json) stored inside each project root.
type ProjectMeta struct {
	ProjectID        string `json:"project_id"`
	Slug             string `json:"slug"`
	Type             string `json:"type"`
	InstalledVersion string `json:"installed_version,omitempty"`
}

// ScanReport summarizes a workspace scan run.
type ScanReport struct {
	ScannedDirs int      `json:"scanned_dirs"`
	Discovered  int      `json:"discovered"`
	Added       int      `json:"added"`
	Relinked    int      `json:"relinked"`
	Updated     int      `json:"updated"`
	Missing     int      `json:"missing"`
	Errors      []string `json:"errors"`
}

// EnvSyncResult summarizes generating .env.example from .env.
type EnvSyncResult struct {
	ProjectID   string `json:"project_id"`
	EnvPath     string `json:"env_path"`
	ExamplePath string `json:"example_path"`
	VarsCount   int    `json:"vars_count"`
}

// LaunchResult informs the UI of launch outcomes, including Pre-flight .env checks.
type LaunchResult struct {
	ProjectID  string `json:"project_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	OpenedFile string `json:"opened_file"`
}

// BulkDeleteResult reports the outcome of a batch project removal.
type BulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	Failed  []string `json:"failed"`
}

// SyncReport summarizes a two-way R2 catalog and thumbnail sync.
type SyncReport struct {
	Pulled         int    `json:"pulled"`
	Pushed         int    `json:"pushed"`
	UpdatedLocal   int    `json:"updated_local"`
	UpdatedRemote  int    `json:"updated_remote"`
	ThumbnailsDown int    `json:"thumbnails_down"`
	ThumbnailsUp   int    `json:"thumbnails_up"`
	SyncedAt       int64  `json:"synced_at"`
	Message        string `json:"message"`
}

// PublishRequest describes a desktop release packaging and upload request.
type PublishRequest struct {
	ProjectID    string `json:"project_id"`
	SourcePath   string `json:"source_path"`
	Version      string `json:"version"`
	ReleaseNotes string `json:"release_notes"`
}

// Catalog represents the remote catalog.json payload stored on R2.
type Catalog struct {
	Version    int             `json:"version"`
	UpdatedAt  int64           `json:"updated_at"`
	Categories []Category      `json:"categories,omitempty"`
	Projects   []Project       `json:"projects"`
	Targets    []ProjectTarget `json:"targets,omitempty"`
}

// AutoUpdateManifest represents releases/<slug>/<channel>/latest.json on R2.
type AutoUpdateManifest struct {
	Slug      string                        `json:"slug"`
	Version   string                        `json:"version"`
	Channel   string                        `json:"channel"`
	PubDate   string                        `json:"pub_date"`
	Notes     string                        `json:"notes"`
	Platforms map[string]AutoUpdatePlatform `json:"platforms"`
}

// AutoUpdatePlatform describes one OS/arch download payload inside latest.json.
type AutoUpdatePlatform struct {
	URL          string `json:"url"`
	InstallerURL string `json:"installer_url,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Size         int64  `json:"size"`
}

// BuildPublishRequest triggers an async build_cmd + publish job (Phase 3).
type BuildPublishRequest struct {
	ProjectID string `json:"project_id"`
	TargetID  string `json:"target_id"`
	Version   string `json:"version"`
	Channel   string `json:"channel"`
	AssetKind string `json:"asset_kind"`
	Changelog string `json:"changelog"`
}

// RunningProcess describes a local running child process and its detected listening TCP ports.
type RunningProcess struct {
	ProjectID string `json:"project_id"`
	PID       int    `json:"pid"`
	Ports     []int  `json:"ports"`
	StartedAt int64  `json:"started_at"`
	Command   string `json:"command"`
}

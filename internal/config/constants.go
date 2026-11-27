package config

// System file names, directories, and R2 object key prefixes.
// All system identifiers are centralized here to avoid hardcoded literals.
const (
	AppName           = "TDeck"
	AppDisplayName    = "T-Deck"
	SettingsFileName  = "settings.json"
	DatabaseFileName  = "tdeck.db"
	ThumbnailsDirName = "thumbnails"

	MetaFileName   = ".tdeck-meta.json"
	DefaultBatFile = "start.bat"
	EnvFile        = ".env"
	EnvExampleFile = ".env.example"
	ReadmeFile     = "README.md"
	WailsConfig    = "wails.json"
	GitConfigRel   = ".git/config"

	CatalogKey        = "catalog.json"
	ThumbnailsPrefix  = "thumbnails/"
	ReleasesPrefix    = "releases/"
	DefaultBucketName = "my-storage"

	MaxScanDepth           = 4
	MaxReadmeSummaryLen    = 500
	DefaultScanRecentDays  = 120
	TombstoneRetentionDays = 90
	ManifestFileName       = ".tdeck.json"
)

// Release channels.
const (
	ReleaseChannelStable = "stable"
	ReleaseChannelBeta   = "beta"
	ReleaseChannelCanary = "canary"
)

// Environment variable keys externalizing all config, API keys, and secrets.
const (
	EnvKeyWorkspaceDirs   = "TDECK_WORKSPACE_DIRS"
	EnvKeyAppsDir         = "TDECK_APPS_DIR"
	EnvKeyScanRecentDays  = "TDECK_SCAN_RECENT_DAYS"
	EnvKeyScanExcludeDirs = "TDECK_SCAN_EXCLUDE_DIRS"

	EnvKeyR2Enabled      = "R2_ENABLED"
	EnvKeyR2AccountID    = "R2_ACCOUNT_ID"
	EnvKeyR2Endpoint     = "R2_ENDPOINT"
	EnvKeyR2AccessKey    = "R2_ACCESS_KEY"
	EnvKeyR2SecretKey    = "R2_SECRET_KEY"
	EnvKeyR2Bucket       = "R2_BUCKET"
	EnvKeyR2PublicDomain = "R2_PUBLIC_DOMAIN"
)

// Project types.
const (
	ProjectTypeWeb     = "web"
	ProjectTypeDesktop = "desktop"
	ProjectTypeHybrid  = "hybrid"
)

// Local project states.
const (
	StatusReady      = "ready"
	StatusCloudOnly  = "cloud_only"
	StatusMissingEnv = "missing_env"
	StatusMissingDir = "missing_dir"
)

// Launch result statuses.
const (
	LaunchStatusLaunched              = "LAUNCHED"
	LaunchStatusEnvCreatedNeedsConfig = "ENV_CREATED_NEEDS_CONFIG"
	LaunchStatusMissingBat            = "MISSING_BAT"
	LaunchStatusMissingExe            = "MISSING_EXE"
)

// Editor targets supported by OpenInEditor.
const (
	EditorVSCode   = "code"
	EditorIntelliJ = "idea"
	EditorTerminal = "terminal"
)

// IgnoredDirs are skipped during recursive workspace scanning.
var IgnoredDirs = []string{
	"node_modules",
	".git",
	"target",
	"vendor",
	".venv",
	"venv",
	".graphify-venv",
	"__pycache__",
	"__macosx",
	"dist",
	"build",
	".next",
	".nuxt",
	".idea",
	".vscode",
	".work",
	".runtime",
	".tmp",
	".superpowers",
	".pnpm-store",
	"$recycle.bin",
	"$mfedeeprem",
	"$windows.~tmp",
	"system volume information",
	"recovery",
	"config.msi",
	"perflogs",
	"programdata",
	"program files",
	"program files (x86)",
	"windows",
	"msocache",
	"agent-runtime",
	"agent-skills-master",
	"model_ai",
	"postgresql",
	"mongodb",
	"mysql",
	"laragon",
	"xampp",
	"netsarang",
	"tableplus",
	"dbeaver",
	"hibit uninstaller",
	"antigravity ide",
	"cursor",
	"jetbrains",
	"obs-studio",
	"openvpn",
	"softether vpn client",
	"steam",
	"riot games",
	"half life",
	"treesize",
	"nvm4w",
	"locales",
	"resources",
	"runtimes",
	"jre",
	"inspection",
	"extracted",
}

// ProjectMarkerFiles identify a directory as a managed or discoverable project.
var ProjectMarkerFiles = []string{
	"package.json",
	"go.mod",
	"pom.xml",
	"requirements.txt",
	"gradlew.bat",
	"build.gradle",
	"build.gradle.kts",
	DefaultBatFile,
}

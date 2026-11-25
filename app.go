package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"t-deck/internal/apperr"
	"t-deck/internal/cloud"
	"t-deck/internal/config"
	"t-deck/internal/database"
	"t-deck/internal/launcher"
	"t-deck/internal/media"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

// App exposes T-Deck backend services as a thin Wails facade.
// Domain methods are split by responsibility across app_projects.go, app_releases.go, app_runtime.go, and app_settings.go.
type App struct {
	ctx         context.Context
	settingsMgr *config.SettingsManager
	db          *sql.DB
	repo        *database.ProjectRepo
	scanner     *scanner.Scanner
	runner      *launcher.Runner
	thumbs      *media.ThumbnailService
	procMonitor *launcher.ProcessMonitor

	r2Mu       sync.RWMutex
	r2CacheKey string
	r2Client   *cloud.R2Client

	jobsMu     sync.Mutex
	activeJobs map[string]context.CancelFunc
}

// NewApp creates a new App instance.
func NewApp() *App {
	mgr, err := config.NewSettingsManager("")
	if err != nil {
		mgr, _ = config.NewSettingsManager(filepath.Join(".", ".tdeck-data"))
	}
	return &App{
		settingsMgr: mgr,
		thumbs:      media.NewThumbnailService(nil, thumbDirFromMgr(mgr)),
	}
}

func thumbDirFromMgr(mgr *config.SettingsManager) string {
	if mgr == nil {
		return ""
	}
	return mgr.ThumbnailsDir()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.initServices(); err != nil {
		wailsruntime.LogErrorf(ctx, "T-Deck init failed: %v", err)
	}
}

// domReady reveals the main window only after WebView2 DOM & dark canvas are ready (prevents startup white flash).
func (a *App) domReady(ctx context.Context) {
	if ctx != nil {
		wailsruntime.WindowShow(ctx)
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.procMonitor != nil {
		a.procMonitor.Close()
	}
	a.jobsMu.Lock()
	for id, cancel := range a.activeJobs {
		cancel()
		delete(a.activeJobs, id)
	}
	a.jobsMu.Unlock()
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) initServices() error {
	if a.settingsMgr == nil {
		mgr, err := config.NewSettingsManager("")
		if err != nil {
			return err
		}
		a.settingsMgr = mgr
	}
	db, err := database.Open(a.settingsMgr.DBPath())
	if err != nil {
		return err
	}
	a.db = db
	a.repo = database.NewProjectRepo(db, a.settingsMgr.ThumbnailsDir())
	a.scanner = scanner.NewScanner(a.repo)
	a.procMonitor = launcher.NewProcessMonitor(func(proc model.RunningProcess) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "process:changed", map[string]any{
				"project_id": proc.ProjectID,
				"running":    proc.PID > 0,
				"pid":        proc.PID,
				"ports":      proc.Ports,
			})
		}
	})
	a.runner = launcher.NewRunner(a.repo)
	a.runner.SetMonitor(a.procMonitor)
	a.thumbs = media.NewThumbnailService(a.repo, a.settingsMgr.ThumbnailsDir())
	a.activeJobs = make(map[string]context.CancelFunc)
	return nil
}

func (a *App) serveThumbnail(w http.ResponseWriter, r *http.Request) {
	defer apperr.Recover("serve-thumbnail", nil, a.logPanic)
	if a.thumbs == nil {
		http.NotFound(w, r)
		return
	}
	a.thumbs.ServeHTTP(w, r)
}

func (a *App) loadR2Store() (config.Settings, cloud.ObjectStore, error) {
	settings, err := a.settingsMgr.Load()
	if err != nil {
		return config.Settings{}, nil, err
	}
	cacheKey := fmt.Sprintf("%v|%s|%s|%s|%s",
		settings.R2.Enabled,
		settings.R2.AccountID,
		settings.R2.AccessKeyID,
		settings.R2.SecretAccessKey,
		settings.R2.BucketName,
	)

	a.r2Mu.RLock()
	if a.r2Client != nil && a.r2CacheKey == cacheKey {
		cached := a.r2Client
		a.r2Mu.RUnlock()
		return settings, cached, nil
	}
	a.r2Mu.RUnlock()

	store, err := cloud.NewR2Client(a.opContext(), settings.R2)
	if err != nil {
		return config.Settings{}, nil, err
	}

	a.r2Mu.Lock()
	a.r2CacheKey = cacheKey
	a.r2Client = store
	a.r2Mu.Unlock()
	return settings, store, nil
}

func (a *App) openSafeURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return apperr.Validation("chưa cấu hình URL")
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return apperr.Validation(fmt.Sprintf("URL không hợp lệ (chỉ hỗ trợ http/https): %s", trimmed))
	}
	if a.ctx == nil {
		return apperr.NotReady("wails context unavailable")
	}
	wailsruntime.BrowserOpenURL(a.ctx, trimmed)
	return nil
}

func (a *App) opContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) logPanic(scope string, r any, stack string) {
	if a.ctx != nil {
		wailsruntime.LogErrorf(a.ctx, "Recovered panic in %s: %v\n%s", scope, r, stack)
	}
}

func (a *App) requireRepo() error {
	if a.repo == nil {
		return apperr.NotReady("cơ sở dữ liệu chưa sẵn sàng")
	}
	return nil
}

package main

import (
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"t-deck/internal/apperr"
	"t-deck/internal/cloud"
	"t-deck/internal/config"
)

// --- Settings & Native OS Dialogs ---

func (a *App) GetSettings() (config.Settings, error) {
	return a.settingsMgr.Load()
}

func (a *App) SaveSettings(s config.Settings) (config.Settings, error) {
	saved, err := a.settingsMgr.Save(s)
	if err == nil {
		a.r2Mu.Lock()
		a.r2CacheKey = ""
		a.r2Client = nil
		a.r2Mu.Unlock()
	}
	return saved, err
}

func (a *App) TestR2Connection(cfg config.R2Config) error {
	cfg.Enabled = true
	client, err := cloud.NewR2Client(a.opContext(), cfg)
	if err != nil {
		return err
	}
	return client.TestConnection(a.opContext())
}

func (a *App) SelectDirectory(title string) (string, error) {
	if a.ctx == nil {
		return "", apperr.NotReady("wails context unavailable")
	}
	if strings.TrimSpace(title) == "" {
		title = "Chọn thư mục"
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
	})
}

func (a *App) SelectExecutableFile(title string) (string, error) {
	if a.ctx == nil {
		return "", apperr.NotReady("wails context unavailable")
	}
	if strings.TrimSpace(title) == "" {
		title = "Chọn file thực thi (.exe)"
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Executable (*.exe)", Pattern: "*.exe"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ProjectStore defines the persistence operations required by Runner (DIP).
type ProjectStore interface {
	GetProject(id string) (model.ProjectView, error)
	UpdateLocalStatus(projectID string, status string) error
	TouchLastOpened(projectID string) error
}

// ProcessStarter abstracts OS process execution for testability.
type ProcessStarter interface {
	Start(dir string, name string, args ...string) error
	LookPath(file string) (string, error)
}

type osProcessStarter struct{}

func (osProcessStarter) Start(dir string, name string, args ...string) error {
	_, err := startWithPID(dir, name, args...)
	return err
}

func startWithPID(dir string, name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if strings.EqualFold(filepath.Base(name), "cmd.exe") {
		hideConsoleWindow(cmd)
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	return pid, nil
}

func (osProcessStarter) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Runner handles pre-flight .env checks, launching Web/Desktop apps, and opening OS tools.
type Runner struct {
	repo    ProjectStore
	starter ProcessStarter
	monitor *ProcessMonitor
}

// NewRunner creates a Runner with the real OS process starter.
func NewRunner(repo ProjectStore) *Runner {
	return &Runner{
		repo:    repo,
		starter: osProcessStarter{},
	}
}

// SetMonitor attaches a ProcessMonitor for tracking launched PIDs and listening ports.
func (r *Runner) SetMonitor(m *ProcessMonitor) {
	r.monitor = m
}

// NewRunnerWithStarter creates a Runner with a custom ProcessStarter (for unit tests).
func NewRunnerWithStarter(repo ProjectStore, starter ProcessStarter) *Runner {
	return &Runner{
		repo:    repo,
		starter: starter,
	}
}

// LaunchProject runs pre-flight checks (for web apps) and starts the project's .bat or .exe.
func (r *Runner) LaunchProject(projectID string) (model.LaunchResult, error) {
	p, err := r.repo.GetProject(projectID)
	if err != nil {
		return model.LaunchResult{}, err
	}

	if strings.TrimSpace(p.LocalDir) == "" || !fsutil.DirExists(p.LocalDir) {
		if p.Status != config.StatusCloudOnly {
			_ = r.repo.UpdateLocalStatus(projectID, config.StatusMissingDir)
		}
		return model.LaunchResult{}, fmt.Errorf("project directory does not exist locally: %s", p.LocalDir)
	}

	if p.Type == config.ProjectTypeWeb {
		return r.launchWeb(p)
	}
	return r.launchDesktop(p)
}

func (r *Runner) launchWeb(p model.ProjectView) (model.LaunchResult, error) {
	envPath := filepath.Join(p.LocalDir, config.EnvFile)
	examplePath := filepath.Join(p.LocalDir, config.EnvExampleFile)

	// Pre-flight guard: if .env is missing but .env.example exists, copy and open .env.
	if !fsutil.FileExists(envPath) && fsutil.FileExists(examplePath) {
		raw, err := os.ReadFile(examplePath)
		if err != nil {
			return model.LaunchResult{}, fmt.Errorf("read %s: %w", config.EnvExampleFile, err)
		}
		if err := os.WriteFile(envPath, raw, 0o644); err != nil {
			return model.LaunchResult{}, fmt.Errorf("create %s: %w", config.EnvFile, err)
		}
		_ = r.repo.UpdateLocalStatus(p.ID, config.StatusMissingEnv)
		if refresher, ok := r.repo.(interface{ RefreshLocalFlags(string) error }); ok {
			_ = refresher.RefreshLocalFlags(p.ID)
		}
		_ = r.starter.Start(p.LocalDir, "notepad.exe", envPath)

		return model.LaunchResult{
			ProjectID:  p.ID,
			Status:     config.LaunchStatusEnvCreatedNeedsConfig,
			Message:    "Đã tạo file .env từ .env.example. Vui lòng điền biến môi trường rồi bấm Run lại.",
			OpenedFile: envPath,
		}, nil
	}

	if p.Status != config.StatusReady {
		_ = r.repo.UpdateLocalStatus(p.ID, config.StatusReady)
	}

	batName, batPath := resolveOrGenerateWebBat(p.LocalDir, p.BatFile, p.Name)
	if batPath == "" || !fsutil.FileExists(batPath) {
		return model.LaunchResult{
			ProjectID: p.ID,
			Status:    config.LaunchStatusMissingBat,
			Message:   fmt.Sprintf("Không tìm thấy file %s trong thư mục dự án.", batName),
		}, nil
	}

	batPath, batErr := fsutil.ResolveSafeSubpath(p.LocalDir, batName)
	if batErr != nil {
		return model.LaunchResult{}, fmt.Errorf("launch %s: %w", batName, batErr)
	}
	if !fsutil.FileExists(batPath) {
		return model.LaunchResult{
			ProjectID: p.ID,
			Status:    config.LaunchStatusMissingBat,
			Message:   fmt.Sprintf("Không tìm thấy file %s trong thư mục dự án.", batName),
		}, nil
	}
	winTitle := "T-Deck - " + sanitizeConsoleTitle(p.Name)
	if pid, err := r.startConsoleTracked(p.LocalDir, batPath, winTitle); err != nil {
		return model.LaunchResult{}, fmt.Errorf("launch %s: %w", batName, err)
	} else if pid > 0 && r.monitor != nil {
		r.monitor.Track(p.ID, pid, batName)
	}
	if refresher, ok := r.repo.(interface{ RefreshLocalFlags(string) error }); ok {
		_ = refresher.RefreshLocalFlags(p.ID)
	}
	_ = r.repo.TouchLastOpened(p.ID)

	return model.LaunchResult{
		ProjectID:  p.ID,
		Status:     config.LaunchStatusLaunched,
		Message:    fmt.Sprintf("Đã khởi chạy %s", batName),
		OpenedFile: batPath,
	}, nil
}

func (r *Runner) launchDesktop(p model.ProjectView) (model.LaunchResult, error) {
	exePath := resolveDesktopExePath(p.LocalDir, p.ExeName)
	if exePath == "" {
		// Fallback: if desktop project has a startup .bat script (e.g. start-studio-app.bat), run it.
		if batName := findExistingBat(p.LocalDir); batName != "" {
			batPath, batErr := fsutil.ResolveSafeSubpath(p.LocalDir, batName)
			if batErr != nil {
				return model.LaunchResult{}, fmt.Errorf("launch %s: %w", batName, batErr)
			}
			winTitle := "T-Deck - " + sanitizeConsoleTitle(p.Name)
			if pid, err := r.startConsoleTracked(p.LocalDir, batPath, winTitle); err != nil {
				return model.LaunchResult{}, fmt.Errorf("launch %s: %w", batName, err)
			} else if pid > 0 && r.monitor != nil {
				r.monitor.Track(p.ID, pid, batName)
			}
			_ = r.repo.TouchLastOpened(p.ID)
			return model.LaunchResult{
				ProjectID:  p.ID,
				Status:     config.LaunchStatusLaunched,
				Message:    fmt.Sprintf("Đã khởi chạy %s", batName),
				OpenedFile: batPath,
			}, nil
		}

		return model.LaunchResult{
			ProjectID: p.ID,
			Status:    config.LaunchStatusMissingExe,
			Message:   "Không tìm thấy file .exe thực thi trong thư mục dự án.",
		}, nil
	}

	exeDir := filepath.Dir(exePath)
	// Launch the desktop executable directly first so Windows never flashes an intermediate
	// black cmd.exe window; fall back to hidden-window Windows Shell start only if elevation (UAC) is needed.
	if pid, err := r.startTracked(exeDir, exePath); err != nil {
		if pidShell, shellErr := r.startTracked(exeDir, "cmd.exe", "/C", "start", "", exePath); shellErr != nil {
			return model.LaunchResult{}, fmt.Errorf("launch %s: %w", filepath.Base(exePath), shellErr)
		} else if pidShell > 0 && r.monitor != nil {
			r.monitor.Track(p.ID, pidShell, filepath.Base(exePath))
		}
	} else if pid > 0 && r.monitor != nil {
		r.monitor.Track(p.ID, pid, filepath.Base(exePath))
	}
	_ = r.repo.TouchLastOpened(p.ID)

	return model.LaunchResult{
		ProjectID:  p.ID,
		Status:     config.LaunchStatusLaunched,
		Message:    fmt.Sprintf("Đã khởi chạy %s", filepath.Base(exePath)),
		OpenedFile: exePath,
	}, nil
}

func (r *Runner) startTracked(dir string, name string, args ...string) (int, error) {
	if _, isOS := r.starter.(osProcessStarter); isOS {
		return startWithPID(dir, name, args...)
	}
	return 0, r.starter.Start(dir, name, args...)
}

func (r *Runner) startConsoleTracked(dir, batPath, winTitle string) (int, error) {
	batBase := filepath.Base(batPath)
	if strings.ContainsAny(batBase, "&|<>^\"\r\n") {
		return 0, fmt.Errorf("tên script chứa ký tự không hợp lệ: %s", batBase)
	}
	workDir := filepath.Dir(batPath)
	if _, isOS := r.starter.(osProcessStarter); !isOS {
		return 0, r.starter.Start(dir, "cmd.exe", "/C", "start", winTitle, "/D", workDir, "cmd.exe", "/K", batBase)
	}
	cmd := exec.Command("cmd.exe", "/K", batBase)
	cmd.Dir = workDir
	openNewConsoleWindow(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	if cmd.Process != nil {
		return cmd.Process.Pid, nil
	}
	return 0, nil
}

func resolveOrGenerateWebBat(localDir, configuredBat, projectName string) (string, string) {
	batName := strings.TrimSpace(configuredBat)
	if batName == "" {
		batName = config.DefaultBatFile
	}
	batPath := filepath.Join(localDir, batName)
	if fsutil.FileExists(batPath) {
		return batName, batPath
	}

	if existing := findExistingBat(localDir); existing != "" {
		return existing, filepath.Join(localDir, existing)
	}

	// Auto-generate start.bat when the project has recognizable markers (package.json, go.mod, etc.)
	if hasBuildableMarker(localDir) {
		if createdPath, err := AutoCreateStartBat(localDir, batName, projectName); err == nil {
			return filepath.Base(createdPath), createdPath
		}
	}
	return batName, ""
}

func hasBuildableMarker(dir string) bool {
	for _, m := range []string{"package.json", "go.mod", "pom.xml", "requirements.txt", "docker-compose.yml"} {
		if fsutil.FileExists(filepath.Join(dir, m)) {
			return true
		}
	}
	return false
}

func findExistingBat(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var fallback string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".bat") || lower == "gradlew.bat" || strings.HasPrefix(lower, "build") || strings.HasPrefix(lower, "tao-phim-tat") || strings.HasPrefix(lower, "get_cookie") {
			continue
		}
		if lower == strings.ToLower(config.DefaultBatFile) || strings.HasPrefix(lower, "start") || strings.HasPrefix(lower, "run") {
			return name
		}
		if fallback == "" {
			fallback = name
		}
	}
	return fallback
}

func resolveDesktopExePath(localDir, exeName string) string {
	trimmed := strings.TrimSpace(exeName)
	if trimmed != "" {
		if filepath.IsAbs(trimmed) && fsutil.FileExists(trimmed) {
			return trimmed
		}
		if direct, err := fsutil.ResolveSafeSubpath(localDir, trimmed); err == nil && fsutil.FileExists(direct) {
			return direct
		}
	}

	targetBase := strings.ToLower(filepath.Base(trimmed))
	searchDirs := []string{
		localDir,
		filepath.Join(localDir, "build", "bin"),
		filepath.Join(localDir, "bin"),
		filepath.Join(localDir, "VNBot_Clean", "dist", "win-unpacked"),
		filepath.Join(localDir, "VNBot_Clean", "dist"),
		filepath.Join(localDir, "dist", "win-unpacked"),
		filepath.Join(localDir, "dist"),
		filepath.Join(localDir, "release"),
		filepath.Join(localDir, "desktop-app", "installer_output"),
	}

	var firstUnpacked, firstAny string
	for _, sub := range searchDirs {
		if !fsutil.DirExists(sub) {
			continue
		}
		_ = filepath.WalkDir(sub, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				if d != nil && d.IsDir() && path != sub {
					lowerDir := strings.ToLower(d.Name())
					if lowerDir == "node_modules" || lowerDir == ".git" || lowerDir == ".venv" || lowerDir == "venv" {
						return filepath.SkipDir
					}
				}
				return nil
			}
			lower := strings.ToLower(d.Name())
			if !isRunnableDesktopExe(lower) {
				return nil
			}
			if targetBase != "" && lower == targetBase {
				firstUnpacked = path
				return filepath.SkipAll
			}
			if !strings.Contains(lower, "setup") && firstUnpacked == "" {
				firstUnpacked = path
			}
			if firstAny == "" {
				firstAny = path
			}
			return nil
		})
		if firstUnpacked != "" && targetBase != "" && strings.EqualFold(filepath.Base(firstUnpacked), targetBase) {
			return firstUnpacked
		}
	}
	if firstUnpacked != "" {
		return firstUnpacked
	}
	return firstAny
}

func isRunnableDesktopExe(lower string) bool {
	if !strings.HasSuffix(lower, ".exe") {
		return false
	}
	for _, skip := range []string{"unins", "elevate.exe", "ffmpeg.exe", "ffprobe.exe", "node.exe", "wails3.exe", ".test.exe"} {
		if strings.HasPrefix(lower, skip) || strings.HasSuffix(lower, skip) || lower == skip {
			return false
		}
	}
	return true
}

func sanitizeConsoleTitle(name string) string {
	cleaned := strings.NewReplacer(`"`, "", "&", "", "|", "", "<", "", ">", "").Replace(strings.TrimSpace(name))
	if cleaned == "" {
		return "App"
	}
	return cleaned
}

// OpenInExplorer opens Windows Explorer at the project's local_dir.
func (r *Runner) OpenInExplorer(projectID string) error {
	p, err := r.repo.GetProject(projectID)
	if err != nil {
		return err
	}
	if !fsutil.DirExists(p.LocalDir) {
		return fmt.Errorf("directory not found: %s", p.LocalDir)
	}
	return r.starter.Start("", "explorer.exe", p.LocalDir)
}

// OpenInEditor opens the project's local_dir in VS Code, IntelliJ, or Windows Terminal.
func (r *Runner) OpenInEditor(projectID string, editor string) error {
	p, err := r.repo.GetProject(projectID)
	if err != nil {
		return err
	}
	if !fsutil.DirExists(p.LocalDir) {
		return fmt.Errorf("directory not found: %s", p.LocalDir)
	}

	switch strings.ToLower(strings.TrimSpace(editor)) {
	case config.EditorVSCode, "":
		if path, err := r.starter.LookPath("code.cmd"); err == nil {
			return r.starter.Start(p.LocalDir, path, p.LocalDir)
		}
		return r.starter.Start(p.LocalDir, "cmd.exe", "/C", "code", p.LocalDir)
	case config.EditorIntelliJ:
		if path, err := r.starter.LookPath("idea64.exe"); err == nil {
			return r.starter.Start(p.LocalDir, path, p.LocalDir)
		}
		return r.starter.Start(p.LocalDir, "cmd.exe", "/C", "idea", p.LocalDir)
	case config.EditorTerminal:
		if path, err := r.starter.LookPath("wt.exe"); err == nil {
			return r.starter.Start(p.LocalDir, path, "-d", p.LocalDir)
		}
		return r.starter.Start(p.LocalDir, "cmd.exe", "/C", "start", "", "cmd.exe", "/K", "cd", "/d", p.LocalDir)
	default:
		return fmt.Errorf("unsupported editor %q", editor)
	}
}

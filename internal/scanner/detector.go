package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
)

type markerDetection struct {
	projectType string
	techStack   string
	batFile     string
	exeName     string
}

var nestedStackSubdirs = []string{
	"frontend",
	"client",
	"web",
	"ui",
	"app",
	"backend",
	"server",
	"api",
	"desktop",
	"desktop-app",
	"src-tauri",
}

func detectProjectMarkers(dir string, isAppsRoot bool) (bool, markerDetection) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, markerDetection{}
	}

	var (
		hasPackageJSON, hasGoMod, hasPomXML, hasRequirements, hasPyProject bool
		hasCargo, hasComposer, hasWails, hasStartBat, hasGradle           bool
		hasPubspec, hasDocker, hasTauri                                   bool
		csprojFile                                                        string
		firstBat, firstExe                                                string
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		switch lower {
		case "package.json":
			hasPackageJSON = true
		case "go.mod":
			hasGoMod = true
		case "pom.xml":
			hasPomXML = true
		case "requirements.txt":
			hasRequirements = true
		case "pyproject.toml":
			hasPyProject = true
		case "cargo.toml":
			hasCargo = true
		case "composer.json":
			hasComposer = true
		case "gradlew.bat", "build.gradle", "build.gradle.kts":
			hasGradle = true
		case "pubspec.yaml":
			hasPubspec = true
		case "dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
			hasDocker = true
		case "tauri.conf.json":
			hasTauri = true
		case strings.ToLower(config.WailsConfig):
			hasWails = true
		case strings.ToLower(config.DefaultBatFile):
			hasStartBat = true
			firstBat = name
		}
		if strings.HasSuffix(lower, ".csproj") && csprojFile == "" {
			csprojFile = name
		}
		if strings.HasSuffix(lower, ".bat") && lower != "gradlew.bat" && firstBat == "" {
			firstBat = name
		}
		if isAppExecutable(lower) && firstExe == "" {
			firstExe = name
		}
	}

	if !hasTauri && fsutil.FileExists(filepath.Join(dir, "src-tauri", "tauri.conf.json")) {
		hasTauri = true
	}

	hasCodeMarker := hasPackageJSON || hasGoMod || hasPomXML || hasRequirements || hasPyProject ||
		hasCargo || hasComposer || hasGradle || hasWails || hasStartBat || hasPubspec ||
		hasTauri || csprojFile != "" || firstBat != ""
	hasAnyMarker := hasCodeMarker || firstExe != ""
	if !hasAnyMarker {
		return false, markerDetection{}
	}

	var stacks []string
	isPythonDesktop := false
	isElectronDesktop := false
	isTauriDesktop := hasTauri
	isDotNetDesktop := false
	isGoWails := hasWails

	if hasGoMod {
		goStacks, wailsInMod := detectGoStack(filepath.Join(dir, "go.mod"))
		stacks = append(stacks, goStacks...)
		if wailsInMod {
			isGoWails = true
		}
	}
	if isGoWails {
		stacks = append(stacks, "Wails")
	}
	if hasPackageJSON {
		nodeStacks, isElectron, isTauriPkg := detectNodeStack(filepath.Join(dir, "package.json"))
		stacks = append(stacks, nodeStacks...)
		if isElectron {
			isElectronDesktop = true
		}
		if isTauriPkg {
			isTauriDesktop = true
		}
	}
	if hasPomXML {
		stacks = append(stacks, detectJavaStack(filepath.Join(dir, "pom.xml"), "Maven")...)
	}
	if hasGradle {
		stacks = append(stacks, detectGradleStack(dir)...)
	}
	if hasCargo {
		rustStacks, tauriCargo := detectRustStack(filepath.Join(dir, "Cargo.toml"))
		stacks = append(stacks, rustStacks...)
		if tauriCargo {
			isTauriDesktop = true
		}
	}
	if hasTauri {
		stacks = append(stacks, "Tauri")
	}
	if hasComposer {
		stacks = append(stacks, "PHP")
	}
	if hasPubspec {
		stacks = append(stacks, detectFlutterStack(filepath.Join(dir, "pubspec.yaml"))...)
	}
	if csprojFile != "" {
		csStacks, winApp := detectDotNetStack(filepath.Join(dir, csprojFile))
		stacks = append(stacks, csStacks...)
		if winApp {
			isDotNetDesktop = true
		}
	}
	if hasRequirements || hasPyProject {
		pyStacks, pyDesk := detectPythonStack(dir)
		stacks = append(stacks, pyStacks...)
		if pyDesk {
			isPythonDesktop = true
		}
	}

	// Inspect standard immediate subdirs (e.g. frontend/, backend/, src-tauri/) so monorepo / Wails+React projects
	// capture both backend and frontend stacks instead of only root markers.
	for _, sub := range nestedStackSubdirs {
		subDir := filepath.Join(dir, sub)
		if !fsutil.DirExists(subDir) {
			continue
		}
		if pkg := filepath.Join(subDir, "package.json"); fsutil.FileExists(pkg) {
			nodeStacks, isElectron, isTauriPkg := detectNodeStack(pkg)
			stacks = append(stacks, nodeStacks...)
			if isElectron {
				isElectronDesktop = true
			}
			if isTauriPkg {
				isTauriDesktop = true
			}
		}
		if gm := filepath.Join(subDir, "go.mod"); fsutil.FileExists(gm) {
			goStacks, wailsInMod := detectGoStack(gm)
			stacks = append(stacks, goStacks...)
			if wailsInMod {
				isGoWails = true
			}
		}
		if fsutil.FileExists(filepath.Join(subDir, "requirements.txt")) || fsutil.FileExists(filepath.Join(subDir, "pyproject.toml")) {
			pyStacks, pyDesk := detectPythonStack(subDir)
			stacks = append(stacks, pyStacks...)
			if pyDesk {
				isPythonDesktop = true
			}
		}
		if pom := filepath.Join(subDir, "pom.xml"); fsutil.FileExists(pom) {
			stacks = append(stacks, detectJavaStack(pom, "Maven")...)
		}
		if cargo := filepath.Join(subDir, "Cargo.toml"); fsutil.FileExists(cargo) {
			rustStacks, tauriCargo := detectRustStack(cargo)
			stacks = append(stacks, rustStacks...)
			if tauriCargo {
				isTauriDesktop = true
			}
		}
	}

	if hasDocker {
		stacks = append(stacks, "Docker")
	}

	if firstExe == "" {
		firstExe = findBuiltExe(dir)
	}

	pType := config.ProjectTypeWeb
	switch {
	case isAppsRoot && firstExe != "":
		pType = config.ProjectTypeDesktop
	case isGoWails || isPythonDesktop || isElectronDesktop || isTauriDesktop || isDotNetDesktop:
		pType = config.ProjectTypeDesktop
	case hasCodeMarker:
		pType = config.ProjectTypeWeb
	case firstExe != "":
		pType = config.ProjectTypeDesktop
	}

	batFile := firstBat
	if batFile == "" {
		if hasGradle {
			batFile = "gradlew.bat"
		} else {
			batFile = config.DefaultBatFile
		}
	}

	return true, markerDetection{
		projectType: pType,
		techStack:   strings.Join(fsutil.UniqueStrings(stacks), ", "),
		batFile:     batFile,
		exeName:     firstExe,
	}
}

func detectGoStack(goModPath string) ([]string, bool) {
	stacks := []string{"Go"}
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return stacks, false
	}
	lower := strings.ToLower(string(content))
	isWails := strings.Contains(lower, "wails")
	if isWails {
		stacks = append(stacks, "Wails")
	}
	if strings.Contains(lower, "gin-gonic/gin") {
		stacks = append(stacks, "Gin")
	}
	if strings.Contains(lower, "gofiber/fiber") {
		stacks = append(stacks, "Fiber")
	}
	if strings.Contains(lower, "labstack/echo") {
		stacks = append(stacks, "Echo")
	}
	if strings.Contains(lower, "sqlite") {
		stacks = append(stacks, "SQLite")
	}
	return stacks, isWails
}

func detectJavaStack(pomPath string, tool string) []string {
	stacks := []string{"Java", tool}
	if content, err := os.ReadFile(pomPath); err == nil {
		if strings.Contains(strings.ToLower(string(content)), "spring-boot") {
			stacks = append(stacks, "Spring Boot")
		}
	}
	return stacks
}

func detectGradleStack(dir string) []string {
	stacks := []string{"Android", "Gradle"}
	for _, gf := range []string{"build.gradle", "build.gradle.kts"} {
		if content, err := os.ReadFile(filepath.Join(dir, gf)); err == nil {
			lower := strings.ToLower(string(content))
			if strings.Contains(lower, "spring-boot") {
				stacks = []string{"Java", "Spring Boot", "Gradle"}
				break
			}
			if strings.Contains(lower, "kotlin") {
				stacks = append(stacks, "Kotlin")
			}
		}
	}
	return stacks
}

func detectRustStack(cargoPath string) ([]string, bool) {
	stacks := []string{"Rust"}
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return stacks, false
	}
	lower := strings.ToLower(string(content))
	isTauri := strings.Contains(lower, "tauri")
	if isTauri {
		stacks = append(stacks, "Tauri")
	}
	return stacks, isTauri
}

func detectFlutterStack(pubspecPath string) []string {
	stacks := []string{"Dart"}
	if content, err := os.ReadFile(pubspecPath); err == nil {
		if strings.Contains(strings.ToLower(string(content)), "flutter") {
			stacks = append([]string{"Flutter"}, stacks...)
		}
	}
	return stacks
}

func detectDotNetStack(csprojPath string) ([]string, bool) {
	stacks := []string{".NET", "C#"}
	content, err := os.ReadFile(csprojPath)
	if err != nil {
		return stacks, false
	}
	lower := strings.ToLower(string(content))
	isWinApp := strings.Contains(lower, "usewpf") ||
		strings.Contains(lower, "usewindowsforms") ||
		strings.Contains(lower, "<outputtype>winexe</outputtype>")
	return stacks, isWinApp
}

func detectPythonStack(dir string) ([]string, bool) {
	stacks := []string{"Python"}
	isDesktop := false
	for _, pyFile := range []string{"requirements.txt", "pyproject.toml"} {
		content, err := os.ReadFile(filepath.Join(dir, pyFile))
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(content))
		if strings.Contains(lower, "pyside") || strings.Contains(lower, "pyqt") ||
			strings.Contains(lower, "customtkinter") || strings.Contains(lower, "wxpython") ||
			strings.Contains(lower, "pyinstaller") {
			isDesktop = true
		}
		if strings.Contains(lower, "fastapi") {
			stacks = append(stacks, "FastAPI")
		}
		if strings.Contains(lower, "django") {
			stacks = append(stacks, "Django")
		}
		if strings.Contains(lower, "flask") {
			stacks = append(stacks, "Flask")
		}
		if strings.Contains(lower, "streamlit") {
			stacks = append(stacks, "Streamlit")
		}
		if strings.Contains(lower, "gradio") {
			stacks = append(stacks, "Gradio")
		}
	}
	return stacks, isDesktop
}

func findBuiltExe(dir string) string {
	candidates := []string{
		filepath.Join(dir, "build", "bin"),
		filepath.Join(dir, "bin"),
		filepath.Join(dir, "dist"),
		filepath.Join(dir, "release"),
		filepath.Join(dir, "desktop-app", "installer_output"),
		filepath.Join(dir, "src-tauri", "target", "release"),
	}
	for _, sub := range candidates {
		var match string
		_ = filepath.WalkDir(sub, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if isAppExecutable(strings.ToLower(d.Name())) {
				if rel, err := filepath.Rel(dir, path); err == nil {
					match = rel
					return filepath.SkipAll
				}
			}
			return nil
		})
		if match != "" {
			return match
		}
	}
	return ""
}

func isAppExecutable(lowerName string) bool {
	if !strings.HasSuffix(lowerName, ".exe") {
		return false
	}
	if strings.HasPrefix(lowerName, "unins") ||
		strings.HasPrefix(lowerName, "uninstall") ||
		lowerName == "ffmpeg.exe" ||
		lowerName == "ffprobe.exe" ||
		lowerName == "node.exe" ||
		lowerName == "wails3.exe" ||
		lowerName == "cloudflared.exe" ||
		lowerName == "protoc.exe" ||
		lowerName == "chromedriver.exe" ||
		lowerName == "rustup-init.exe" ||
		lowerName == "vs_buildtools.exe" ||
		lowerName == "service.exe" ||
		lowerName == "defaults.exe" ||
		lowerName == "tdeck-sync.exe" ||
		strings.HasSuffix(lowerName, ".test.exe") {
		return false
	}
	return true
}

func detectNodeStack(pkgPath string) ([]string, bool, bool) {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return []string{"Node.js"}, false, false
	}
	lower := strings.ToLower(string(data))
	stacks := []string{"Node.js"}
	isElectron := strings.Contains(lower, `"electron"`)
	isTauri := strings.Contains(lower, `"@tauri-apps/`)
	if isElectron {
		stacks = append(stacks, "Electron")
	}
	if isTauri {
		stacks = append(stacks, "Tauri")
	}
	if strings.Contains(lower, `"react"`) {
		stacks = append(stacks, "React")
	}
	if strings.Contains(lower, `"next"`) {
		stacks = append(stacks, "Next.js")
	}
	if strings.Contains(lower, `"vite"`) {
		stacks = append(stacks, "Vite")
	}
	if strings.Contains(lower, `"vue"`) {
		stacks = append(stacks, "Vue")
	}
	if strings.Contains(lower, `"nuxt"`) {
		stacks = append(stacks, "Nuxt")
	}
	if strings.Contains(lower, `"svelte"`) {
		stacks = append(stacks, "Svelte")
	}
	if strings.Contains(lower, `"@angular/core"`) {
		stacks = append(stacks, "Angular")
	}
	if strings.Contains(lower, `"typescript"`) {
		stacks = append(stacks, "TypeScript")
	}
	if strings.Contains(lower, `"tailwindcss"`) {
		stacks = append(stacks, "Tailwind CSS")
	}
	if strings.Contains(lower, `"express"`) {
		stacks = append(stacks, "Express")
	}
	if strings.Contains(lower, `"@nestjs/core"`) {
		stacks = append(stacks, "NestJS")
	}
	if strings.Contains(lower, `"fastify"`) {
		stacks = append(stacks, "Fastify")
	}
	return stacks, isElectron, isTauri
}

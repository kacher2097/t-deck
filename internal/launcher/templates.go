package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
)

// BatTemplate describes a selectable start.bat preset.
type BatTemplate struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Command string `json:"command"`
}

// AvailableBatTemplates returns the 5 standard start.bat templates from the SRS.
func AvailableBatTemplates() []BatTemplate {
	return []BatTemplate{
		{ID: "node", Label: "Node / Vite (npm run dev)", Command: "npm run dev"},
		{ID: "go", Label: "Go (go run .)", Command: "go run ."},
		{ID: "maven", Label: "Java Maven (mvn spring-boot:run)", Command: "mvn spring-boot:run"},
		{ID: "python", Label: "Python (python main.py)", Command: "python main.py"},
		{ID: "docker", Label: "Docker Compose (docker-compose up -d)", Command: "docker-compose up -d"},
	}
}

// RenderBatScript builds Windows batch script content for the chosen template ID.
func RenderBatScript(projectName, templateID string) (string, error) {
	var selected *BatTemplate
	for _, tpl := range AvailableBatTemplates() {
		if strings.EqualFold(tpl.ID, strings.TrimSpace(templateID)) {
			t := tpl
			selected = &t
			break
		}
	}
	if selected == nil {
		return "", fmt.Errorf("unknown start.bat template %q", templateID)
	}
	return renderCustomBatCommand(projectName, selected.Command), nil
}

func renderCustomBatCommand(projectName, command string) string {
	title := strings.NewReplacer("&", "", "|", "", "<", "", ">", "", `"`, "").Replace(projectName)
	if strings.TrimSpace(title) == "" {
		title = "T-Deck App"
	}
	lines := []string{
		"@echo off",
		"cd /d \"%~dp0\"",
		fmt.Sprintf("title %s", title),
		command,
		"if errorlevel 1 pause",
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}

// WriteStartBat writes a start.bat file using the given template into projectDir.
func WriteStartBat(projectDir, batFile, projectName, templateID string) (string, error) {
	if strings.TrimSpace(projectDir) == "" {
		return "", fmt.Errorf("project directory is empty")
	}
	targetName := sanitizeBatFileName(batFile)

	content, err := RenderBatScript(projectName, templateID)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(projectDir, targetName)
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", targetName, err)
	}
	return targetPath, nil
}

// AutoCreateStartBat inspects projectDir markers and generates a working start.bat automatically.
func AutoCreateStartBat(projectDir, batFile, projectName string) (string, error) {
	if strings.TrimSpace(projectDir) == "" {
		return "", fmt.Errorf("project directory is empty")
	}
	targetName := sanitizeBatFileName(batFile)
	command := inferStartCommand(projectDir)
	content := renderCustomBatCommand(projectName, command)
	targetPath := filepath.Join(projectDir, targetName)
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("auto-create %s: %w", targetName, err)
	}
	return targetPath, nil
}

func inferStartCommand(projectDir string) string {
	pkgPath := filepath.Join(projectDir, "package.json")
	if fsutil.FileExists(pkgPath) {
		if data, err := os.ReadFile(pkgPath); err == nil {
			var parsed struct {
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &parsed) == nil && len(parsed.Scripts) > 0 {
				for _, candidate := range []string{"dev", "start", "studio", "dev:web", "dev:dashboard", "serve", "preview"} {
					if _, ok := parsed.Scripts[candidate]; ok {
						if candidate == "start" {
							return "npm start"
						}
						return "npm run " + candidate
					}
				}
			}
		}
		return "npm run dev"
	}
	if fsutil.FileExists(filepath.Join(projectDir, "go.mod")) {
		return "go run ."
	}
	if fsutil.FileExists(filepath.Join(projectDir, "pom.xml")) {
		return "mvn spring-boot:run"
	}
	if fsutil.FileExists(filepath.Join(projectDir, "requirements.txt")) {
		for _, pyEntry := range []string{"main.py", "app.py", "run.py", "server.py"} {
			if fsutil.FileExists(filepath.Join(projectDir, pyEntry)) {
				return "python " + pyEntry
			}
		}
		return "python main.py"
	}
	if fsutil.FileExists(filepath.Join(projectDir, "docker-compose.yml")) {
		return "docker-compose up -d"
	}
	return "npm run dev"
}

func sanitizeBatFileName(batFile string) string {
	targetName := strings.TrimSpace(batFile)
	if targetName == "" {
		return config.DefaultBatFile
	}
	cleanRel := filepath.Base(targetName)
	if cleanRel == "." || cleanRel == ".." {
		return config.DefaultBatFile
	}
	return cleanRel
}

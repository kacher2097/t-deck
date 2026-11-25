package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t-deck/internal/config"
)

func TestGenerateEnvExampleStripsSecrets(t *testing.T) {
	dir := t.TempDir()
	envContent := "# Database config\nDB_HOST=localhost\nDB_PASS=super_secret=123\n\nAPI_TOKEN = abc-xyz\n# End\n"
	if err := os.WriteFile(filepath.Join(dir, config.EnvFile), []byte(envContent), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := GenerateEnvExample("proj-1", dir)
	if err != nil {
		t.Fatalf("GenerateEnvExample failed: %v", err)
	}
	if res.VarsCount != 3 {
		t.Fatalf("expected 3 vars, got %d", res.VarsCount)
	}

	outBytes, err := os.ReadFile(filepath.Join(dir, config.EnvExampleFile))
	if err != nil {
		t.Fatal(err)
	}
	out := string(outBytes)

	if strings.Contains(out, "super_secret") || strings.Contains(out, "abc-xyz") || strings.Contains(out, "localhost") {
		t.Fatalf("secret leaked into .env.example:\n%s", out)
	}
	expected := "# Database config\nDB_HOST=\nDB_PASS=\n\nAPI_TOKEN=\n# End\n"
	if out != expected {
		t.Fatalf("unexpected .env.example:\ngot:\n%q\nwant:\n%q", out, expected)
	}
}


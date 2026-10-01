package config

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// runInitEnv executes the app:init-env script in dir and returns the
// resulting .env contents.
func runInitEnv(t *testing.T, dir string) []byte {
	t.Helper()
	script, err := filepath.Abs("../../resources/scripts/init-env.sh")
	if err != nil {
		t.Fatalf("resolve init-env script: %v", err)
	}
	command := exec.Command("sh", script)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("init-env failed: %v\n%s", err, output)
	}
	contents, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("read generated .env: %v", err)
	}
	return contents
}

func newInitEnvDir(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}
	example, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), example, 0o644); err != nil {
		t.Fatalf("write .env.example: %v", err)
	}
	return dir
}

func TestInitEnvProducesStartableServerConfiguration(t *testing.T) {
	dir := newInitEnvDir(t)
	values, err := godotenv.Unmarshal(string(runInitEnv(t, dir)))
	if err != nil {
		t.Fatalf("parse generated .env: %v", err)
	}

	server, err := LoadFrom(lookup(values)).Server()
	if err != nil {
		t.Fatalf("generated .env does not load: %v", err)
	}
	if len(server.Postcards.TokenKeyring().keys) != 1 {
		t.Fatalf("expected one generated postcard token key")
	}

	other, err := godotenv.Unmarshal(string(runInitEnv(t, newInitEnvDir(t))))
	if err != nil {
		t.Fatalf("parse second generated .env: %v", err)
	}
	if other["WGA_POSTCARD_TOKEN_KEYS"] == values["WGA_POSTCARD_TOKEN_KEYS"] {
		t.Fatalf("expected each generated .env to receive a fresh postcard token key")
	}
}

func TestInitEnvKeepsExistingEnvFile(t *testing.T) {
	dir := newInitEnvDir(t)
	existing := []byte("WGA_ENV=development\n# local edits\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), existing, 0o600); err != nil {
		t.Fatalf("write existing .env: %v", err)
	}

	if got := runInitEnv(t, dir); !bytes.Equal(got, existing) {
		t.Fatalf("expected existing .env to be unchanged, got %q", got)
	}
}

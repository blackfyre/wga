package config

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

// runInitEnv executes the app:init-env script in dir and returns the
// resulting .env contents.
func runInitEnv(t *testing.T, dir string) []byte {
	t.Helper()
	contents, _ := runInitEnvWithStderr(t, dir)
	return contents
}

// runInitEnvWithStderr also returns what the script wrote to stderr.
func runInitEnvWithStderr(t *testing.T, dir string) ([]byte, string) {
	t.Helper()
	script, err := filepath.Abs("../../resources/scripts/init-env.sh")
	if err != nil {
		t.Fatalf("resolve init-env script: %v", err)
	}
	var stderr bytes.Buffer
	command := exec.Command("sh", script)
	command.Dir = dir
	command.Stderr = &stderr
	if output, err := command.Output(); err != nil {
		t.Fatalf("init-env failed: %v\n%s%s", err, output, stderr.String())
	}
	contents, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("read generated .env: %v", err)
	}
	return contents, stderr.String()
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
	tests := []struct {
		name     string
		existing string
		warns    bool
	}{
		{name: "keyring absent", existing: "WGA_ENV=development\n# local edits\n", warns: true},
		{name: "keyring empty", existing: "WGA_POSTCARD_TOKEN_KEYS=\nWGA_POSTCARD_TOKEN_ACTIVE_KEY_ID=\n", warns: true},
		{name: "keyring quoted empty", existing: "WGA_POSTCARD_TOKEN_KEYS=''\nWGA_POSTCARD_TOKEN_ACTIVE_KEY_ID=\"\"\n", warns: true},
		{name: "keyring whitespace", existing: "WGA_POSTCARD_TOKEN_KEYS= \nWGA_POSTCARD_TOKEN_ACTIVE_KEY_ID=\" \"\n", warns: true},
		{name: "active key ID missing", existing: "WGA_POSTCARD_TOKEN_KEYS='{\"dev\":\"key\"}'\n", warns: true},
		{name: "keyring set", existing: "WGA_POSTCARD_TOKEN_KEYS='{\"dev\":\"key\"}'\nexport WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID = dev\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := newInitEnvDir(t)
			existing := []byte(test.existing)
			if err := os.WriteFile(filepath.Join(dir, ".env"), existing, 0o600); err != nil {
				t.Fatalf("write existing .env: %v", err)
			}

			got, stderr := runInitEnvWithStderr(t, dir)
			if !bytes.Equal(got, existing) {
				t.Fatalf("expected existing .env to be unchanged, got %q", got)
			}
			if warned := strings.Contains(stderr, "WGA_POSTCARD_TOKEN_KEYS"); warned != test.warns {
				t.Fatalf("expected warning=%t, got stderr %q", test.warns, stderr)
			}
		})
	}
}

func TestInitEnvAcceptsExistingEnvFileWithKeyring(t *testing.T) {
	dir := newInitEnvDir(t)
	generated := runInitEnv(t, dir)

	got, stderr := runInitEnvWithStderr(t, dir)
	if !bytes.Equal(got, generated) {
		t.Fatalf("expected generated .env to be unchanged on re-run")
	}
	if stderr != "" {
		t.Fatalf("expected no warning for a complete .env, got %q", stderr)
	}
}

package migrations

import (
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase/core"
)

func TestMigrationsKeepExistingSettings(t *testing.T) {
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_PROTOCOL":         "https",
			"WGA_HOSTNAME":         "gallery.example",
			"WGA_S3_ENDPOINT":      "https://storage.example",
			"WGA_S3_BUCKET":        "wga-test",
			"WGA_S3_REGION":        "eu-central-1",
			"WGA_S3_ACCESS_KEY":    "access-key",
			"WGA_S3_ACCESS_SECRET": "access-secret",
			"WGA_SMTP_HOST":        "smtp.example",
			"WGA_SMTP_PORT":        "2525",
			"WGA_SENDER_NAME":      "WGA Test",
			"WGA_SENDER_ADDRESS":   "sender@example.com",
		}[key]
	})
	if err := Configure(configuration.Migrations()); err != nil {
		t.Fatalf("configure migrations: %v", err)
	}

	dataDir := t.TempDir()
	fresh := newMigrationTestApp(t, dataDir)
	if err := fresh.RunAllMigrations(); err != nil {
		t.Fatalf("run fresh migrations: %v", err)
	}

	freshSettings := fresh.Settings()
	if got, want := freshSettings.Meta.AppURL, "https://gallery.example"; got != want {
		t.Fatalf("expected app URL %q, got %q", want, got)
	}
	if got, want := freshSettings.S3.Endpoint, "https://storage.example"; got != want {
		t.Fatalf("expected storage endpoint %q, got %q", want, got)
	}
	if got, want := freshSettings.SMTP.Port, 2525; got != want {
		t.Fatalf("expected SMTP port %d, got %d", want, got)
	}
	superusers, err := fresh.FindRecordsByFilter(core.CollectionNameSuperusers, "", "", 0, 0)
	if err != nil {
		t.Fatalf("find superusers: %v", err)
	}
	if len(superusers) != 0 {
		t.Fatalf("expected no bootstrap administrator, got %d superusers", len(superusers))
	}
	if err := fresh.ResetBootstrapState(); err != nil {
		t.Fatalf("close fresh app: %v", err)
	}

	existing := newMigrationTestApp(t, dataDir)
	defer func() {
		if err := existing.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	if err := existing.RunAllMigrations(); err != nil {
		t.Fatalf("run existing migrations: %v", err)
	}

	existingSettings := existing.Settings()
	if got, want := existingSettings.Meta.AppURL, freshSettings.Meta.AppURL; got != want {
		t.Fatalf("expected existing app URL %q, got %q", want, got)
	}
	if got, want := existingSettings.S3.Endpoint, freshSettings.S3.Endpoint; got != want {
		t.Fatalf("expected existing storage endpoint %q, got %q", want, got)
	}
	if got, want := existingSettings.SMTP.Port, freshSettings.SMTP.Port; got != want {
		t.Fatalf("expected existing SMTP port %d, got %d", want, got)
	}
}

func newMigrationTestApp(t *testing.T, dataDir string) *core.BaseApp {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dataDir,
		EncryptionEnv: "test-encryption-key",
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}

	return app
}

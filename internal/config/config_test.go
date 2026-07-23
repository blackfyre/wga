package config

import (
	"strings"
	"testing"
)

func TestServerCaptchaPolicy(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		secret      string
		siteKey     string
		wantVerify  bool
		wantErr     string
	}{
		{
			name:        "development permits bypass",
			environment: "development",
		},
		{
			name:        "test permits bypass",
			environment: "test",
		},
		{
			name:        "staging requires a secret",
			environment: "staging",
			wantErr:     "WGA_RECAPTCHA_SECRET",
		},
		{
			name:        "staging requires a site key",
			environment: "staging",
			secret:      "captcha-secret",
			wantErr:     "WGA_RECAPTCHA_SITE_KEY",
		},
		{
			name:        "production verifies configured secret",
			environment: "production",
			secret:      "captcha-secret",
			siteKey:     "captcha-site-key",
			wantVerify:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_ENV"] = test.environment
			values["WGA_RECAPTCHA_SECRET"] = test.secret
			values["WGA_RECAPTCHA_SITE_KEY"] = test.siteKey

			server, err := LoadFrom(lookup(values)).Server()
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if server.Captcha.Verify() != test.wantVerify {
				t.Fatalf("expected captcha verification %t", test.wantVerify)
			}
			if got, want := server.Captcha.SiteKey(), test.siteKey; got != want {
				t.Fatalf("expected site key %q, got %q", want, got)
			}
		})
	}
}

func TestConfigurationValidationIsCapabilitySpecific(t *testing.T) {
	values := validValues()
	values["WGA_ENV"] = "production"
	values["WGA_RECAPTCHA_SECRET"] = ""
	values["WGA_SMTP_PORT"] = "not-a-port"
	values["WGA_S3_ENDPOINT"] = "not-a-url"

	configuration := LoadFrom(lookup(values))

	if _, err := configuration.Sitemap(); err != nil {
		t.Fatalf("sitemap should not require mail, storage, or captcha: %v", err)
	}

	if _, err := configuration.Server(); err == nil || !strings.Contains(err.Error(), "WGA_RECAPTCHA_SECRET") {
		t.Fatalf("expected protected server captcha error, got %v", err)
	} else if strings.Contains(err.Error(), "WGA_SMTP_PORT") {
		t.Fatalf("server should not require migration SMTP settings: %v", err)
	}

	if _, err := configuration.Migrations().InitialSettings(); err == nil {
		t.Fatal("expected invalid migration settings")
	} else if !strings.Contains(err.Error(), "WGA_SMTP_PORT") {
		t.Fatalf("expected migration settings errors, got %v", err)
	} else if strings.Contains(err.Error(), "WGA_S3_ENDPOINT") {
		t.Fatalf("migration should ignore invalid storage settings: %v", err)
	}
}

func TestConfigurationParsesTypedValues(t *testing.T) {
	configuration := LoadFrom(lookup(validValues()))

	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("unexpected server configuration error: %v", err)
	}
	if got, want := server.PublicURL.Resolve("postcard?p=abc"), "http://localhost:8090/postcard?p=abc"; got != want {
		t.Fatalf("expected resolved URL %q, got %q", want, got)
	}
	if got, want := server.Postcards.Expression(), "*/5 * * * *"; got != want {
		t.Fatalf("expected postcard schedule %q, got %q", want, got)
	}

	settings, err := configuration.Migrations().InitialSettings()
	if err != nil {
		t.Fatalf("unexpected migration configuration error: %v", err)
	}
	if got, want := settings.Mail.SMTP.Port, 1025; got != want {
		t.Fatalf("expected SMTP port %d, got %d", want, got)
	}
	if !settings.Storage.Enabled {
		t.Fatal("expected valid storage configuration to be enabled")
	}
}

func TestSeedConfigurationUsesConfiguredPaths(t *testing.T) {
	values := validValues()
	values["WGA_SEED_SQLITE_PATH"] = "/data/reference.sqlite"
	values["WGA_SEED_STORAGE_PATH"] = "/data/storage"

	seed := LoadFrom(lookup(values)).Seed()
	if got, want := seed.Environment, EnvironmentDevelopment; got != want {
		t.Fatalf("expected seed environment %q, got %q", want, got)
	}
	if got, want := seed.SQLitePath, "/data/reference.sqlite"; got != want {
		t.Fatalf("expected SQLite path %q, got %q", want, got)
	}
	if got, want := seed.StoragePath, "/data/storage"; got != want {
		t.Fatalf("expected storage path %q, got %q", want, got)
	}
}

func TestServerRejectsInvalidTypedSettings(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "missing environment", key: "WGA_ENV", value: "", want: "WGA_ENV"},
		{name: "environment", key: "WGA_ENV", value: "preview", want: "WGA_ENV"},
		{name: "protocol", key: "WGA_PROTOCOL", value: "ftp", want: "WGA_PROTOCOL"},
		{name: "hostname", key: "WGA_HOSTNAME", value: "https://gallery.example", want: "WGA_HOSTNAME"},
		{name: "hostname port", key: "WGA_HOSTNAME", value: "gallery.example:not-a-port", want: "WGA_HOSTNAME"},
		{name: "postcard schedule", key: "WGA_POSTCARD_FREQUENCY", value: "not a cron expression", want: "WGA_POSTCARD_FREQUENCY"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values[test.key] = test.value

			_, err := LoadFrom(lookup(values)).Server()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestInvalidStorageConfigurationIsDisabled(t *testing.T) {
	values := validValues()
	values["WGA_S3_ENDPOINT"] = "not-a-url"
	values["WGA_S3_ACCESS_SECRET"] = "private-storage-secret"

	settings, err := LoadFrom(lookup(values)).Migrations().InitialSettings()
	if err != nil {
		t.Fatalf("expected invalid storage to be ignored, got %v", err)
	}
	if settings.Storage.Enabled {
		t.Fatal("expected invalid storage configuration to be disabled")
	}

	values = validValues()
	for _, key := range []string{"WGA_S3_ENDPOINT", "WGA_S3_BUCKET", "WGA_S3_ACCESS_KEY", "WGA_S3_ACCESS_SECRET"} {
		values[key] = ""
	}
	settings, err = LoadFrom(lookup(values)).Migrations().InitialSettings()
	if err != nil {
		t.Fatalf("expected omitted storage to be ignored, got %v", err)
	}
	if settings.Storage.Enabled {
		t.Fatal("expected omitted storage configuration to be disabled")
	}
}

func TestMigrationRequiresMailSettings(t *testing.T) {
	values := validValues()
	values["WGA_SMTP_HOST"] = ""
	values["WGA_SMTP_PORT"] = ""
	values["WGA_SENDER_NAME"] = ""
	values["WGA_SENDER_ADDRESS"] = ""

	_, err := LoadFrom(lookup(values)).Migrations().InitialSettings()
	if err == nil {
		t.Fatal("expected missing migration mail settings")
	}
	for _, key := range []string{"WGA_SMTP_HOST", "WGA_SMTP_PORT", "WGA_SENDER_NAME", "WGA_SENDER_ADDRESS"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("expected error to contain %q, got %v", key, err)
		}
	}
}

func TestAdministratorCredentialsMustBePaired(t *testing.T) {
	t.Run("omitted credentials disable bootstrap", func(t *testing.T) {
		values := validValues()
		values["WGA_ADMIN_EMAIL"] = ""
		values["WGA_ADMIN_PASSWORD"] = ""

		administrator, err := LoadFrom(lookup(values)).Migrations().Administrator()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if administrator.Enabled {
			t.Fatal("expected omitted credentials to disable administrator bootstrap")
		}
	})

	t.Run("partial credentials are rejected", func(t *testing.T) {
		values := validValues()
		values["WGA_ADMIN_PASSWORD"] = ""

		_, err := LoadFrom(lookup(values)).Migrations().Administrator()
		if err == nil || !strings.Contains(err.Error(), "WGA_ADMIN_EMAIL and WGA_ADMIN_PASSWORD") {
			t.Fatalf("expected paired administrator credentials error, got %v", err)
		}
	})
}

func validValues() map[string]string {
	return map[string]string{
		"WGA_ENV":                "development",
		"WGA_PROTOCOL":           "http",
		"WGA_HOSTNAME":           "localhost:8090",
		"WGA_S3_ENDPOINT":        "http://localhost:9000",
		"WGA_S3_BUCKET":          "wga",
		"WGA_S3_REGION":          "us-east-1",
		"WGA_S3_ACCESS_KEY":      "access-key",
		"WGA_S3_ACCESS_SECRET":   "access-secret",
		"WGA_SMTP_HOST":          "127.0.0.1",
		"WGA_SMTP_PORT":          "1025",
		"WGA_SMTP_USERNAME":      "",
		"WGA_SMTP_PASSWORD":      "",
		"WGA_SENDER_NAME":        "WGA",
		"WGA_SENDER_ADDRESS":     "do-not-reply@wga.hu",
		"WGA_POSTCARD_FREQUENCY": "*/5 * * * *",
		"WGA_RECAPTCHA_SITE_KEY": "captcha-site-key",
		"WGA_ADMIN_EMAIL":        "admin@wga.hu",
		"WGA_ADMIN_PASSWORD":     "admin-password",
	}
}

func lookup(values map[string]string) Lookup {
	return func(key string) string {
		return values[key]
	}
}

package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestServerSentryConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		dsn        string
		browserDSN string
		wantErr    string
	}{
		{name: "omitted DSN disables monitoring"},
		{name: "configured DSN", dsn: "https://public@example.ingest.sentry.io/1"},
		{name: "configured browser DSN", browserDSN: "https://browser@example.ingest.sentry.io/2"},
		{name: "separate server and browser DSNs", dsn: "https://server@example.ingest.sentry.io/1", browserDSN: "https://browser@example.ingest.sentry.io/2"},
		{name: "malformed DSN", dsn: "not-a-dsn", wantErr: "WGA_SENTRY_DSN"},
		{name: "DSN with secret key", dsn: "https://public:secret@example.ingest.sentry.io/1", wantErr: "WGA_SENTRY_DSN"},
		{name: "browser DSN with secret key", browserDSN: "https://public:secret@example.ingest.sentry.io/1", wantErr: "WGA_SENTRY_BROWSER_DSN"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_SENTRY_DSN"] = test.dsn
			values["WGA_SENTRY_BROWSER_DSN"] = test.browserDSN

			server, err := LoadFrom(lookup(values)).Server()
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected server configuration error: %v", err)
			}
			if got := server.Sentry.DSN(); got != test.dsn {
				t.Fatalf("expected DSN %q, got %q", test.dsn, got)
			}
			if got := server.Sentry.BrowserDSN(); got != test.browserDSN {
				t.Fatalf("expected browser DSN %q, got %q", test.browserDSN, got)
			}
			if got := fmt.Sprint(server.Sentry); got != "[redacted]" {
				t.Fatalf("expected redacted Sentry configuration, got %q", got)
			}
		})
	}
}

func TestServerCaptchaPolicy(t *testing.T) {
	tests := []struct {
		name           string
		environment    string
		secret         string
		siteKey        string
		clientIPSource string
		wantVerify     bool
		wantErr        string
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
			name:           "production verifies configured secret",
			environment:    "production",
			secret:         "captcha-secret",
			siteKey:        "captcha-site-key",
			clientIPSource: "railway",
			wantVerify:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_ENV"] = test.environment
			values["WGA_RECAPTCHA_SECRET"] = test.secret
			values["WGA_RECAPTCHA_SITE_KEY"] = test.siteKey
			values["WGA_CLIENT_IP_SOURCE"] = test.clientIPSource

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

func TestServerClientIPSource(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		source      string
		want        ClientIPSource
		wantErr     string
	}{
		{name: "development defaults to direct", environment: "development", want: ClientIPSourceDirect},
		{name: "test defaults to direct", environment: "test", want: ClientIPSourceDirect},
		{name: "development explicit direct", environment: "development", source: "direct", want: ClientIPSourceDirect},
		{name: "development explicit railway", environment: "development", source: "railway", want: ClientIPSourceRailway},
		{name: "production requires explicit source", environment: "production", wantErr: "WGA_CLIENT_IP_SOURCE"},
		{name: "staging requires explicit source", environment: "staging", wantErr: "WGA_CLIENT_IP_SOURCE"},
		{name: "production explicit railway", environment: "production", source: "railway", want: ClientIPSourceRailway},
		{name: "unknown source", environment: "development", source: "forwarded", wantErr: "WGA_CLIENT_IP_SOURCE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_ENV"] = test.environment
			values["WGA_CLIENT_IP_SOURCE"] = test.source
			values["WGA_RECAPTCHA_SECRET"] = "captcha-secret"

			server, err := LoadFrom(lookup(values)).Server()
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected server configuration error: %v", err)
			}
			if server.ClientIPSource != test.want {
				t.Fatalf("client IP source = %q, want %q", server.ClientIPSource, test.want)
			}
		})
	}
}

func TestPostcardTokenKeyringParsesMultipleKeys(t *testing.T) {
	activeBytes := bytes.Repeat([]byte{0x22}, 32)
	previousBytes := bytes.Repeat([]byte{0x11}, 32)
	values := validValues()
	values["WGA_POSTCARD_TOKEN_KEYS"] = fmt.Sprintf(
		`{"v1":%q,"v2":%q}`,
		base64.RawURLEncoding.EncodeToString(previousBytes),
		base64.RawURLEncoding.EncodeToString(activeBytes),
	)
	values["WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID"] = "v2"

	configuration := LoadFrom(lookup(values))
	keyring, err := configuration.PostcardTokenKeyring()
	if err != nil {
		t.Fatalf("unexpected postcard token keyring error: %v", err)
	}
	if got, want := keyring.ActiveKeyID(), "v2"; got != want {
		t.Fatalf("active key ID = %q, want %q", got, want)
	}
	if got := keyring.ActiveKey().Bytes(); !bytes.Equal(got, activeBytes) {
		t.Fatalf("active key did not match configured key")
	}
	previous, ok := keyring.Key("v1")
	if !ok {
		t.Fatal("expected previous key to remain available")
	}
	if got := previous.Bytes(); !bytes.Equal(got, previousBytes) {
		t.Fatalf("previous key did not match configured key")
	}
	if _, ok := keyring.Key("missing"); ok {
		t.Fatal("unexpected missing key")
	}

	returnedBytes := keyring.ActiveKey().Bytes()
	returnedBytes[0] = 0
	if got := keyring.ActiveKey().Bytes(); !bytes.Equal(got, activeBytes) {
		t.Fatal("mutating returned bytes changed the configured key")
	}

	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("unexpected server configuration error: %v", err)
	}
	if got, want := server.Postcards.TokenKeyring().ActiveKeyID(), "v2"; got != want {
		t.Fatalf("server postcard active key ID = %q, want %q", got, want)
	}
	if got := fmt.Sprint(keyring); got != "[redacted]" {
		t.Fatalf("expected redacted keyring, got %q", got)
	}
	if got := fmt.Sprint(keyring.ActiveKey()); got != "[redacted]" {
		t.Fatalf("expected redacted key, got %q", got)
	}
}

func TestPostcardTokenKeyringRejectsInvalidConfiguration(t *testing.T) {
	validKey := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32))
	shortKey := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x44}, 31))
	tests := []struct {
		name      string
		keys      string
		activeID  string
		wantErr   string
		forbidden []string
	}{
		{
			name:      "missing keys",
			activeID:  "v1",
			wantErr:   "WGA_POSTCARD_TOKEN_KEYS must be set",
			forbidden: []string{"v1"},
		},
		{
			name:      "malformed JSON",
			keys:      `{"v1":"json-secret-marker"`,
			activeID:  "v1",
			wantErr:   "JSON object",
			forbidden: []string{"json-secret-marker", `{"v1"`},
		},
		{
			name:      "malformed Base64URL",
			keys:      `{"v1":"base64+secret/marker"}`,
			activeID:  "v1",
			wantErr:   "valid Base64URL-encoded keys",
			forbidden: []string{"base64+secret/marker", "v1"},
		},
		{
			name:      "invalid key ID",
			keys:      fmt.Sprintf(`{"bad key secret-marker":%q}`, validKey),
			activeID:  "v1",
			wantErr:   "invalid key ID",
			forbidden: []string{"bad key secret-marker", validKey},
		},
		{
			name:      "wrong key length",
			keys:      fmt.Sprintf(`{"v1":%q}`, shortKey),
			activeID:  "v1",
			wantErr:   "32-byte AES-256 keys",
			forbidden: []string{shortKey, "v1"},
		},
		{
			name:      "missing active ID",
			keys:      fmt.Sprintf(`{"v1":%q}`, validKey),
			wantErr:   "WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID must be set",
			forbidden: []string{validKey, "v1"},
		},
		{
			name:      "missing active key",
			keys:      fmt.Sprintf(`{"v1":%q}`, validKey),
			activeID:  "v2-secret-marker",
			wantErr:   "must identify a key",
			forbidden: []string{validKey, "v2-secret-marker"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_POSTCARD_TOKEN_KEYS"] = test.keys
			values["WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID"] = test.activeID

			_, err := LoadFrom(lookup(values)).PostcardTokenKeyring()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(err.Error(), forbidden) {
					t.Fatalf("configuration error exposed a supplied value: %v", err)
				}
			}
		})
	}
}

func TestPostcardTokenKeyringValidationIsCapabilitySpecific(t *testing.T) {
	configuration := LoadFrom(lookup(validValues()))

	if _, err := configuration.Server(); err != nil {
		t.Fatalf("server should not require postcard token keys: %v", err)
	}
	if _, err := configuration.PostcardTokenKeyring(); err == nil {
		t.Fatal("expected postcard token keyring validation error")
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

func TestMigrationSeedSQLitePath(t *testing.T) {
	values := validValues()
	values["WGA_SEED_SQLITE_PATH"] = "/prod-data/wga-src.sqlite"

	if got, want := LoadFrom(lookup(values)).Migrations().SeedSQLitePath(), "/prod-data/wga-src.sqlite"; got != want {
		t.Fatalf("seed SQLite path = %q, want %q", got, want)
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
		"WGA_S3_ENDPOINT":        "http://127.0.0.1:3900",
		"WGA_S3_BUCKET":          "wga-assets",
		"WGA_S3_REGION":          "garage",
		"WGA_S3_ACCESS_KEY":      "GKlocaluploads",
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

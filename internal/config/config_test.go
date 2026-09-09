package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
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
			clientIPSource: "cloudflare-railway",
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
			if test.clientIPSource == "cloudflare-railway" {
				values["WGA_CLOUDFLARE_EDGE_SECRET"] = cloudflareSecret(0x40)
			}

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
		{name: "development explicit cloudflare via railway", environment: "development", source: "cloudflare-railway", want: ClientIPSourceCloudflareRailway},
		{name: "production requires explicit source", environment: "production", wantErr: "WGA_CLIENT_IP_SOURCE"},
		{name: "staging requires explicit source", environment: "staging", wantErr: "WGA_CLIENT_IP_SOURCE"},
		{name: "production rejects railway", environment: "production", source: "railway", wantErr: "must be cloudflare-railway"},
		{name: "staging rejects direct", environment: "staging", source: "direct", wantErr: "must be cloudflare-railway"},
		{name: "production accepts authenticated cloudflare", environment: "production", source: "cloudflare-railway", want: ClientIPSourceCloudflareRailway},
		{name: "unknown source", environment: "development", source: "forwarded", wantErr: "WGA_CLIENT_IP_SOURCE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_ENV"] = test.environment
			values["WGA_CLIENT_IP_SOURCE"] = test.source
			values["WGA_RECAPTCHA_SECRET"] = "captcha-secret"
			if test.source == "cloudflare-railway" {
				values["WGA_CLOUDFLARE_EDGE_SECRET"] = cloudflareSecret(0x41)
			}

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

func TestServerCloudflareOriginSecretsSupportRotationAndRedaction(t *testing.T) {
	current := cloudflareSecret(0x51)
	next := cloudflareSecret(0x52)
	values := validValues()
	values["WGA_CLIENT_IP_SOURCE"] = "cloudflare-railway"
	values["WGA_CLOUDFLARE_EDGE_SECRET"] = current
	values["WGA_CLOUDFLARE_EDGE_SECRET_NEXT"] = next

	server, err := LoadFrom(lookup(values)).Server()
	if err != nil {
		t.Fatalf("unexpected server configuration error: %v", err)
	}
	if got := server.CloudflareOriginSecrets.Current().Value(); got != current {
		t.Fatal("current Cloudflare origin secret did not match configured value")
	}
	staged, ok := server.CloudflareOriginSecrets.Next()
	if !ok || staged.Value() != next {
		t.Fatal("next Cloudflare origin secret did not match configured value")
	}
	for _, formatted := range []string{
		fmt.Sprint(server.CloudflareOriginSecrets),
		fmt.Sprintf("%#v", server.CloudflareOriginSecrets),
		fmt.Sprint(server.CloudflareOriginSecrets.Current()),
		fmt.Sprint(staged),
	} {
		if formatted != "[redacted]" && formatted != "config.CloudflareOriginSecrets([redacted])" {
			t.Fatalf("expected redacted Cloudflare origin secrets, got %q", formatted)
		}
		if strings.Contains(formatted, current) || strings.Contains(formatted, next) {
			t.Fatalf("formatted keyring exposed a configured secret: %q", formatted)
		}
	}
}

func TestServerRejectsInvalidCloudflareOriginSecrets(t *testing.T) {
	validSecret := cloudflareSecret(0x61)
	tests := []struct {
		name      string
		current   string
		next      string
		wantErr   string
		forbidden []string
	}{
		{
			name:      "malformed current secret",
			current:   "not+a+base64url+secret",
			wantErr:   "WGA_CLOUDFLARE_EDGE_SECRET",
			forbidden: []string{"not+a+base64url+secret"},
		},
		{
			name:      "short current secret",
			current:   base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x62}, 31)),
			wantErr:   "WGA_CLOUDFLARE_EDGE_SECRET",
			forbidden: []string{"YmJi"},
		},
		{
			name:      "next secret without current",
			next:      validSecret,
			wantErr:   "WGA_CLOUDFLARE_EDGE_SECRET must be set",
			forbidden: []string{validSecret},
		},
		{
			name:      "malformed next secret",
			current:   validSecret,
			next:      "malformed-next-secret",
			wantErr:   "WGA_CLOUDFLARE_EDGE_SECRET_NEXT",
			forbidden: []string{validSecret, "malformed-next-secret"},
		},
		{
			name:      "duplicate rotation secret",
			current:   validSecret,
			next:      validSecret,
			wantErr:   "must differ",
			forbidden: []string{validSecret},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values["WGA_CLOUDFLARE_EDGE_SECRET"] = test.current
			values["WGA_CLOUDFLARE_EDGE_SECRET_NEXT"] = test.next

			_, err := LoadFrom(lookup(values)).Server()
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

func TestServerRequiresCloudflareTrustForEnabledProductionProtection(t *testing.T) {
	t.Run("missing Cloudflare origin secret", func(t *testing.T) {
		values := validValues()
		values["WGA_ENV"] = "production"
		values["WGA_CLIENT_IP_SOURCE"] = "cloudflare-railway"
		values["WGA_PUBLIC_REQUEST_PROTECTION_MODE"] = "observe"

		_, err := LoadFrom(lookup(values)).Server()
		if err == nil || !strings.Contains(err.Error(), "WGA_CLOUDFLARE_EDGE_SECRET") {
			t.Fatalf("expected missing Cloudflare origin secret error, got %v", err)
		}
	})

	t.Run("unsupported production trust source", func(t *testing.T) {
		values := validValues()
		values["WGA_ENV"] = "production"
		values["WGA_CLIENT_IP_SOURCE"] = "railway"
		values["WGA_PUBLIC_REQUEST_PROTECTION_MODE"] = "enforce"

		_, err := LoadFrom(lookup(values)).Server()
		if err == nil || !strings.Contains(err.Error(), "WGA_CLIENT_IP_SOURCE must be cloudflare-railway") {
			t.Fatalf("expected Cloudflare trust-source error, got %v", err)
		}
	})
}

func cloudflareSecret(fill byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{fill}, 32))
}

func TestServerPublicRequestProtectionConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		want   PublicRequestProtection
	}{
		{
			name: "defaults to disabled conservative limits",
			want: PublicRequestProtection{
				Mode:                      ProtectionModeOff,
				MaxConcurrentReads:        8,
				SearchRequestsPerMinute:   20,
				FragmentRequestsPerMinute: 30,
				DetailRequestsPerMinute:   60,
				LimiterEntryCapacity:      4096,
				RetryAfter:                5 * time.Second,
			},
		},
		{
			name: "parses configured limits",
			values: map[string]string{
				"WGA_PUBLIC_REQUEST_PROTECTION_MODE":  "observe",
				"WGA_PUBLIC_READ_MAX_CONCURRENT":      "12",
				"WGA_PUBLIC_READ_SEARCH_PER_MINUTE":   "24",
				"WGA_PUBLIC_READ_FRAGMENT_PER_MINUTE": "36",
				"WGA_PUBLIC_READ_DETAIL_PER_MINUTE":   "72",
				"WGA_PUBLIC_READ_LIMITER_CAPACITY":    "2048",
				"WGA_PUBLIC_READ_RETRY_AFTER":         "1500ms",
			},
			want: PublicRequestProtection{
				Mode:                      ProtectionModeObserve,
				MaxConcurrentReads:        12,
				SearchRequestsPerMinute:   24,
				FragmentRequestsPerMinute: 36,
				DetailRequestsPerMinute:   72,
				LimiterEntryCapacity:      2048,
				RetryAfter:                1500 * time.Millisecond,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			for key, value := range test.values {
				values[key] = value
			}

			server, err := LoadFrom(lookup(values)).Server()
			if err != nil {
				t.Fatalf("unexpected server configuration error: %v", err)
			}
			if server.PublicRequestProtection != test.want {
				t.Fatalf("public request protection = %#v, want %#v", server.PublicRequestProtection, test.want)
			}
		})
	}
}

func TestServerRejectsInvalidPublicRequestProtectionConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "mode", key: "WGA_PUBLIC_REQUEST_PROTECTION_MODE", value: "enabled"},
		{name: "concurrent limit", key: "WGA_PUBLIC_READ_MAX_CONCURRENT", value: "0"},
		{name: "search limit", key: "WGA_PUBLIC_READ_SEARCH_PER_MINUTE", value: "not-an-integer"},
		{name: "fragment limit", key: "WGA_PUBLIC_READ_FRAGMENT_PER_MINUTE", value: "-1"},
		{name: "detail limit", key: "WGA_PUBLIC_READ_DETAIL_PER_MINUTE", value: "0"},
		{name: "limiter capacity", key: "WGA_PUBLIC_READ_LIMITER_CAPACITY", value: "0"},
		{name: "retry duration", key: "WGA_PUBLIC_READ_RETRY_AFTER", value: "immediately"},
		{name: "zero retry duration", key: "WGA_PUBLIC_READ_RETRY_AFTER", value: "0s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validValues()
			values[test.key] = test.value

			_, err := LoadFrom(lookup(values)).Server()
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected error containing %q, got %v", test.key, err)
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

package main

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/requesttrust"
)

func TestCommandCapabilityFor(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want commandCapability
	}{
		{name: "default server", want: commandNeedsServer},
		{name: "server flags", args: []string{"--dev"}, want: commandNeedsServer},
		{name: "HTTP listener", args: []string{"--http", "0.0.0.0:8090"}, want: commandNeedsServer},
		{name: "HTTPS listener", args: []string{"--https", "0.0.0.0:8443"}, want: commandNeedsServer},
		{name: "allowed origins", args: []string{"--origins", "https://gallery.example"}, want: commandNeedsServer},
		{name: "equals HTTP listener", args: []string{"--http=0.0.0.0:8090"}, want: commandNeedsServer},
		{name: "serve", args: []string{"serve"}, want: commandNeedsServer},
		{name: "sitemap", args: []string{"generate-sitemap"}, want: commandNeedsSitemap},
		{name: "migration", args: []string{"migrate", "up"}, want: commandNeedsNothing},
		{name: "migration collections", args: []string{"migrate", "collections"}, want: commandNeedsNothing},
		{name: "postcard inspect", args: []string{"postcards", "inspect"}, want: commandNeedsNothing},
		{name: "postcard rewrap", args: []string{"postcards", "rewrap-token-key", "--from", "old"}, want: commandNeedsNothing},
		{name: "music URLs", args: []string{"generate-music-urls"}, want: commandNeedsNothing},
		{name: "unknown command", args: []string{"not-a-command"}, want: commandNeedsNothing},
		{name: "server data directory", args: []string{"--dir", "test_data"}, want: commandNeedsServer},
		{name: "migration data directory", args: []string{"--dir", "test_data", "migrate", "up"}, want: commandNeedsNothing},
		{name: "help", args: []string{"serve", "--help"}, want: commandNeedsNothing},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := commandCapabilityFor(test.args); got != test.want {
				t.Fatalf("expected capability %d, got %d", test.want, got)
			}
		})
	}
}

func TestServerConfigForRequiresPostcardTokenKeyring(t *testing.T) {
	values := map[string]string{
		"WGA_ENV":            "development",
		"WGA_PROTOCOL":       "http",
		"WGA_HOSTNAME":       "example.test",
		"WGA_SENDER_NAME":    "WGA",
		"WGA_SENDER_ADDRESS": "sender@example.test",
	}
	runtimeConfig := config.LoadFrom(func(key string) string { return values[key] })
	if _, err := serverConfigFor(runtimeConfig); err == nil || !strings.Contains(err.Error(), "WGA_POSTCARD_TOKEN_KEYS") {
		t.Fatalf("server token keyring error = %v", err)
	}

	values["WGA_POSTCARD_TOKEN_KEYS"] = `{"primary":"` + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x61}, 32)) + `"}`
	values["WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID"] = "primary"
	runtimeConfig = config.LoadFrom(func(key string) string { return values[key] })
	server, err := serverConfigFor(runtimeConfig)
	if err != nil {
		t.Fatalf("valid server configuration: %v", err)
	}
	if server.Postcards.TokenKeyring().ActiveKeyID() != "primary" {
		t.Fatal("validated token keyring was not carried into server postcard settings")
	}
}

func TestServerConfigForJoinsServerAndTokenKeyringErrors(t *testing.T) {
	runtimeConfig := config.LoadFrom(func(string) string { return "" })
	_, err := serverConfigFor(runtimeConfig)
	if err == nil || !strings.Contains(err.Error(), "WGA_ENV") || !strings.Contains(err.Error(), "WGA_POSTCARD_TOKEN_KEYS") {
		t.Fatalf("joined server configuration error = %v", err)
	}
}

func TestItinerarySecurityPolicy(t *testing.T) {
	t.Run("production HTTPS uses secure production cookie", func(t *testing.T) {
		server := loadServer(t, map[string]string{
			"WGA_ENV":                "production",
			"WGA_PROTOCOL":           "https",
			"WGA_HOSTNAME":           "gallery.example",
			"WGA_CLIENT_IP_SOURCE":   "railway",
			"WGA_RECAPTCHA_SECRET":   "captcha-secret",
			"WGA_RECAPTCHA_SITE_KEY": "captcha-site-key",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.test",
		})

		policy, err := itinerarySecurityPolicy(server, requesttrust.New(requesttrust.Source(server.ClientIPSource)))
		if err != nil {
			t.Fatalf("itinerary security policy: %v", err)
		}
		if policy.CanonicalOrigin != "https://gallery.example" {
			t.Errorf("canonical origin = %q", policy.CanonicalOrigin)
		}
		if policy.Production.Name != "__Host-wga_itinerary" || !policy.Production.Secure {
			t.Errorf("production cookie policy = %+v", policy.Production)
		}
		if policy.Development.Name != "" {
			t.Errorf("development cookie must not be opted in for production, got %q", policy.Development.Name)
		}
		if policy.TrustedClientID == nil {
			t.Fatal("trusted client resolver is required")
		}
	})

	t.Run("development HTTP opts in the development cookie", func(t *testing.T) {
		server := loadServer(t, map[string]string{
			"WGA_ENV":            "development",
			"WGA_PROTOCOL":       "http",
			"WGA_HOSTNAME":       "localhost:8090",
			"WGA_SENDER_NAME":    "WGA",
			"WGA_SENDER_ADDRESS": "sender@example.test",
		})

		policy, err := itinerarySecurityPolicy(server, requesttrust.New(requesttrust.Source(server.ClientIPSource)))
		if err != nil {
			t.Fatalf("itinerary security policy: %v", err)
		}
		if policy.Development.Name != "wga_itinerary_dev" || policy.Development.Secure {
			t.Errorf("development cookie policy = %+v", policy.Development)
		}
		if policy.Production.Name != "__Host-wga_itinerary" || !policy.Production.Secure {
			t.Errorf("production cookie policy = %+v", policy.Production)
		}
	})

	t.Run("staging HTTP without development opt-in fails closed", func(t *testing.T) {
		server := loadServer(t, map[string]string{
			"WGA_ENV":                "staging",
			"WGA_PROTOCOL":           "http",
			"WGA_HOSTNAME":           "staging.example",
			"WGA_CLIENT_IP_SOURCE":   "railway",
			"WGA_RECAPTCHA_SECRET":   "captcha-secret",
			"WGA_RECAPTCHA_SITE_KEY": "captcha-site-key",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.test",
		})

		if _, err := itinerarySecurityPolicy(server, requesttrust.New(requesttrust.Source(server.ClientIPSource))); err == nil {
			t.Fatal("expected staging HTTP policy to fail closed without a development cookie")
		}
	})
}

func loadServer(t *testing.T, values map[string]string) config.Server {
	t.Helper()
	server, err := config.LoadFrom(func(key string) string { return values[key] }).Server()
	if err != nil {
		t.Fatalf("load server config: %v", err)
	}
	return server
}

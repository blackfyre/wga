package feedback

import (
	"testing"

	"github.com/blackfyre/wga/internal/buildinfo"
	"github.com/blackfyre/wga/internal/config"
)

func TestFeedbackFormView(t *testing.T) {
	originalVersion := buildinfo.Version
	buildinfo.Version = "2.0.0-rc4"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
	})

	tests := []struct {
		name    string
		referer string
		context string
	}{
		{name: "home", referer: "https://wga.example/", context: "Home"},
		{name: "other page", referer: "https://wga.example/artworks", context: "Current page"},
		{name: "missing", context: "Current page"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := feedbackFormView(test.referer, config.EnvironmentStaging)
			if view.Context != test.context {
				t.Errorf("context = %q, want %q", view.Context, test.context)
			}
			if view.Build != "staging · 2.0.0-rc4" {
				t.Errorf("build = %q", view.Build)
			}
		})
	}
}

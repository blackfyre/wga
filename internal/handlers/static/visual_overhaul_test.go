package static

import (
	"testing"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/config"
)

func TestVisualOverhaulReferenceIsEmbedded(t *testing.T) {
	content, err := assets.ReferenceFiles.ReadFile("reference/visual-overhaul.html")
	if err != nil {
		t.Fatalf("read visual overhaul reference: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("visual overhaul reference is empty")
	}
	if string(content[:15]) != "<!DOCTYPE html>" {
		t.Fatal("visual overhaul reference must remain a standalone HTML document")
	}
}

func TestVisualOverhaulRouteExcludesProduction(t *testing.T) {
	if shouldRegisterVisualOverhaul(config.EnvironmentProduction) {
		t.Fatal("visual overhaul route must remain excluded from production")
	}
	if !shouldRegisterVisualOverhaul(config.EnvironmentDevelopment) {
		t.Fatal("visual overhaul route must be available in development")
	}
}

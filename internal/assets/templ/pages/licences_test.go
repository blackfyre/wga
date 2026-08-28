package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestLicencesPageRendersSharedPageHeadOnce(t *testing.T) {
	ctx := utils.DecorateContext(context.Background(), utils.TitleKey, "Open-source licences")
	var output bytes.Buffer
	if err := LicencesPage("<p>third-party components</p>").Render(ctx, &output); err != nil {
		t.Fatalf("render licences page: %v", err)
	}
	rendered := output.String()

	for _, expected := range []string{
		"OPEN SOURCE",
		">Open-source licences</h1>",
		"text-(length:--t-11)",
		"text-muted",
		"text-(length:--t-32)",
		"md:text-(length:--t-44)",
		"third-party components",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("licences page does not contain %q", expected)
		}
	}
	if got := strings.Count(rendered, "<h1"); got != 1 {
		t.Errorf("h1 count = %d, want 1", got)
	}
}

package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLayoutMainHeaderIsTheBackToTopTarget(t *testing.T) {
	var output bytes.Buffer
	if err := LayoutMain().Render(context.Background(), &output); err != nil {
		t.Fatalf("render layout: %v", err)
	}
	rendered := output.String()
	if got := strings.Count(rendered, `id="top"`); got != 1 {
		t.Fatalf("expected exactly one #top target, got %d", got)
	}
	if !strings.Contains(rendered, `<header id="top"`) {
		t.Error(`the shared header must carry id="top"`)
	}
	if !strings.Contains(rendered, `href="#top"`) {
		t.Error("the shared footer must link back to #top")
	}
	if strings.Contains(rendered, "jump") {
		t.Error("no floating back-to-top control may render")
	}
}

package error_pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderErrorBlock(t *testing.T, component templ.Component) string {
	t.Helper()

	var output bytes.Buffer
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render error block: %v", err)
	}

	return output.String()
}

func TestErrorBlocksProvideRecoveryRoutesWithoutJavascript(t *testing.T) {
	cases := []struct {
		name      string
		component templ.Component
		kicker    string
		title     string
		copy      string
	}{
		{
			name:      "404",
			component: NotFoundBlock(),
			kicker:    "404 — NOT FOUND",
			title:     "This record is not in the collection.",
			copy:      "The address may have changed",
		},
		{
			name:      "500",
			component: ServerFaultBlock(),
			kicker:    "500 — SERVICE ERROR",
			title:     "The archive could not complete that request.",
			copy:      "Please try again shortly",
		},
		{
			name:      "400",
			component: BadRequestBlock(),
			kicker:    "400 — BAD REQUEST",
			title:     "That request is not supported.",
			copy:      "Please return to the collection",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderErrorBlock(t, tc.component)
			for _, expected := range []string{
				tc.kicker,
				tc.title,
				tc.copy,
				"text-(length:--t-32)",
				"md:text-(length:--t-44)",
				`href="/"`,
				`href="/artworks"`,
				"RETURN TO THE GALLERY",
				"BROWSE ALL ARTWORKS",
			} {
				if !strings.Contains(rendered, expected) {
					t.Errorf("rendered %s block does not contain %q\ngot: %s", tc.name, expected, rendered)
				}
			}
			if strings.Count(rendered, "<h1") != 1 {
				t.Errorf("h1 count = %d, want 1", strings.Count(rendered, "<h1"))
			}
			if strings.Contains(rendered, "hx-get") || strings.Contains(rendered, "hx-post") {
				t.Error("error blocks must not depend on JavaScript")
			}
		})
	}
}

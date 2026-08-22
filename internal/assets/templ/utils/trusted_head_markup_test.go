package utils_test

import (
	"context"
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestWithTrustedHeadMarkupReturnsSuppliedValue(t *testing.T) {
	const markup = `<script src="/assets/js/trusted.js"></script>`

	ctx := templutils.WithTrustedHeadMarkup(context.Background(), markup)

	if got := templutils.GetTrustedHeadMarkup(ctx); got != markup {
		t.Fatalf("expected supplied trusted markup %q, got %q", markup, got)
	}
}

func TestGetTrustedHeadMarkupDefaultsToEmpty(t *testing.T) {
	if got := templutils.GetTrustedHeadMarkup(context.Background()); got != "" {
		t.Fatalf("expected empty trusted markup by default, got %q", got)
	}
}

func TestWithTrustedHeadMarkupDoesNotMutateParent(t *testing.T) {
	parent := context.Background()

	templutils.WithTrustedHeadMarkup(parent, `<script>alert(1)</script>`)

	if got := templutils.GetTrustedHeadMarkup(parent); got != "" {
		t.Fatalf("expected parent context to remain empty, got %q", got)
	}
}

func TestWithTrustedHeadMarkupEmptyValue(t *testing.T) {
	ctx := templutils.WithTrustedHeadMarkup(context.Background(), "")

	if got := templutils.GetTrustedHeadMarkup(ctx); got != "" {
		t.Fatalf("expected empty trusted markup to round-trip as empty, got %q", got)
	}
}

func TestGetTrustedHeadMarkupIgnoresUnrelatedStringKey(t *testing.T) {
	ctx := context.WithValue(context.Background(), "scripts:header", "<script>alert(1)</script>")

	if got := templutils.GetTrustedHeadMarkup(ctx); got != "" {
		t.Fatalf("expected unrelated string context value to be ignored, got %q", got)
	}
}

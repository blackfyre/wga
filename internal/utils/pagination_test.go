package utils

import (
	"strings"
	"testing"
)

func TestPaginationSetHtmxSwap(t *testing.T) {
	pagination := NewPagination(32, 16, 1, "/artworks", "artwork-search-results", "/artworks/results")
	pagination.SetHtmxSwap("#artwork-search-results > *", "innerHTML")

	rendered := string(pagination.Render())
	if !strings.Contains(rendered, "hx-select='#artwork-search-results > *' hx-swap='innerHTML'") {
		t.Fatalf("expected pagination to preserve the results container, got %q", rendered)
	}
}

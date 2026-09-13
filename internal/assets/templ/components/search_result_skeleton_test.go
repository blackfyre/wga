package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSearchResultSkeletonRendersNineInertCardsOrRows(t *testing.T) {
	for _, view := range []string{"grid", "list"} {
		var output bytes.Buffer
		if err := SearchResultSkeleton("search-skeleton", view).Render(context.Background(), &output); err != nil {
			t.Fatalf("render %s skeleton: %v", view, err)
		}
		rendered := output.String()
		for _, expected := range []string{`id="search-skeleton"`, `aria-hidden="true"`, `data-skeleton-count="9"`, `data-search-result-skeleton`} {
			if !strings.Contains(rendered, expected) {
				t.Errorf("%s skeleton missing %q", view, expected)
			}
		}
		marker := "aspect-[4/5]"
		if view == "list" {
			marker = "min-h-[72px]"
		}
		if count := strings.Count(rendered, marker); count != 9 {
			t.Errorf("%s skeleton items = %d, want 9", view, count)
		}
	}
}

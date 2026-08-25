package utils_test

import (
	"context"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestItineraryProjectionGettersDefaultEmpty(t *testing.T) {
	ctx := context.Background()

	if got := templutils.GetItineraryCSRF(ctx); got != "" {
		t.Fatalf("default CSRF = %q, want empty", got)
	}
	if got := templutils.GetItineraryTray(ctx); got.Count != 0 {
		t.Fatalf("default tray count = %d, want 0", got.Count)
	}
	if templutils.GetItineraryAdded(ctx, "aw0000000000001") {
		t.Fatal("default added state must be false")
	}
}

func TestItineraryProjectionRoundTrip(t *testing.T) {
	tray := dto.ItineraryTrayView{Count: 2, BuilderURL: "/itineraries/new"}
	added := map[string]bool{"aw0000000000001": true}

	ctx := templutils.WithItineraryProjection(context.Background(), "csrf-token", tray, added)

	if got := templutils.GetItineraryCSRF(ctx); got != "csrf-token" {
		t.Fatalf("CSRF = %q, want csrf-token", got)
	}
	if got := templutils.GetItineraryTray(ctx); got.Count != 2 {
		t.Fatalf("tray count = %d, want 2", got.Count)
	}
	if !templutils.GetItineraryAdded(ctx, "aw0000000000001") {
		t.Fatal("added artwork must be reported")
	}
	if templutils.GetItineraryAdded(ctx, "aw0000000000002") {
		t.Fatal("unadded artwork must not be reported")
	}
}

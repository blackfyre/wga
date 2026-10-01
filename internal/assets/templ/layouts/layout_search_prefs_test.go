package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestLayoutBaseRendersRememberedArtworkActions(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "unset", value: "", want: `<html lang="en">`},
		{name: "on", value: "on", want: `<html lang="en" data-aw-actions="on">`},
		{name: "off", value: "off", want: `<html lang="en" data-aw-actions="off">`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := utils.WithArtworkActions(context.Background(), test.value)
			if err := LayoutBase("", "").Render(ctx, &output); err != nil {
				t.Fatalf("render layout: %v", err)
			}
			if !strings.Contains(output.String(), test.want) {
				t.Fatalf("layout root missing %s", test.want)
			}
		})
	}
}

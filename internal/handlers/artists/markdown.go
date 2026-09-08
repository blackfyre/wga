package artists

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const markdownMediaType = "text/markdown"

func prefersMarkdown(accept string) bool {
	markdownQuality := -1.0
	htmlQuality := -1.0
	for _, value := range strings.Split(accept, ",") {
		mediaType, parameters, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		quality := 1.0
		if raw := parameters["q"]; raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil || parsed < 0 || parsed > 1 {
				continue
			}
			quality = parsed
		}
		switch strings.ToLower(mediaType) {
		case markdownMediaType:
			markdownQuality = max(markdownQuality, quality)
		case "text/html", "application/xhtml+xml":
			htmlQuality = max(htmlQuality, quality)
		}
	}
	return markdownQuality > 0 && markdownQuality >= htmlQuality
}

func generatedMarkdownPath(kind, slug string) string {
	return fmt.Sprintf("/agents/%s/%s.md", kind, utils.ExtractIdFromString(slug))
}

func negotiateMarkdown(c *core.RequestEvent, kind, slug string) (bool, error) {
	c.Response.Header().Add("Vary", "Accept")
	if !prefersMarkdown(c.Request.Header.Get("Accept")) {
		return false, nil
	}
	return true, c.Redirect(http.StatusTemporaryRedirect, generatedMarkdownPath(kind, slug))
}

func advertiseMarkdown(c *core.RequestEvent, path string) {
	c.Response.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"alternate\"; type=\"%s\"", utils.AssetUrl(path), markdownMediaType))
}

func decorateMarkdownAlternate(ctx context.Context, path string) context.Context {
	return tmplUtils.DecorateContext(ctx, tmplUtils.AlternateMarkdownURLKey, utils.AssetUrl(path))
}

package artists

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/blackfyre/wga/internal/agentcontent"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const markdownMediaType = "text/markdown"

func generatedMarkdownPath(kind, slug string) string {
	return fmt.Sprintf("/agents/%s/%s.md", kind, utils.ExtractIdFromString(slug))
}

func generatedMarkdownAvailable(app core.App, path string, canonicalPath string) bool {
	resource, err := agentcontent.LookupCurrent(app, strings.TrimPrefix(path, "/"))
	return err == nil && slices.Contains(resource.AcceptedPaths, canonicalPath)
}

func advertiseMarkdown(c *core.RequestEvent, path string) {
	c.Response.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"alternate\"; type=\"%s\"", utils.AssetUrl(path), markdownMediaType))
}

func decorateMarkdownAlternate(ctx context.Context, path string) context.Context {
	return tmplUtils.DecorateContext(ctx, tmplUtils.AlternateMarkdownURLKey, utils.AssetUrl(path))
}

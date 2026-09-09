package handlers

import (
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/agentcontent"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const markdownMediaType = "text/markdown"

type generatedResourceReader func(core.App, string) (agentcontent.Resource, error)

func registerMarkdownNegotiationMiddleware(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return negotiateGeneratedMarkdown(app, e, agentcontent.ReadCurrent, e.Next)
		})
		return se.Next()
	})
}

// negotiateGeneratedMarkdown redirects only canonical, currently published
// records. It runs before document/session middleware and reads generated files,
// never PocketBase, so missing and alias routes retain their normal handler
// validation without paying personalised document work for successful redirects.
func negotiateGeneratedMarkdown(app core.App, e *core.RequestEvent, read generatedResourceReader, next func() error) error {
	if e == nil || e.Request == nil || (e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead) {
		return next()
	}

	relative, location, ok := generatedMarkdownResource(e.Request.URL.Path)
	if !ok {
		return next()
	}
	e.Response.Header().Add("Vary", "Accept")
	if !prefersMarkdown(e.Request.Header.Get("Accept")) {
		return next()
	}

	resource, err := read(app, relative)
	if err != nil {
		return next()
	}
	canonical, err := url.Parse(resource.CanonicalURL)
	if err != nil || canonical.Path != e.Request.URL.Path {
		return next()
	}
	return e.Redirect(http.StatusTemporaryRedirect, location)
}

func generatedMarkdownResource(path string) (relative string, location string, ok bool) {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	kind := ""
	slug := ""
	switch {
	case len(segments) == 2 && segments[0] == "artists":
		kind, slug = "artists", segments[1]
	case len(segments) == 3 && segments[0] == "artists":
		kind, slug = "artworks", segments[2]
	default:
		return "", "", false
	}
	if dash := strings.LastIndexByte(slug, '-'); dash <= 0 || dash == len(slug)-1 {
		return "", "", false
	}
	id := utils.ExtractIdFromString(slug)
	relative = "agents/" + kind + "/" + id + ".md"
	return relative, "/" + relative, true
}

type representationQuality struct {
	value       float64
	specificity int
}

func prefersMarkdown(accept string) bool {
	markdown := acceptedQuality(accept, markdownMediaType)
	html := acceptedQuality(accept, "text/html")
	xhtml := acceptedQuality(accept, "application/xhtml+xml")
	if xhtml.value > html.value {
		html = xhtml
	}
	return markdown.value > 0 && (markdown.value > html.value || markdown.value == html.value && markdown.specificity == 2)
}

func acceptedQuality(accept string, representation string) representationQuality {
	target := strings.SplitN(representation, "/", 2)
	best := representationQuality{value: -1, specificity: -1}
	for _, raw := range strings.Split(accept, ",") {
		mediaType, parameters, err := mime.ParseMediaType(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		parts := strings.SplitN(strings.ToLower(mediaType), "/", 2)
		if len(parts) != 2 || (parts[0] != "*" && parts[0] != target[0]) || (parts[1] != "*" && parts[1] != target[1]) {
			continue
		}
		quality := 1.0
		if rawQuality := parameters["q"]; rawQuality != "" {
			quality, err = strconv.ParseFloat(rawQuality, 64)
			if err != nil || quality < 0 || quality > 1 {
				continue
			}
		}
		specificity := 2
		if parts[0] == "*" {
			specificity = 0
		} else if parts[1] == "*" {
			specificity = 1
		}
		if specificity > best.specificity || specificity == best.specificity && quality > best.value {
			best = representationQuality{value: quality, specificity: specificity}
		}
	}
	return best
}

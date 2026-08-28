package contributors

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	contributorworkflow "github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App, reader contributorworkflow.Reader) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/contributors", func(c *core.RequestEvent) error {
			fullUrl := tmplUtils.AssetUrl("/contributors")
			pushUrl := utils.GenerateCurrentRelativePageUrl(c)
			snapshot, err := reader.Current(c.Request.Context())
			if err != nil {
				return contributorServerError(app, c, "fetch_error", err)
			}

			if snapshot.Source == contributorworkflow.SnapshotSourceFileFallback {
				logging.RequestLogger(app, c).Warn("Contributors fallback served",
					"event", "contributors.request.completed",
					"outcome", "fallback",
					"source", snapshot.Source,
				)
			}

			content := pages.ContributorsPageDTO{
				Contributors: pageContributors(snapshot.Contributors),
			}

			ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Contributors")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "The people who have contributed to the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response.Header().Set("HX-Push-Url", pushUrl)
			c.Response.Header().Set("X-WGA-Contributors-Source", string(snapshot.Source))

			// Create a bytes buffer to write the response to
			var buf bytes.Buffer

			err = pages.ContributorsPage(content).Render(ctx, &buf)

			if err != nil {
				return contributorServerError(app, c, "render_error", err)
			}

			return c.HTML(http.StatusOK, buf.String())

		})

		return se.Next()
	})
}

func pageContributors(contributors []contributorworkflow.Contributor) []pages.GithubContributor {
	pageContributors := make([]pages.GithubContributor, len(contributors))
	for index, contributor := range contributors {
		pageContributors[index] = pages.GithubContributor{
			Login:         contributor.Login,
			AvatarURL:     safeAbsoluteURL(contributor.AvatarURL),
			HTMLURL:       safeAbsoluteURL(contributor.HTMLURL),
			Contributions: contributor.Contributions,
		}
	}

	return pageContributors
}

// safeAbsoluteURL returns raw only when it is an absolute HTTP(S) URL with a
// host. Any other value, including relative, protocol-relative, non-HTTP
// scheme, or empty input, resolves to an empty string so the template omits
// the avatar or profile anchor rather than rendering an untrusted URL.
func safeAbsoluteURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}

	return raw
}

func contributorServerError(app core.App, c *core.RequestEvent, outcome string, err error) error {
	logging.RequestLogger(app, c).Error("Contributors request failed",
		"event", "contributors.request.failed",
		"outcome", outcome,
		"error_type", logging.ErrorType(err),
		"error", logging.Redact(err),
	)

	return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
}

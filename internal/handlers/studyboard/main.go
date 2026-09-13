package studyboard

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	boardworkflow "github.com/blackfyre/wga/internal/studyboard"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const route = "/study-board"
const shelfRoute = route + "/shelf"

// RegisterHandlers registers the anonymous Study Board page.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET(shelfRoute, func(c *core.RequestEvent) error {
			board, err := boardworkflow.Resolve(app, c.Request.URL.Query().Get("board"))
			if err != nil {
				logging.RequestLogger(app, c).Error("Resolve Study Board shelf", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}
			var buffer bytes.Buffer
			if err := components.StudyBoardShelf(studyBoardWorks(board.Works)).Render(c.Request.Context(), &buffer); err != nil {
				logging.RequestLogger(app, c).Error("Render Study Board shelf", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}
			return c.HTML(http.StatusOK, buffer.String())
		})
		se.Router.GET(route, func(c *core.RequestEvent) error {
			query := c.Request.URL.Query()
			raw := strings.Join(query["board"], ",")
			board, err := boardworkflow.Resolve(app, raw)
			if err != nil {
				logging.RequestLogger(app, c).Error("Resolve Study Board", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}

			ids := board.IDs()
			canonical := canonicalPath(ids)
			if !canonicalQuery(query, ids) && !utils.IsHtmxRequest(c) {
				return c.Redirect(http.StatusFound, canonical)
			}

			ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Study Board")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "A temporary, shareable workspace for comparing artworks.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl(canonical))
			c.Response.Header().Set("HX-Push-Url", canonical)

			view := pages.StudyBoardView{
				Works:       studyBoardWorks(board.Works),
				IDs:         ids,
				HasURLState: len(ids) > 0,
				AtCapacity:  board.AtCapacity(),
			}
			var buffer bytes.Buffer
			if err := pages.StudyBoardPage(view).Render(ctx, &buffer); err != nil {
				logging.RequestLogger(app, c).Error("Render Study Board", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}
			return c.HTML(http.StatusOK, buffer.String())
		})

		return se.Next()
	})
}

func studyBoardWorks(works []boardworkflow.Artwork) []dto.StudyBoardWork {
	result := make([]dto.StudyBoardWork, len(works))
	for i, work := range works {
		result[i] = dto.StudyBoardWork{
			ID: work.ID, Title: work.Title, Artist: work.Artist, URL: work.URL,
			ImageURL: work.ImageURL, Date: work.Date, Dimensions: work.Dimensions,
			Medium: work.Medium, Location: work.Location, School: work.School,
			Form: work.Form, Type: work.Type,
		}
	}
	return result
}

func canonicalPath(ids []string) string {
	if len(ids) == 0 {
		return route
	}
	return route + "?board=" + strings.Join(ids, ",")
}

func canonicalQuery(query map[string][]string, ids []string) bool {
	if len(ids) == 0 {
		return len(query) == 0
	}
	values, ok := query["board"]
	return len(query) == 1 && ok && len(values) == 1 && values[0] == strings.Join(ids, ",")
}

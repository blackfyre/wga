package search

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const resultsPerKind = 10

func searchView(app *pocketbase.PocketBase, term string) (pages.SearchView, error) {
	artistFilter := "published = true"
	artworkFilter := "published = true && author:length > 0"
	params := dbx.Params{}
	if term != "" {
		params["query"] = term
		artistFilter += " && (name ~ {:query} || profession ~ {:query} || school.name ~ {:query})"
		artworkFilter += " && (title ~ {:query} || author.name ~ {:query} || school.name ~ {:query})"
	}

	artists, err := app.FindRecordsByFilter(constants.CollectionArtists, artistFilter, "+name", resultsPerKind, 0, params)
	if err != nil {
		return pages.SearchView{}, err
	}
	artistTotal, err := utils.CountRecordsByFilter(app, constants.CollectionArtists, artistFilter, params)
	if err != nil {
		return pages.SearchView{}, err
	}

	works, err := app.FindRecordsByFilter(constants.CollectionArtworks, artworkFilter, "+title", resultsPerKind, 0, params)
	if err != nil {
		return pages.SearchView{}, err
	}
	workTotal, err := utils.CountRecordsByFilter(app, constants.CollectionArtworks, artworkFilter, params)
	if err != nil {
		return pages.SearchView{}, err
	}

	view := pages.SearchView{Term: term, ArtistTotal: artistTotal, WorkTotal: workTotal}
	for _, artist := range artists {
		view.Artists = append(view.Artists, pages.SearchArtist{
			Name:   artist.GetString("name"),
			Dates:  utils.NormalizedBirthDeathActivity(artist),
			School: utils.RenderSchoolNames(app, artist.GetStringSlice("school")),
			Href:   url.GenerateArtistUrlFromRecord(artist),
		})
	}
	for _, work := range works {
		authorIDs := work.GetStringSlice("author")
		author, err := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
		if err != nil {
			return pages.SearchView{}, err
		}
		view.Works = append(view.Works, pages.SearchWork{
			Title:  work.GetString("title"),
			Artist: author.GetString("name"),
			Href: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId: author.Id, ArtistName: author.GetString("name"), ArtworkId: work.Id, ArtworkTitle: work.GetString("title"),
			}),
		})
	}

	return view, nil
}

func render(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	term := strings.TrimSpace(c.Request.URL.Query().Get("q"))
	view, err := searchView(app, term)
	if err != nil {
		app.Logger().Error("Global search failed", "error", err)
		return utils.ServerFaultError(c)
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Search artists and artworks in the collection.")
	var output bytes.Buffer
	if utils.IsHtmxRequest(c) && c.Request.URL.Path == "/search/results" {
		err = pages.SearchResults(view).Render(ctx, &output)
	} else {
		err = pages.SearchPage(view).Render(ctx, &output)
	}
	if err != nil {
		app.Logger().Error("Render global search", "error", err)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, output.String())
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/search", func(c *core.RequestEvent) error {
			return render(app, c)
		})
		se.Router.GET("/search/results", func(c *core.RequestEvent) error {
			return render(app, c)
		})

		return se.Next()
	})
}

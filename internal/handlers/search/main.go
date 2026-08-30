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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	resultsPerKind   = 10
	searchTracerName = "wga/search"
)

func searchView(ctx context.Context, app *pocketbase.PocketBase, term string) (view pages.SearchView, err error) {
	tracer := otel.Tracer(searchTracerName)
	ctx, span := tracer.Start(ctx, "search.lookup")
	defer func() {
		finishSearchSpan(span, err)
	}()

	artistFilter := "published = true && filing_name != '' && short_name != ''"
	artworkFilter := "published = true && author:length > 0 && author.filing_name != '' && author.short_name != ''"
	params := dbx.Params{}
	if term != "" {
		params["query"] = term
		params["query_upper"] = strings.ToUpper(term)
		artistFilter += " && (filing_name ~ {:query} || filing_name ~ {:query_upper} || profession ~ {:query} || school.name ~ {:query})"
		artworkFilter += " && (title ~ {:query} || author.filing_name ~ {:query} || school.name ~ {:query})"
	}

	_, artistSpan := tracer.Start(ctx, "search.artists")
	artists, err := app.FindRecordsByFilter(constants.CollectionArtists, artistFilter, "+filing_name", resultsPerKind, 0, params)
	if err != nil {
		finishSearchSpan(artistSpan, err)
		return pages.SearchView{}, err
	}
	artistTotal, err := utils.CountRecordsByFilter(app, constants.CollectionArtists, artistFilter, params)
	finishSearchSpan(artistSpan, err)
	if err != nil {
		return pages.SearchView{}, err
	}

	_, artworkSpan := tracer.Start(ctx, "search.artworks")
	works, err := app.FindRecordsByFilter(constants.CollectionArtworks, artworkFilter, "+title", resultsPerKind, 0, params)
	if err != nil {
		finishSearchSpan(artworkSpan, err)
		return pages.SearchView{}, err
	}
	workTotal, err := utils.CountRecordsByFilter(app, constants.CollectionArtworks, artworkFilter, params)
	finishSearchSpan(artworkSpan, err)
	if err != nil {
		return pages.SearchView{}, err
	}

	view = pages.SearchView{Term: term, ArtistTotal: artistTotal, WorkTotal: workTotal}
	for _, artist := range artists {
		view.Artists = append(view.Artists, pages.SearchArtist{
			Name:   artist.GetString("filing_name"),
			Dates:  utils.NormalizedBirthDeathActivity(artist),
			School: utils.RenderSchoolNames(app, artist.GetStringSlice("school")),
			Href:   url.GenerateArtistUrlFromRecord(artist),
		})
	}
	_, authorSpan := tracer.Start(ctx, "search.artwork_authors")
	for _, work := range works {
		authorIDs := work.GetStringSlice("author")
		author, err := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
		if err != nil {
			finishSearchSpan(authorSpan, err)
			return pages.SearchView{}, err
		}
		view.Works = append(view.Works, pages.SearchWork{
			Title:  work.GetString("title"),
			Artist: author.GetString("filing_name"),
			Href: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId: author.Id, ArtistName: author.GetString("name"), ArtworkId: work.Id, ArtworkTitle: work.GetString("title"),
			}),
		})
	}
	finishSearchSpan(authorSpan, nil)

	return view, nil
}

func finishSearchSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, "search failed")
	}
	span.End()
}

func render(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	term := strings.TrimSpace(c.Request.URL.Query().Get("q"))
	view, err := searchView(c.Request.Context(), app, term)
	if err != nil {
		app.Logger().Error("Global search failed", "error", err)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Search artists and artworks in the collection.")
	var output bytes.Buffer
	if utils.IsHtmxRequest(c) && c.Request.URL.Path == "/search/results" {
		err = pages.SearchResults(view).Render(ctx, &output)
	} else {
		err = pages.SearchPage(view).Render(ctx, &output)
	}
	if err != nil {
		app.Logger().Error("Render global search", "error", err)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
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

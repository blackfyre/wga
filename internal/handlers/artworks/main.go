package artworks

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func searchPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	fullUrl := utils.AssetUrl(c.Request.URL.String())
	pushUrl := utils.GenerateCurrentRelativePageUrl(c)
	filters := buildFilters(c)
	dualModeContext := getDualModeSearchContext(c)

	if filters.AnyFilterActive() {
		// redirect to the search results page
		return c.Redirect(http.StatusFound, buildArtworkSearchPath("/artworks/results", filters, dualModeContext))
	}

	content := dto.ArtworkSearchDTO{
		ActiveFilterValues: &dto.ArtworkSearchFilterValues{
			Title:         filters.Title,
			SchoolString:  filters.SchoolString,
			ArtFormString: filters.ArtFormString,
			ArtTypeString: filters.ArtTypeString,
			ArtistString:  filters.ArtistString,
		},
		ClearUrl:        buildArtworkSearchClearPath(dualModeContext),
		DualModeContext: dualModeContext,
		HxTarget:        "#artwork-search-results",
		Results: dto.ArtworkSearchResultDTO{
			ResultSummary: "Use the filters below, then run a search to browse matching artworks.",
		},
	}

	content.ArtFormOptions, _ = getArtFormOptions(app)
	content.ArtTypeOptions, _ = getArtTypesOptions(app)
	content.ArtSchoolOptions, _ = getArtSchoolOptions(app)
	content.ArtistNameList, _ = GetArtistNameList(app)
	content.NewFilterValues = filters.BuildFilterString()

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artworks Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, fullUrl)

	c.Response.Header().Set("HX-Push-Url", pushUrl)

	var buff bytes.Buffer

	err := pages.ArtworkSearchPage(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())

}

func search(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	limit := 16
	page := 1
	offset := 0

	queryParams := c.Request.URL.Query()

	if queryParams.Has("page") {
		err := error(nil)
		page, err = strconv.Atoi(queryParams.Get("page"))

		if err != nil {
			app.Logger().Error("Failed to parse page number", "error", err.Error())
			return utils.BadRequestError(c)
		}
	}

	offset = (page - 1) * limit

	//build filters
	filters := buildFilters(c)
	dualModeContext := getDualModeSearchContext(c)

	filterString, filterParams := filters.BuildFilter()

	records, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		filterString,
		"+title",
		limit,
		offset,
		filterParams,
	)

	if err != nil {
		app.Logger().Error("Failed to get artwork records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	// this could be replaced with a dedicated SQL query, but this is more convinient
	totalRecords, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		filterString,
		"",
		0,
		0,
		filterParams,
	)

	if err != nil {
		app.Logger().Error("Failed to count artwork records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	recordsCount := len(totalRecords)

	content := dto.ArtworkSearchDTO{
		HxTarget:        "#artwork-search-results",
		ClearUrl:        buildArtworkSearchClearPath(dualModeContext),
		DualModeContext: dualModeContext,
		Results: dto.ArtworkSearchResultDTO{
			Artworks: dto.ImageGrid{},
		},
		ActiveFilterValues: &dto.ArtworkSearchFilterValues{
			Title:         filters.Title,
			SchoolString:  filters.SchoolString,
			ArtFormString: filters.ArtFormString,
			ArtTypeString: filters.ArtTypeString,
			ArtistString:  filters.ArtistString,
		},
	}

	if dualModeContext != nil {
		content.Results.DualModeUrls = map[string]string{}
		content.Results.DualModeTarget = dualModeContext.Target
	}

	content.ArtFormOptions, _ = getArtFormOptions(app)
	content.ArtTypeOptions, _ = getArtTypesOptions(app)
	content.ArtSchoolOptions, _ = getArtSchoolOptions(app)
	content.ArtistNameList, _ = GetArtistNameList(app)
	content.NewFilterValues = filters.BuildFilterString()
	content.Results.ActiveFiltering = filters.AnyFilterActive()
	content.Results.ResultCount = recordsCount
	content.Results.ResultSummary = buildResultsSummary(recordsCount, filters.AnyFilterActive())

	artistsByID, err := getArtistsByIDs(app, records)

	if err != nil {
		app.Logger().Error("Failed to batch load artists", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	for _, v := range records {

		artistIds := v.GetStringSlice("author")

		if len(artistIds) == 0 {
			// waiting for the promised logging system by @pocketbase
			continue
		}

		artist, ok := artistsByID[artistIds[0]]

		if !ok {
			// waiting for the promised logging system by @pocketbase
			continue
		}

		imageURL := utils.AssetUrl("/assets/images/no-image.png")
		thumbURL := imageURL

		if imageName := v.GetString("image"); imageName != "" {
			imageURL = url.GenerateFileUrl(constants.CollectionArtworks, v.GetString("id"), imageName, "")
			thumbURL = url.GenerateThumbUrl(constants.CollectionArtworks, v.GetString("id"), imageName, "320x240", "")
		}

		artwork := dto.Image{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   artist.GetString("name"),
				ArtistId:     artist.GetString("id"),
				ArtworkTitle: v.GetString("title"),
				ArtworkId:    v.GetString("id"),
			}),
			Image:     imageURL,
			Thumb:     thumbURL,
			Comment:   v.GetString("comment"),
			Title:     v.GetString("title"),
			Technique: v.GetString("technique"),
			Id:        v.GetString("id"),
			Artist: dto.Artist{
				Id:   artist.GetString("id"),
				Name: artist.GetString("name"),
				Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
					ArtistId:   artist.Id,
					ArtistName: artist.GetString("name"),
				}),
				Profession: artist.GetString("profession"),
			},
		}

		content.Results.Artworks = append(content.Results.Artworks, artwork)

		if dualModeContext != nil {
			content.Results.DualModeUrls[artwork.Id] = buildDualModeArtworkURL(artwork.Url, dualModeContext)
		}
	}

	pUrl := buildArtworkSearchPath("/artworks", filters, dualModeContext)
	pHtmxUrl := buildArtworkSearchPath("/artworks/results", filters, dualModeContext)

	pagination := utils.NewPagination(recordsCount, limit, page, pUrl, "artwork-search-results", pHtmxUrl)

	content.Results.Pagination = string(pagination.Render())

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artworks Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, pHtmxUrl)

	c.Response.Header().Set("HX-Push-Url", pHtmxUrl)

	var buff bytes.Buffer

	if utils.IsHtmxRequest(c) {
		err = pages.ArtworkSearchResults(content.Results).Render(ctx, &buff)
	} else {
		err = pages.ArtworkSearchPage(content).Render(ctx, &buff)
	}

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func getDualModeSearchContext(c *core.RequestEvent) *dto.ArtworkSearchDualModeDto {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return nil
	}

	queryValues := c.Request.URL.Query()
	target := strings.TrimSpace(queryValues.Get("dual_target"))
	if target != "left" && target != "right" {
		return nil
	}

	return &dto.ArtworkSearchDualModeDto{
		LeftPath:      cmp.Or(strings.TrimSpace(queryValues.Get("dual_left")), "default"),
		RightPath:     cmp.Or(strings.TrimSpace(queryValues.Get("dual_right")), "default"),
		LeftRenderTo:  resolveDualModeSearchRenderTo("left", queryValues.Get("dual_left_render_to")),
		RightRenderTo: resolveDualModeSearchRenderTo("right", queryValues.Get("dual_right_render_to")),
		Target:        target,
	}
}

func resolveDualModeSearchRenderTo(side string, renderTo string) string {
	renderTo = strings.TrimSpace(renderTo)
	if renderTo == "left" || renderTo == "right" {
		return renderTo
	}

	if side == "left" {
		return "right"
	}

	return "left"
}

func buildArtworkSearchClearPath(dualModeContext *dto.ArtworkSearchDualModeDto) string {
	return buildArtworkSearchPath("/artworks", &filters{}, dualModeContext)
}

func buildArtworkSearchPath(basePath string, filters *filters, dualModeContext *dto.ArtworkSearchDualModeDto) string {
	if dualModeContext == nil {
		return filters.BuildPath(basePath)
	}

	queryValues := filters.queryValues()
	queryValues.Set("dual_left", dualModeContext.LeftPath)
	queryValues.Set("dual_right", dualModeContext.RightPath)
	queryValues.Set("dual_left_render_to", dualModeContext.LeftRenderTo)
	queryValues.Set("dual_right_render_to", dualModeContext.RightRenderTo)
	queryValues.Set("dual_target", dualModeContext.Target)

	return basePath + "?" + queryValues.Encode()
}

func buildDualModeArtworkURL(artworkURL string, dualModeContext *dto.ArtworkSearchDualModeDto) string {
	dualModeURL := url.GenerateDualModeUrl()
	queryValues := dualModeURL.Query()
	leftPath := dualModeContext.LeftPath
	rightPath := dualModeContext.RightPath

	if dualModeContext.Target == "left" {
		leftPath = artworkURL
	} else {
		rightPath = artworkURL
	}

	queryValues.Set("left", leftPath)
	queryValues.Set("right", rightPath)
	queryValues.Set("left_render_to", dualModeContext.LeftRenderTo)
	queryValues.Set("right_render_to", dualModeContext.RightRenderTo)
	dualModeURL.RawQuery = queryValues.Encode()

	return dualModeURL.String()
}

func buildResultsSummary(recordsCount int, hasActiveFilters bool) string {
	if !hasActiveFilters {
		if recordsCount == 1 {
			return "Showing 1 artwork from the collection."
		}

		return fmt.Sprintf("Showing %d artworks from the collection.", recordsCount)
	}

	if recordsCount == 1 {
		return "1 artwork found."
	}

	return fmt.Sprintf("%d artworks found.", recordsCount)
}

func getArtistsByIDs(app *pocketbase.PocketBase, artworks []*core.Record) (map[string]*core.Record, error) {
	artistIDs := uniqueArtistIDs(artworks)

	if len(artistIDs) == 0 {
		return map[string]*core.Record{}, nil
	}

	params := dbx.Params{}
	conditions := make([]string, 0, len(artistIDs))

	for index, artistID := range artistIDs {
		key := fmt.Sprintf("artist_id_%d", index)
		conditions = append(conditions, fmt.Sprintf("id = {:%s}", key))
		params[key] = artistID
	}

	artists, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		strings.Join(conditions, " || "),
		"+name",
		0,
		0,
		params,
	)

	if err != nil {
		return nil, err
	}

	artistsByID := make(map[string]*core.Record, len(artists))

	for _, artist := range artists {
		artistsByID[artist.Id] = artist
	}

	return artistsByID, nil
}

func uniqueArtistIDs(artworks []*core.Record) []string {
	seen := map[string]struct{}{}
	artistIDs := make([]string, 0, len(artworks))

	for _, artwork := range artworks {
		for _, artistID := range artwork.GetStringSlice("author") {
			if _, exists := seen[artistID]; exists {
				continue
			}

			seen[artistID] = struct{}{}
			artistIDs = append(artistIDs, artistID)
		}
	}

	return artistIDs
}

// RegisterArtworksHandlers registers search handlers to the given PocketBase app.
func RegisterArtworksHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/artworks", func(c *core.RequestEvent) error {
			return searchPage(app, c)
		})

		se.Router.GET("/artworks/results", func(c *core.RequestEvent) error {
			return search(app, c)
		})
		return se.Next()
	})
}

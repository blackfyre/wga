package artworks

import (
	"bytes"
	"cmp"
	"fmt"
	"net/http"
	neturl "net/url"
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

const artworkSearchPageSize = 16

func searchPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	return search(app, c)
}

func search(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	page := 1

	queryParams := c.Request.URL.Query()
	if queryParams.Has("page") {
		parsed, err := strconv.Atoi(queryParams.Get("page"))
		if err != nil || parsed < 1 {
			app.Logger().Error("Invalid page number", "page", queryParams.Get("page"), "error", err)
			return utils.BadRequestError(c)
		}
		page = parsed
	}

	view, canonical, err := buildArtworkSearchView(app, queryParams, page, artworkSearchPageSize)
	if err != nil {
		app.Logger().Error("Failed to build artwork search", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Artworks Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Search the collection by title, artist, school, form, type, and technique.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, canonical)

	c.Response.Header().Set("HX-Push-Url", artworkSearchPushURL(c.Request.URL.Path, canonical))

	var buff bytes.Buffer

	if utils.IsHtmxRequest(c) && c.Request.URL.Path == "/artworks/results" {
		err = pages.ArtworkSearchResults(view.Results).Render(ctx, &buff)
	} else {
		err = pages.ArtworkSearchPage(view).Render(ctx, &buff)
	}

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

// buildArtworkSearchView parses the request state, loads the bounded result
// page, and assembles the page-owned view plus the canonical /artworks URL.
func buildArtworkSearchView(app *pocketbase.PocketBase, values neturl.Values, page int, limit int) (pages.ArtworkSearchView, string, error) {
	filters := buildFilters(values)
	dualModeContext := getDualModeSearchContext(values)

	recordsCount, err := countArtworkRecords(app, filters)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}

	pageCount := (recordsCount + limit - 1) / limit
	if pageCount == 0 {
		page = 1
	} else if page > pageCount {
		page = pageCount
	}
	filters.Page = strconv.Itoa(page)
	offset := (page - 1) * limit

	records, err := listArtworkRecords(app, filters, limit, offset)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}

	artFormOptions, err := getArtFormOptions(app)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}
	artTypeOptions, err := getArtTypesOptions(app)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}
	artSchoolOptions, err := getArtSchoolOptions(app)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}
	artPeriodOptions, err := getArtPeriodOptions(app)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}
	locationOptions, err := getLocationOptions(app)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}
	results, err := buildArtworkSearchResults(app, filters, dualModeContext, records, recordsCount, page, limit)
	if err != nil {
		return pages.ArtworkSearchView{}, "", err
	}

	view := pages.ArtworkSearchView{
		NameField: dto.Field{
			ID:          "artwork-query",
			Name:        "q",
			Label:       "TITLE OR ARTIST",
			Type:        "search",
			Value:       filters.Query,
			Placeholder: "e.g. milkmaid",
		},
		TechniqueField: dto.Field{
			ID:          "artwork-technique",
			Name:        "technique",
			Label:       "TECHNIQUE",
			Type:        "search",
			Value:       filters.TechniqueString,
			Placeholder: "e.g. oil on canvas",
		},
		SchoolGroup:     buildChipGroup("SCHOOL", "art_school", artSchoolOptions, filters.SchoolString),
		FormGroup:       buildChipGroup("FORM", "art_form", artFormOptions, filters.ArtFormString),
		TypeGroup:       buildChipGroup("TYPE", "art_type", artTypeOptions, filters.ArtTypeString),
		PeriodGroup:     buildFilterGroup("PERIOD", "period", artPeriodOptions, filters.PeriodString),
		LocationGroup:   buildFilterGroup("LOCATION", "location", locationOptions, filters.LocationString),
		YearFrom:        cmp.Or(filters.YearFrom, "200"),
		YearTo:          cmp.Or(filters.YearTo, "1900"),
		ClearUrl:        buildArtworkSearchClearPath(dualModeContext),
		DualModeContext: dualModeContext,
		HxTarget:        "#artwork-search-results",
		Results:         results,
	}

	canonical := buildArtworkSearchPath("/artworks", filters, dualModeContext)

	return view, canonical, nil
}

func buildArtworkSearchResults(app *pocketbase.PocketBase, filters *filters, dualModeContext *pages.ArtworkSearchDualMode, records []*core.Record, recordsCount int, page int, limit int) (pages.ArtworkSearchResultsView, error) {
	results := pages.ArtworkSearchResultsView{
		ActiveFiltering: filters.AnyFilterActive(),
		ResultCount:     recordsCount,
		Artworks:        dto.ImageGrid{},
		View:            filters.View,
		GridUrl:         buildArtworkSearchPath("/artworks/results", filters.forView("grid"), dualModeContext),
		ListUrl:         buildArtworkSearchPath("/artworks/results", filters.forView("list"), dualModeContext),
		ResetUrl:        buildArtworkSearchClearPath(dualModeContext),
		SortOptions:     buildSortOptions(filters, dualModeContext),
		SortDirLabel:    filters.sortDirLabel(),
		SortToggleUrl:   buildArtworkSearchPath("/artworks/results", filters.forSortDir(flipSortDir(filters.SortDir)), dualModeContext),
	}

	if dualModeContext != nil {
		results.DualModeUrls = map[string]string{}
		results.DualModeTarget = dualModeContext.Target
	}

	artistsByID, err := getArtistsByIDs(app, records)
	if err != nil {
		return pages.ArtworkSearchResultsView{}, err
	}

	for _, v := range records {
		artistIds := v.GetStringSlice("author")

		if len(artistIds) == 0 {
			continue
		}

		artist, ok := artistsByID[artistIds[0]]
		if !ok {
			continue
		}

		imageURL := utils.AssetUrl("/assets/images/no-image.png")

		if imageName := v.GetString("image"); imageName != "" {
			imageURL = url.GenerateArtworkImageURL(v, artworkSearchThumbnail(filters.View, dualModeContext != nil), "")
		}

		artwork := dto.Image{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   artist.GetString("name"),
				ArtistId:     artist.GetString("id"),
				ArtworkTitle: v.GetString("title"),
				ArtworkId:    v.GetString("id"),
			}),
			Image:     imageURL,
			Thumb:     imageURL,
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

		results.Artworks = append(results.Artworks, artwork)

		if dualModeContext != nil {
			results.DualModeUrls[artwork.Id] = buildDualModeArtworkURL(artwork.Url, dualModeContext)
		}
	}

	pUrl := buildArtworkSearchPath("/artworks", filters, dualModeContext)
	pHtmxUrl := buildArtworkSearchPath("/artworks/results", filters, dualModeContext)

	pagination := utils.NewPagination(recordsCount, limit, page, pUrl, "artwork-search-results", pHtmxUrl)
	results.Pagination = string(pagination.Render())

	return results, nil
}

func artworkSearchPushURL(requestPath string, canonical string) string {
	if requestPath == "/artworks/results" {
		return "/artworks/results" + strings.TrimPrefix(canonical, "/artworks")
	}

	return canonical
}

func artworkSearchThumbnail(view string, dualMode bool) url.DeliveryProfile {
	if view == "list" && !dualMode {
		return url.DeliveryProfileSearchRow
	}

	return url.DeliveryProfileCardAndArtistIndex
}

func getDualModeSearchContext(values neturl.Values) *pages.ArtworkSearchDualMode {
	target := strings.TrimSpace(values.Get("dual_target"))
	if target != "left" && target != "right" {
		return nil
	}

	return &pages.ArtworkSearchDualMode{
		LeftPath:      cmp.Or(strings.TrimSpace(values.Get("dual_left")), "default"),
		RightPath:     cmp.Or(strings.TrimSpace(values.Get("dual_right")), "default"),
		LeftRenderTo:  resolveDualModeSearchRenderTo("left", values.Get("dual_left_render_to")),
		RightRenderTo: resolveDualModeSearchRenderTo("right", values.Get("dual_right_render_to")),
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

func buildArtworkSearchClearPath(dualModeContext *pages.ArtworkSearchDualMode) string {
	return buildArtworkSearchPath("/artworks", &filters{}, dualModeContext)
}

func buildArtworkSearchPath(basePath string, filters *filters, dualModeContext *pages.ArtworkSearchDualMode) string {
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

func buildDualModeArtworkURL(artworkURL string, dualModeContext *pages.ArtworkSearchDualMode) string {
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

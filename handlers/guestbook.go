package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/handlers/guestbook"
	"github.com/blackfyre/wga/utils"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

type SearchSettings struct {
	searchExpression        string
	searchExpressionPresent bool
	filter                  string
}

type GbData struct {
	Title         string
	FirstContent  string
	SecondContent string
	MainContent   []GuestBookMessagePrepared
	LatestEntries string
}

type GbPreparedData struct {
	Title                   string
	FirstContent            string
	SecondContent           string
	MainContent             []GuestBookMessagePrepared
	LatestEntries           string
	CalendarYears           [][]string
	SearchExpressionPresent bool
	SearchExpression        string
}

type GuestBookMessage struct {
	Name          string `json:"name" form:"name" query:"name" validate:"required"`
	Email         string `json:"email" form:"email" query:"email" validate:"required"`
	Location      string `json:"location" form:"location" query:"location" validate:"required"`
	Message       string `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName  string `json:"honey_pot_name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" query:"honey_pot_email"`
}

type GuestBookMessagePrepared struct {
	Message  string
	Name     string
	Location string
	Created  string
}

// registerGuestbookHandlers registers the handlers for the guestbook routes.
// It takes an instance of pocketbase.PocketBase as input and adds the necessary
// route handlers to the app's router. The handlers include GET and POST methods
// for displaying and adding messages to the guestbook.
func registerGuestbookHandlers(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/guestbook", func(c echo.Context) error {
			return guestbook.EntriesHandler(app, c)
		})

		e.Router.GET("/guestbook/addMessage", func(c echo.Context) error {
			return guestbook.StoreEntryViewHandler(app, c)
		})

		e.Router.POST("/guestbook/addMessage", func(c echo.Context) error {
			return guestbook.EntriesHandler(app, c)
		})

		return nil

	})
}

func loadGuestbook(c echo.Context, app *pocketbase.PocketBase, url string) (string, bool, error) {
	confirmedHtmxRequest := utils.IsHtmxRequest(c)
	searchSettings := gbProcessRequest(c)

	cacheKey := gbSetCacheSettings(confirmedHtmxRequest, searchSettings)
	shouldReturn, err := gbIsCached(app, cacheKey, c)
	if shouldReturn {
		return "", true, err
	}

	setUrl(c, url)

	data, err := gbGetData(app, searchSettings)

	if err != nil {

		return "", true, apis.NewNotFoundError("", err)
	}

	html, err := gbRender(confirmedHtmxRequest, data)

	if err != nil {

		return "", true, apis.NewNotFoundError("", err)
	}
	return html, false, nil
}

func setUrl(c echo.Context, url string) {
	if url != "" {
		c.Response().Header().Set("HX-Push-Url", url)
	} else {
		currentUrl := c.Request().URL.String()
		c.Response().Header().Set("HX-Push-Url", currentUrl)
	}
}

// gbProcessRequest is a function that processes the request and returns a SearchSettings struct.
// It takes an echo.Context as a parameter and extracts the search expression from the query parameters.
// If the search expression is present, it updates the filter to include the search expression.
// The function returns a SearchSettings struct containing the search expression, a flag indicating if the search expression is present, and the filter.
func gbProcessRequest(c echo.Context) SearchSettings {
	searchExpression := strconv.Itoa(time.Now().Year())
	searchExpressionPresent := false
	filter := "id != null"

	if c.QueryParam("q") != "" {
		searchExpression = c.QueryParam("q")
	}

	if c.QueryParams().Has("q") {
		searchExpressionPresent = true
	}

	if searchExpression != "" {
		filter = filter + " && created ~ {:searchExpression}"
	}

	searchSettings := SearchSettings{
		searchExpression:        searchExpression,
		searchExpressionPresent: searchExpressionPresent,
		filter:                  filter,
	}

	return searchSettings
}

// gbSetCacheSettings generates a cache key based on the provided parameters.
// The cache key is used to store and retrieve data from the cache.
//
// Parameters:
// - confirmedHtmxRequest: A boolean indicating whether the request is confirmed as an htmx request.
// - searchSettings: The search settings used to generate the cache key.
//
// Returns:
// - The generated cache key as a string.
func gbSetCacheSettings(confirmedHtmxRequest bool, searchSettings SearchSettings) string {
	cacheKey := "guestbook:" + ":" + searchSettings.searchExpression

	if confirmedHtmxRequest {
		cacheKey = cacheKey + ":htmx"
	}

	if searchSettings.searchExpressionPresent {
		cacheKey = cacheKey + ":search"
	}
	return cacheKey
}

// gbIsCached checks if the given cacheKey exists in the app's store.
// If the cacheKey exists, it retrieves the HTML content from the store and returns true along with the HTML response.
// If the cacheKey does not exist, it returns false and a nil error.
func gbIsCached(app *pocketbase.PocketBase, cacheKey string, c echo.Context) (bool, error) {
	if app.Store().Has(cacheKey) {
		html := app.Store().Get(cacheKey).(string)
		return true, c.HTML(http.StatusOK, html)
	}
	return false, nil
}

// gbGetData retrieves data for the guestbook based on the provided search settings.
// It calls various functions to fetch the guestbook title, text content, main content, and latest entries.
// The retrieved data is then prepared for rendering and returned along with any error encountered.
func gbGetData(app *pocketbase.PocketBase, searchSettings SearchSettings) (GbPreparedData, error) {
	guestBookTitle, err := gbGetGuestbookTextContent(app, "guestbook")

	if err != nil {
		fmt.Println(err)
	}

	guestBookTextFirst, err := gbGetGuestbookTextContent(app, "thankYou")

	if err != nil {
		fmt.Println(err)
	}

	guestBookTextSecond, err := gbGetGuestbookTextContent(app, "guestbookPlayMusicLink")

	if err != nil {
		fmt.Println(err)
	}

	guestBookContent, err := gbGetGuestbookContent(app, searchSettings.filter, searchSettings.searchExpression)

	if err != nil {
		fmt.Println(err)
	}

	latestEntries, err := gbGetGuestbookTextContent(app, "latestEntries")

	if err != nil {
		fmt.Println(err)
	}

	rawData := GbData{
		Title:         guestBookTitle,
		FirstContent:  guestBookTextFirst,
		SecondContent: guestBookTextSecond,
		MainContent:   guestBookContent,
		LatestEntries: latestEntries,
	}

	data := gbPrepareDataForRender(rawData, searchSettings)

	return data, err
}

// gbPrepareDataForRender prepares the data for rendering the guestbook page.
// It takes the GbData and SearchSettings as input parameters and returns the prepared data of type GbPreparedData.
func gbPrepareDataForRender(data GbData, searchSettings SearchSettings) GbPreparedData {
	// Retrieve the years for the calendar
	years := gbGetYears()

	// Create a new instance of GbPreparedData and populate its fields
	preparedData := GbPreparedData{
		Title:                   data.Title,
		FirstContent:            data.FirstContent,
		SecondContent:           data.SecondContent,
		MainContent:             data.MainContent,
		CalendarYears:           years,
		SearchExpressionPresent: searchSettings.searchExpressionPresent,
		SearchExpression:        searchSettings.searchExpression,
		LatestEntries:           data.LatestEntries,
	}

	return preparedData
}

// gbRender is a function that renders the guestbook content based on the provided data.
// It takes a boolean flag 'confirmedHtmxRequest' to indicate whether the request is confirmed as an Htmx request.
// The 'data' parameter is of type GbPreparedData, which contains the necessary data for rendering the guestbook content.
// The function returns a string representing the rendered HTML content and an error if any occurred during the rendering process.
func gbRender(confirmedHtmxRequest bool, data GbPreparedData) (string, error) {
	dataMap, shouldReturn, html, err := gbGeneralizer(data)
	if shouldReturn {
		return html, err
	}

	dto := assets.NewRenderData(nil)

	for k, v := range dataMap {
		dto[k] = v
	}

	return assets.Render(assets.Renderable{
		IsHtmx: confirmedHtmxRequest,
		Block:  "guestbook:content",
		Data:   dto,
	})
}

// gbGetGuestbookTextContent retrieves the text content for a guestbook entry.
// It takes an app object of type *pocketbase.PocketBase and a content string as input.
// It returns the retrieved text content as a string and an error if any.
func gbGetGuestbookTextContent(app *pocketbase.PocketBase, content string) (string, error) {
	// Format the content string
	strContent := fmt.Sprintf("strings:%s", content)

	// Check if the content string exists in the app store
	found := app.Store().Has(strContent)

	if found {
		// If the content string exists, retrieve and return it
		return app.Store().Get(strContent).(string), nil
	}

	// If the content string does not exist in the app store, find the first record by data
	record, err := app.Dao().FindFirstRecordByData("strings", "name", content)

	if err != nil {
		app.Logger().Error("Failed to find guestbook record", err)
		return "", err
	}

	// Get the content from the record
	result := record.Get("content")

	// Set the content in the app store
	app.Store().Set(strContent, result.(string))

	// Return the retrieved content
	return result.(string), nil
}

// gbGetGuestbookContent retrieves the guestbook content based on the provided filter and search expression.
// It returns a slice of GuestBookMessagePrepared and an error if any.
func gbGetGuestbookContent(app *pocketbase.PocketBase, filter string, searchExpression string) ([]GuestBookMessagePrepared, error) {
	records, err := app.Dao().FindRecordsByFilter(
		"guestbook",
		filter,
		"-created",
		0,
		0,
		dbx.Params{
			"searchExpression": searchExpression,
		},
	)

	if err != nil {
		app.Logger().Error("Failed to find guestbook records", err)
		return nil, apis.NewBadRequestError("Invalid something", err)
	}

	preRendered := []GuestBookMessagePrepared{}

	for _, m := range records {

		createDateString := m.GetString("created")
		createTime, err := time.Parse("2006-01-02 15:04:05.000Z", createDateString)
		if err != nil {
			app.Logger().Error("Failed to parse date", err)
			return nil, err
		}
		formattedDateString := createTime.Format("January 2, 2006")

		row := GuestBookMessagePrepared{
			Message:  m.GetString("message"),
			Name:     m.GetString("name"),
			Location: m.GetString("location"),
			Created:  formattedDateString,
		}

		preRendered = append(preRendered, row)
	}

	return preRendered, err
}

// gbGetYears returns a 2D slice of strings representing grouped years.
// The function generates a list of years from 1997 to the current year.
// The years are then grouped into sub-slices of 5 years each.
// The order of the grouped years is reversed before returning the result.
func gbGetYears() [][]string {
	years := []string{}
	thisYear := time.Now().Year()

	for i := 1997; i <= thisYear; i++ {
		years = append(years, fmt.Sprintf("%d", i))
	}

	// group by 5
	groupedYears := [][]string{}
	for i := 0; i < len(years); i += 5 {
		end := i + 5
		if end > len(years) {
			end = len(years)
		}
		groupedYears = append(groupedYears, years[i:end])
	}

	// reverse order
	for i, j := 0, len(groupedYears)-1; i < j; i, j = i+1, j-1 {
		groupedYears[i], groupedYears[j] = groupedYears[j], groupedYears[i]
	}
	return groupedYears
}

// gbGeneralizer is a function that takes a GbPreparedData struct as input and returns a map[string]interface{},
// a boolean value, a string, and an error. It converts the input data into a JSON string, then unmarshals it
// into a map[string]interface{}. If successful, it returns the data map, false, an empty string, and nil error.
// Otherwise, it returns nil, true, an empty string, and the encountered error.
func gbGeneralizer(data GbPreparedData) (map[string]interface{}, bool, string, error) {
	dataMap := make(map[string]interface{})
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, true, "", err
	}
	err = json.Unmarshal(jsonData, &dataMap)
	if err != nil {
		return nil, true, "", err
	}
	return dataMap, false, "", nil
}

// gbAddMessageRender is a function that renders the HTML content for adding a message to the guestbook.
// It takes a boolean parameter, confirmedHtmxRequest, which indicates whether the request is confirmed as an Htmx request.
// If confirmedHtmxRequest is true, it renders a block of HTML content using assets.RenderBlock.
// If confirmedHtmxRequest is false, it renders a page of HTML content using assets.RenderPage.
// The function returns the rendered HTML content as a string and any error that occurred during rendering.
func gbAddMessageRender(confirmedHtmxRequest bool) (string, error) {
	dataMap := make(map[string]any)

	// TODO: delete unnecessary dataMap
	if confirmedHtmxRequest {
		html, err := assets.RenderBlock("addMessage:content", dataMap)
		return html, err
	} else {
		html, err := assets.RenderPage("guestbook/addMessage", dataMap)
		return html, err
	}
}

// addGuestbookMessageToDB adds a guestbook message to the database.
// It takes an app object of type *pocketbase.PocketBase and a content object of type GuestBookMessage as parameters.
// It returns an error if there is any issue with adding the message to the database.
func addGuestbookMessageToDB(app *pocketbase.PocketBase, content GuestBookMessage) error {
	collection, err := app.Dao().FindCollectionByNameOrId("guestbook")
	if err != nil {
		fmt.Println(err)
	}

	record := models.NewRecord(collection)
	form := forms.NewRecordUpsert(app, record)

	messageMap, err := addGuestbookMessageGeneralizer(content)
	if err != nil {
		fmt.Println(err)
	}

	form.LoadData(messageMap)

	if err := form.Submit(); err != nil {
		fmt.Println(err)
	}

	return err
}

// addGuestbookMessageGeneralizer is a function that takes a GuestBookMessage as input and returns a map[string]any and an error.
// It converts the GuestBookMessage into a JSON string, then unmarshals the JSON string into a map[string]any.
// If there is an error during the conversion or unmarshaling process, it returns the error.
func addGuestbookMessageGeneralizer(data GuestBookMessage) (map[string]any, error) {
	messageMap := make(map[string]any)
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(jsonData, &messageMap)
	if err != nil {
		fmt.Println(err)
	}
	return messageMap, err
}

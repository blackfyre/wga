package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"blackfyre.ninja/wga/assets"

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
	Name         	 	 string   `json:"name" form:"name" query:"name" validate:"required"`
	Email          		 string   `json:"email" form:"email" query:"email" validate:"required"`
	Location     		 string   `json:"location" form:"location" query:"location" validate:"required"`
	Message      		 string   `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName         string   `json:"honey_pot_name" query:"honey_pot_name"`
	HoneyPotEmail        string   `json:"honey_pot_email" query:"honey_pot_email"`
}

type GuestBookMessagePrepared struct {
	Message  string
	Name     string
	Location string
	Created  string
}


func registerGuestbookHandlers(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/guestbook", func(c echo.Context) error {
			url := ""
			html, shouldReturn, err := loadGuestbook(c, app, url)
			if shouldReturn {
				return err
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.GET("/guestbook/addMessage", func(c echo.Context) error {
			confirmedHtmxRequest := isHtmxRequest(c)
			url := ""
			cacheKey := gbAddMessageSetCacheSettings(confirmedHtmxRequest)
			shouldReturn, err := gbAddMessageIsCached(app, cacheKey, c)
			if shouldReturn {
				return err
			}

			setUrl(c, url)

			html, err := gbAddMessageRender(confirmedHtmxRequest)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.POST("/guestbook/addMessage", func(c echo.Context) error {
			data := apis.RequestInfo(c).Data

			if err := c.Bind(&data); err != nil {
				sendToastMessage("Failed to create message, please try again later.", "is-danger", true, c)
				return apis.NewBadRequestError("Failed to parse form data", err)
			}

			guestBookMessage := GuestBookMessage{
				Name:       	  c.FormValue("sender_name"),
				Email:      	  c.FormValue("sender_email"),
				Location:   	  c.FormValue("location"),
				Message:    	  c.FormValue("message"),
				HoneyPotName: 	  c.FormValue("name"),
				HoneyPotEmail: 	  c.FormValue("email"),
			}

			if guestBookMessage.HoneyPotEmail != "" || guestBookMessage.HoneyPotName != "" {
				// this is probably a bot
				//TODO: use the new generic logger in pb to log this event
				sendToastMessage("Failed to create message, please try again later.", "is-danger", true, c)
				return c.NoContent(204)
			}

			err := addGuestbookMessageToDB(app, guestBookMessage)

			if err != nil {
				return apis.NewBadRequestError("Failed to add message to database", err)
			}

			sendToastMessage("Message added successfully", "is-success", true, c)

			html, shouldReturn, err := loadGuestbook(c, app, "/guestbook")
			if shouldReturn {
				return err
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil

	})
}

func loadGuestbook(c echo.Context, app *pocketbase.PocketBase, url string) (string, bool, error) {
	confirmedHtmxRequest := isHtmxRequest(c)
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

func gbProcessRequest(c echo.Context) SearchSettings {
	searchExpression := ""
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

func gbIsCached(app *pocketbase.PocketBase, cacheKey string, c echo.Context) (bool, error) {
	if app.Cache().Has(cacheKey) {
		html := app.Cache().Get(cacheKey).(string)
		return true, c.HTML(http.StatusOK, html)
	}
	return false, nil
}

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

func gbPrepareDataForRender(data GbData, searchSettings SearchSettings) GbPreparedData {
	years := gbGetYears()

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

func gbRender(confirmedHtmxRequest bool, data GbPreparedData) (string, error) {
	dataMap, shouldReturn, html, err := gbGeneralizer(data)
	if shouldReturn {
		return html, err
	}

	if confirmedHtmxRequest {
		blockToRender := "guestbook:content"

		html, err := assets.RenderBlock(blockToRender, dataMap)
		return html, err
	} else {
		html, err := assets.RenderPage("guestbook", dataMap)
		return html, err
	}
}

func gbGetGuestbookTextContent(app *pocketbase.PocketBase, content string) (string, error) {
	strContent := fmt.Sprintf("strings:%s", content)

	found := app.Cache().Has(strContent)

	if found {
		return app.Cache().Get(strContent).(string), nil
	}

	record, err := app.Dao().FindFirstRecordByData("strings", "name", content)

	if err != nil {
		log.Println(err)
		return "", err
	}

	result := record.Get("content")

	app.Cache().Set(strContent, result.(string))

	return result.(string), nil
}

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
		return nil, apis.NewBadRequestError("Invalid something", err)
	}

	preRendered := []GuestBookMessagePrepared{}

	for _, m := range records {

		createDateString := m.GetString("created")
		createTime, err := time.Parse("2006-01-02 15:04:05.000Z", createDateString)
		if err != nil {
			fmt.Println("Error parsing the date:", err)
			return nil, err
		}
		formattedDateString := createTime.Format("January 2, 2006")

		row := GuestBookMessagePrepared{
			Message:  m.GetString("message"),
			Name:    m.GetString("name"),
			Location: m.GetString("location"),
			Created:  formattedDateString,
		}

		preRendered = append(preRendered, row)
	}

	return preRendered, err
}

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

func gbAddMessageSetCacheSettings(confirmedHtmxRequest bool) string {
	cacheKey := "addGuestbookMessage"

	return cacheKey
}

func gbAddMessageIsCached(app *pocketbase.PocketBase, cacheKey string, c echo.Context) (bool, error) {
	if app.Cache().Has(cacheKey) {
		html := app.Cache().Get(cacheKey).(string)
		return true, c.HTML(http.StatusOK, html)
	}
	return false, nil
}

func gbAddMessageRender(confirmedHtmxRequest bool) (string, error) {
	dataMap := make(map[string]any)

	// TODO: delete unnecessary dataMap
	html, err := assets.RenderPage("guestbook/addMessage", dataMap)

	if err != nil {
		return "", apis.NewBadRequestError("", err)
	}

	return html, err
}

func addGuestbookMessageToDB(app *pocketbase.PocketBase, content GuestBookMessage) (error) {
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

	if err := form.Submit()
	err != nil {
		fmt.Println(err)
	}

	return err
}

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

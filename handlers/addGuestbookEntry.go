package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

type GuestBookMessage struct {
	Name         string `json:"name" form:"name"`
	Email        string `json:"email" form:"email"`
	Location     string `json:"location" form:"location"`
	Message      string `json:"message" form:"message"`
}

func registerAddGuestbookMessage(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/addGuestbookMessage", func(c echo.Context) error {
			confirmedHtmxRequest := isHtmxRequest(c)
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

			cacheKey := gbAddMessageSetCacheSettings(confirmedHtmxRequest)
			shouldReturn, returnValue := gbAddMessageIsCached(app, cacheKey, c)
			if shouldReturn {
				return returnValue
			}

			html, err := gbAddMessageRender(confirmedHtmxRequest)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.POST("/addGuestbookMessage", func(c echo.Context) error {
			data := apis.RequestInfo(c).Data

			guestBookMessage := GuestBookMessage{
				Name:       c.FormValue("name"),
				Email:      c.FormValue("email"),
				Location:   c.FormValue("location"),
				Message:    c.FormValue("message"),
			}

			if err := c.Bind(&data); err != nil {
				return apis.NewBadRequestError("Failed to read request data", err)
			}

			err := addGuestbookMessageToDB(app, guestBookMessage)

			if err != nil {
				return apis.NewBadRequestError("Failed to add message to database", err)
			}
			
			return c.NoContent(204)
		})

		return nil

	})
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
	if confirmedHtmxRequest {
		blockToRender := "addGuestbookMessage:content"

		html, err := renderBlock(blockToRender, dataMap)
		return html, err
	} else {
		html, err := renderPage("addGuestbookMessage", dataMap)
		return html, err
	}
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

package guestbook

import (
	"context"
	"fmt"
	"time"

	"github.com/blackfyre/wga/assets/templ/pages"
	"github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"

	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
)

type GuestBookMessage struct {
	Name          string `json:"name" form:"name" query:"name" validate:"required"`
	Email         string `json:"email" form:"email" query:"email" validate:"required"`
	Location      string `json:"location" form:"location" query:"location" validate:"required"`
	Message       string `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName  string `json:"honey_pot_name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" query:"honey_pot_email"`
}

func yearOptions() []string {
	years := []string{}

	for i := time.Now().Year(); i >= 1997; i-- {
		years = append(years, fmt.Sprintf("%d", i))
	}

	return years
}

func EntriesHandler(app *pocketbase.PocketBase, c echo.Context) error {

	fullUrl := c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()
	year := c.QueryParamDefault("year", fmt.Sprintf("%d", time.Now().Year()))

	entries, err := models.FindEntriesForYear(app.Dao(), year)

	if err != nil {
		app.Logger().Error("Failed to get guestbook entries", "error", err)
		return utils.ServerFaultError(c)
	}

	content := pages.GuestbookView{
		SelectedYear: year,
		YearOptions:  yearOptions(),
		Entries:      entries,
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Guestbook")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "This is the guestbook of the Web Gallery of Art. Please feel free to leave a message.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

	if utils.IsHtmxRequest(c) {
		c.Response().Header().Set("HX-Push-Url", fullUrl)
		err = pages.GuestbookBlock(content).Render(ctx, c.Response().Writer)
	} else {
		err = pages.GuestbookPage(content).Render(ctx, c.Response().Writer)
	}

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return nil
}

func StoreEntryViewHandler(app *pocketbase.PocketBase, c echo.Context) error {

	if !utils.IsHtmxRequest(c) {
		return utils.ServerFaultError(c)
	}

	return c.String(200, "Hello, World!")
}

func StoreEntryHandler(app *pocketbase.PocketBase, c echo.Context) error {

	if !utils.IsHtmxRequest(c) {
		return utils.ServerFaultError(c)
	}

	data := apis.RequestInfo(c).Data

	if err := c.Bind(&data); err != nil {
		utils.SendToastMessage("Failed to create message, please try again later.", "is-danger", true, c)
		return apis.NewBadRequestError("Failed to parse form data", err)
	}

	guestBookMessage := GuestBookMessage{
		Name:          c.FormValue("sender_name"),
		Email:         c.FormValue("sender_email"),
		Location:      c.FormValue("location"),
		Message:       c.FormValue("message"),
		HoneyPotName:  c.FormValue("name"),
		HoneyPotEmail: c.FormValue("email"),
	}

	if guestBookMessage.HoneyPotEmail != "" || guestBookMessage.HoneyPotName != "" {
		// this is probably a bot
		app.Logger().Error("Guestbook HoneyPot triggered", "ip", c.RealIP())
		utils.SendToastMessage("Failed to create message, please try again later.", "is-danger", true, c)
		return c.NoContent(204)
	}

	// err := addGuestbookMessageToDB(app, guestBookMessage)

	// if err != nil {
	// 	return apis.NewBadRequestError("Failed to add message to database", err)
	// }

	utils.SendToastMessage("Message added successfully", "is-success", true, c)

	return nil
}

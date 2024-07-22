package guestbook

import (
	"context"
	"fmt"
	"time"

	"github.com/blackfyre/wga/assets/templ/pages"
	wgaModels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"

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
	var years []string

	for i := time.Now().Year(); i >= 1997; i-- {
		years = append(years, fmt.Sprintf("%d", i))
	}

	return years
}

func EntriesHandler(app *pocketbase.PocketBase, c echo.Context) error {

	fullUrl := c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()
	year := c.QueryParamDefault("year", fmt.Sprintf("%d", time.Now().Year()))

	entries, err := wgaModels.FindEntriesForYear(app.Dao(), year)

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

	c.Response().Header().Set("HX-Push-Url", fullUrl)
	err = pages.GuestbookPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return nil
}

func StoreEntryViewHandler(app *pocketbase.PocketBase, c echo.Context) error {

	err := pages.GuestbookEntryForm().Render(context.Background(), c.Response().Writer)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return nil
}

func StoreEntryHandler(app *pocketbase.PocketBase, c echo.Context) error {

	data := apis.RequestInfo(c).Data

	if err := c.Bind(&data); err != nil {
		utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
		return apis.NewBadRequestError("Failed to parse form data", err)
	}

	postData := GuestBookMessage{
		Name:          c.FormValue("sender_name"),
		Email:         c.FormValue("sender_email"),
		Location:      c.FormValue("location"),
		Message:       c.FormValue("message"),
		HoneyPotName:  c.FormValue("name"),
		HoneyPotEmail: c.FormValue("email"),
	}

	if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
		// this is probably a bot
		app.Logger().Error("Guestbook HoneyPot triggered", "ip", c.RealIP())
		utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
		return c.NoContent(204)
	}

	collection, err := app.Dao().FindCollectionByNameOrId("Guestbook")
	if err != nil {
		app.Logger().Error("Database table not found", err)
		utils.SendToastMessage("Something went wrong!", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	record := models.NewRecord(collection)

	form := forms.NewRecordUpsert(app, record)

	form.LoadData(map[string]any{
		"email":    postData.Email,
		"name":     postData.Name,
		"message":  postData.Message,
		"location": postData.Location,
	})

	if err := form.Submit(); err != nil {

		err := pages.GuestbookEntryForm().Render(context.Background(), c.Response().Writer)

		if err != nil {
			app.Logger().Error("Failed to render the guestbook entry form after form submission error", err)
			return utils.ServerFaultError(c)
		}

		app.Logger().Error("Failed to store the entry", err)

		utils.SendToastMessage("Failed to store the entry", "error", false, c, "")

		return err
	}

	utils.SendToastMessage("Message added successfully", "success", true, c, "guestbook-updated")

	return nil
}

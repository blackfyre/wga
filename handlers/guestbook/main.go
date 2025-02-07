package guestbook

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
)

type GuestBookMessage struct {
	Name          string `json:"name" form:"name" query:"name" validate:"required"`
	Email         string `json:"email" form:"email" query:"email" validate:"required"`
	Location      string `json:"location" form:"location" query:"location" validate:"required"`
	Message       string `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName  string `json:"honey_pot_name" form:"honey_pot_name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" form:"honey_pot_email" query:"honey_pot_email"`
}

func yearOptions() []string {
	var years []string

	for i := time.Now().Year(); i >= 1997; i-- {
		years = append(years, fmt.Sprintf("%d", i))
	}

	return years
}

func convertRawEntriesToGuestbookEntries(entries []*core.Record) []dto.GuestbookEntry {
	var guestbookEntries []dto.GuestbookEntry

	for _, entry := range entries {
		guestbookEntries = append(guestbookEntries, dto.GuestbookEntry{
			Name:     entry.GetString("name"),
			Email:    entry.GetString("email"),
			Location: entry.GetString("location"),
			Message:  entry.GetString("message"),
		})
	}

	return guestbookEntries
}

func EntriesHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	fullUrl := url.GenerateCurrentPageUrl(c)
	year := cmp.Or(c.Request.URL.Query().Get("year"), fmt.Sprintf("%d", time.Now().Year()))

	// entries, err := wgaModels.FindEntriesForYear(app.Dao(), year)
	entries, err := app.FindRecordsByFilter("Guestbook", "year", year, 0, 0)

	if err != nil {
		app.Logger().Error("Failed to get guestbook entries", "error", err)
		return utils.ServerFaultError(c)
	}

	content := pages.GuestbookView{
		SelectedYear: year,
		YearOptions:  yearOptions(),
		Entries:      convertRawEntriesToGuestbookEntries(entries),
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Guestbook")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "This is the guestbook of the Web Gallery of Art. Please feel free to leave a message.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

	c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buff bytes.Buffer

	err = pages.GuestbookPage(content).Render(ctx, &buff)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func StoreEntryViewHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	var buff bytes.Buffer
	err := pages.GuestbookEntryForm().Render(context.Background(), &buff)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func StoreEntryHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	inputStruct := GuestBookMessage{}

	if err := c.BindBody(&inputStruct); err != nil {
		utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
		return utils.BadRequestError(c)
	}

	if inputStruct.HoneyPotEmail != "" || inputStruct.HoneyPotName != "" {
		// this is probably a bot
		app.Logger().Error("Guestbook HoneyPot triggered", "ip", c.RealIP())
		utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
		return c.NoContent(204)
	}

	collection, err := app.FindCollectionByNameOrId("Guestbook")
	if err != nil {
		app.Logger().Error("Database table not found", "error", err.Error())
		utils.SendToastMessage("Something went wrong!", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	record := core.NewRecord(collection)

	record.Set("name", inputStruct.Name)
	record.Set("email", inputStruct.Email)
	record.Set("location", inputStruct.Location)
	record.Set("message", inputStruct.Message)

	if err := app.Save(record); err != nil {

		var buff bytes.Buffer

		e := pages.GuestbookEntryForm().Render(context.Background(), &buff)

		if e != nil {
			app.Logger().Error("Failed to render the guestbook entry form after form submission error", "error", e.Error())
			return utils.ServerFaultError(c)
		}

		app.Logger().Error("Failed to store the entry", "error", err.Error(), "data", inputStruct)

		utils.SendToastMessage("Failed to store the entry", "error", false, c, "")

		return c.HTML(http.StatusOK, buff.String())
	}

	utils.SendToastMessage("Message added successfully", "success", true, c, "guestbook-updated")

	return nil
}

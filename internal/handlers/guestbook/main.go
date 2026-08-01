package guestbook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/blackfyre/wga/internal/validation"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

type GuestBookMessage struct {
	Name          string `json:"name" form:"sender_name" query:"name" validate:"required"`
	Email         string `json:"email" form:"sender_email" query:"email" validate:"required"`
	Location      string `json:"location" form:"location" query:"location" validate:"required"`
	Message       string `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName  string `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
}

const guestbookYearsCacheTTL = 10 * time.Minute
const guestbookPageSize = 10

type filters struct {
	Query string
	Year  string
	Sort  string
	Show  int
}

func buildFilters(c *core.RequestEvent, currentYear string) filters {
	show, err := strconv.Atoi(c.Request.URL.Query().Get("show"))
	if err != nil || show < guestbookPageSize {
		show = guestbookPageSize
	}

	year := c.Request.URL.Query().Get("year")
	if year == "" {
		year = currentYear
	}

	sort := c.Request.URL.Query().Get("sort")
	if sort != "oldest" {
		sort = "newest"
	}

	return filters{
		Query: strings.TrimSpace(c.Request.URL.Query().Get("q")),
		Year:  year,
		Sort:  sort,
		Show:  show,
	}
}

func (f filters) buildFilter() (string, dbx.Params) {
	parts := []string{}
	params := dbx.Params{}

	if f.Year != "all" {
		parts = append(parts, "created ~ {:year}")
		params["year"] = f.Year
	}
	if f.Query != "" {
		parts = append(parts, "(name ~ {:query} || location ~ {:query} || message ~ {:query})")
		params["query"] = f.Query
	}

	return strings.Join(parts, " && "), params
}

func (f filters) sortExpression() string {
	if f.Sort == "oldest" {
		return "+created"
	}

	return "-created"
}

func yearOptions(app core.App, currentYear string) ([]string, error) {
	years, err := utils.GetOrLoadCachedValue(app, constants.CacheGuestbookYears, guestbookYearsCacheTTL, func() ([]string, error) {
		entries, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "", "-created", 0, 0)
		if err != nil {
			return nil, err
		}

		years := []string{}
		seen := map[string]struct{}{}
		for _, entry := range entries {
			year := fmt.Sprintf("%d", entry.GetDateTime("created").Time().Year())
			if _, ok := seen[year]; ok {
				continue
			}

			seen[year] = struct{}{}
			years = append(years, year)
		}

		return years, nil
	})
	if err != nil {
		return nil, err
	}

	return withCurrentYear(years, currentYear), nil
}

func withCurrentYear(years []string, currentYear string) []string {
	for _, year := range years {
		if year == currentYear {
			return years
		}
	}

	return append([]string{currentYear}, years...)
}

func convertRawEntriesToGuestbookEntries(entries []*core.Record) []dto.GuestbookEntry {
	var guestbookEntries []dto.GuestbookEntry

	for _, entry := range entries {
		guestbookEntries = append(guestbookEntries, dto.GuestbookEntry{
			Name:     entry.GetString("name"),
			Email:    entry.GetString("email"),
			Location: entry.GetString("location"),
			Message:  entry.GetString("message"),
			Created:  entry.GetDateTime("created").Time().Format("2006-01-02"),
		})
	}

	return guestbookEntries
}

func EntriesHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	app.Logger().Debug("Guestbook entries request received", "url", c.Request.URL)

	fullUrl := url.GenerateCurrentPageUrl(c)
	currentYear := fmt.Sprintf("%d", time.Now().Year())
	filters := buildFilters(c, currentYear)
	years, err := yearOptions(app, currentYear)
	if err != nil {
		app.Logger().Error("Failed to get guestbook years", "error", err)
		return utils.ServerFaultError(c)
	}

	app.Logger().Debug("Guestbook entries request", "year", filters.Year, "query", filters.Query, "sort", filters.Sort, "show", filters.Show, "fullUrl", fullUrl)

	filter, params := filters.buildFilter()
	scopeTotal, err := utils.CountRecordsByFilter(app, constants.CollectionGuestbook, filter, params)
	if err != nil {
		app.Logger().Error("Failed to count guestbook entries", "error", err)
		return utils.ServerFaultError(c)
	}
	total, err := utils.CountRecordsByFilter(app, constants.CollectionGuestbook, "", dbx.Params{})
	if err != nil {
		app.Logger().Error("Failed to count all guestbook entries", "error", err)
		return utils.ServerFaultError(c)
	}

	entries, err := app.FindRecordsByFilter(constants.CollectionGuestbook, filter, filters.sortExpression(), filters.Show, 0, params)

	if err != nil {
		app.Logger().Error("Failed to get guestbook entries", "error", err)
		return utils.ServerFaultError(c)
	}

	content := pages.GuestbookView{
		Total:        total,
		Query:        filters.Query,
		Sort:         filters.Sort,
		Shown:        len(entries),
		ScopeTotal:   scopeTotal,
		SelectedYear: filters.Year,
		CurrentYear:  currentYear,
		YearOptions:  years,
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

func StoreEntryHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	inputStruct := GuestBookMessage{}

	if err := c.BindBody(&inputStruct); err != nil {
		utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
		return utils.BadRequestError(c)
	}

	if err := validation.ValidateHoneypot(inputStruct.HoneyPotName, inputStruct.HoneyPotEmail); err != nil {
		if errors.Is(err, errs.ErrHoneypotTriggered) {
			app.Logger().Error("Guestbook HoneyPot triggered", "ip", c.RealIP())
			utils.SendToastMessage("Failed to create message, please try again later.", "error", true, c, "")
			return c.NoContent(204)
		}

		return utils.ServerFaultError(c)
	}

	collection, err := app.FindCollectionByNameOrId(constants.CollectionGuestbook)
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

	utils.SendToastMessage("Message added successfully", "success", false, c, "guestbook-updated")

	c.Response.Header().Set("HX-Push-Url", "/guestbook")

	return c.NoContent(http.StatusNoContent)
}

func RegisterHandlers(app *pocketbase.PocketBase) {

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("/guestbook")

		ag.GET("", func(c *core.RequestEvent) error {
			return EntriesHandler(app, c)
		})

		ag.POST("/add", func(c *core.RequestEvent) error {
			return StoreEntryHandler(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		return se.Next()
	})
}

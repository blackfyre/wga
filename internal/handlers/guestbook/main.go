package guestbook

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	guestbookYearsCacheTTL = 10 * time.Minute
	guestbookPageSize      = 10
)

type filters struct {
	Query string
	Year  string
	Show  int
}

func buildFilters(c *core.RequestEvent, currentYear string) filters {
	show, err := strconv.Atoi(c.Request.URL.Query().Get("show"))
	if err != nil || show < guestbookPageSize {
		show = guestbookPageSize
	}
	if show > 100 {
		show = 100
	}

	year := c.Request.URL.Query().Get("year")
	if year == "" {
		year = currentYear
	}
	if year != "all" {
		parsedYear, err := strconv.Atoi(year)
		if err != nil || parsedYear < 1998 || parsedYear > time.Now().Year() {
			year = currentYear
		}
	}

	return filters{
		Query: strings.TrimSpace(c.Request.URL.Query().Get("q")),
		Year:  year,
		Show:  show,
	}
}

func (f filters) buildFilter() (string, dbx.Params) {
	return publicFilter(f)
}

func yearOptions(app core.App, currentYear string) ([]string, error) {
	years, err := utils.GetOrLoadCachedValue(app, constants.CacheGuestbookYears, guestbookYearsCacheTTL, func() ([]string, error) {
		return (repository{app: app}).approvedYears()
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

func guestbookView(app core.App, c *core.RequestEvent, form pages.GuestbookFormView) (pages.GuestbookView, error) {
	currentYear := fmt.Sprintf("%d", time.Now().Year())
	selected := buildFilters(c, currentYear)
	repo := repository{app: app}

	years, err := yearOptions(app, currentYear)
	if err != nil {
		return pages.GuestbookView{}, err
	}
	total, scopeTotal, err := repo.publicCounts(selected)
	if err != nil {
		return pages.GuestbookView{}, err
	}
	entries, err := repo.publicEntries(selected, selected.Show)
	if err != nil {
		return pages.GuestbookView{}, err
	}

	return pages.GuestbookView{
		Total:        total,
		Query:        selected.Query,
		Shown:        len(entries),
		ScopeTotal:   scopeTotal,
		SelectedYear: selected.Year,
		CurrentYear:  currentYear,
		YearOptions:  years,
		Entries:      entries,
		Form:         form,
	}, nil
}

func renderGuestbook(app core.App, c *core.RequestEvent, status int, form pages.GuestbookFormView) error {
	view, err := guestbookView(app, c, form)
	if err != nil {
		return err
	}

	fullURL := url.GenerateCurrentPageUrl(c)
	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Guestbook")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Read approved notes from visitors or leave a moderated guestbook entry.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullURL)

	var buff bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.GuestbookBlock(view).Render(ctx, &buff)
	} else {
		err = pages.GuestbookPage(view).Render(ctx, &buff)
	}
	if err != nil {
		return err
	}

	c.Response.Header().Set("HX-Push-Url", fullURL)
	return c.HTML(status, buff.String())
}

func EntriesHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	logger := logging.RequestLogger(app, c)
	form := pages.GuestbookFormView{Submitted: c.Request.URL.Query().Get("submitted") == "1"}
	if err := renderGuestbook(app, c, http.StatusOK, form); err != nil {
		logger.Error("Guestbook archive failed",
			"event", "guestbook.archive.failed",
			"outcome", "render_or_query_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return nil
}

// sameOriginRequest reports whether the request carries an explicit same-origin
// Origin or Referer header, or neither header at all. The no-header case
// preserves non-browser and no-JavaScript clients that never send either value.
func sameOriginRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		return sameOriginReference(origin, r.Host, true)
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		return sameOriginReference(referer, r.Host, false)
	}

	return true
}

// sameOriginReference validates a full reference URL against the request Host
// using URL hostname/port semantics, including default-port equivalence so
// that "example.com" and "example.com:443" compare equal.
func sameOriginReference(rawValue, requestHost string, requireHTTPScheme bool) bool {
	reference, err := neturl.Parse(rawValue)
	if err != nil {
		return false
	}
	if requireHTTPScheme && reference.Scheme != "http" && reference.Scheme != "https" {
		return false
	}

	defaultPort := ""
	switch reference.Scheme {
	case "http":
		defaultPort = "80"
	case "https":
		defaultPort = "443"
	}

	referenceHostname, referencePort, ok := authorityParts(reference, defaultPort)
	if !ok {
		return false
	}

	requestURL, err := neturl.Parse("//" + requestHost)
	if err != nil {
		return false
	}
	requestHostname, requestPort, ok := authorityParts(requestURL, defaultPort)
	if !ok {
		return false
	}

	return referenceHostname == requestHostname && referencePort == requestPort
}

// authorityParts reduces a parsed authority to a lower-cased hostname and an
// effective port, substituting the scheme's default port when none is present.
func authorityParts(u *neturl.URL, defaultPort string) (string, string, bool) {
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return "", "", false
	}

	port := u.Port()
	if port == "" {
		port = defaultPort
	}

	return hostname, port, true
}

func storeEntryHandler(app core.App, c *core.RequestEvent, limiter *submissionRateLimiter, resolver requesttrust.Resolver) error {
	logger := logging.RequestLogger(app, c)

	if !sameOriginRequest(c.Request) {
		logger.Warn("Guestbook submission rejected",
			"event", "guestbook.submission.rejected",
			"outcome", "cross_origin",
		)
		return renderSubmissionFailure(app, c, http.StatusForbidden, submissionInput{}, submissionErrors{Form: "Check the form and try again."})
	}

	clientID, ok := "", false
	if resolver != nil {
		clientID, ok = resolver(c.Request)
	}
	if !ok || clientID == "" {
		logger.Warn("Guestbook submission rejected",
			"event", "guestbook.submission.rejected",
			"outcome", "missing_trusted_identity",
		)
		return renderSubmissionFailure(app, c, http.StatusBadRequest, submissionInput{}, submissionErrors{Form: "Check the form and try again."})
	}

	input := submissionInput{}
	if err := c.BindBody(&input); err != nil {
		logger.Warn("Guestbook submission rejected",
			"event", "guestbook.submission.rejected",
			"outcome", "invalid_payload",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return renderSubmissionFailure(app, c, http.StatusBadRequest, input, submissionErrors{Form: "Check the form and try again."})
	}

	prepared, validationErrors, err := prepareSubmission(input)
	if err != nil {
		if errors.Is(err, errs.ErrHoneypotTriggered) {
			logger.Warn("Guestbook submission rejected",
				"event", "guestbook.submission.rejected",
				"outcome", "honeypot",
			)
			return c.NoContent(http.StatusNoContent)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	if validationErrors.any() {
		logger.Info("Guestbook submission rejected",
			"event", "guestbook.submission.rejected",
			"outcome", "validation",
		)
		return renderSubmissionFailure(app, c, http.StatusUnprocessableEntity, prepared, validationErrors)
	}

	if !limiter.allow(clientID) {
		logger.Warn("Guestbook submission rejected",
			"event", "guestbook.submission.rejected",
			"outcome", "rate_limited",
		)
		return renderSubmissionFailure(app, c, http.StatusTooManyRequests, prepared, submissionErrors{Form: "Too many notes were submitted from this connection. Please try again later."})
	}

	repo := repository{app: app}
	if err := repo.createUnreviewed(prepared, time.Now()); err != nil {
		logger.Error("Guestbook submission persistence failed",
			"event", "guestbook.submission.failed",
			"outcome", "persistence_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return renderSubmissionFailure(app, c, http.StatusInternalServerError, prepared, submissionErrors{Form: "The note could not be saved. Please try again later."})
	}

	logger.Info("Guestbook submission queued for review",
		"event", "guestbook.submission.queued",
		"outcome", guestbookStateUnreviewed,
	)

	if c.Request.Header.Get("HX-Request") != "true" {
		return c.Redirect(http.StatusSeeOther, "/guestbook?submitted=1")
	}

	return renderGuestbook(app, c, http.StatusOK, pages.GuestbookFormView{Submitted: true})
}

func renderSubmissionFailure(app core.App, c *core.RequestEvent, status int, input submissionInput, validationErrors submissionErrors) error {
	return renderGuestbook(app, c, status, pages.GuestbookFormView{
		Name:          input.Name,
		Location:      input.Location,
		Message:       input.Message,
		NameError:     validationErrors.Name,
		LocationError: validationErrors.Location,
		MessageError:  validationErrors.Message,
		FormError:     validationErrors.Form,
	})
}

func RegisterHandlers(app *pocketbase.PocketBase, resolver requesttrust.Resolver) {
	limiter := newSubmissionRateLimiter(time.Now)

	RegisterRedactionHook(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		group := se.Router.Group("/guestbook")
		group.GET("", func(c *core.RequestEvent) error {
			return EntriesHandler(app, c)
		})
		group.POST("/add", func(c *core.RequestEvent) error {
			return storeEntryHandler(app, c, limiter, resolver)
		})

		return se.Next()
	})
}

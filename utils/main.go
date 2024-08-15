package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/blackfyre/wga/assets/templ/error_pages"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	day  = 24 * time.Hour
	year = 365 * day
)

var printer = message.NewPrinter(language.English)

var TemplateFuncs = template.FuncMap{
	// Time functions
	"now":            time.Now,
	"timeSince":      time.Since,
	"timeUntil":      time.Until,
	"formatTime":     formatTime,
	"approxDuration": approxDuration,

	// String functions
	"uppercase":    strings.ToUpper,
	"lowercase":    strings.ToLower,
	"pluralize":    pluralize,
	"slugify":      Slugify,
	"safeHTML":     safeHTML,
	"strippedHTML": StrippedHTML,
	"removeExt":    RemoveExtension,

	// Slice functions
	"join": strings.Join,

	// Number functions
	"incr":        incr,
	"incrBy":      incrBy,
	"decr":        decr,
	"decrBy":      decrBy,
	"formatInt":   formatInt,
	"formatFloat": formatFloat,

	// Boolean functions
	"yesno": yesno,

	// URL functions
	"urlSetParam":        urlSetParam,
	"urlDelParam":        urlDelParam,
	"getFileNameFromUrl": GetFileNameFromUrl,

	// JSON functions
	"marshalJSON": marshalJSON,
}

func marshalJSON(v any) (template.JS, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return template.JS(b), nil
}

func StrippedHTML(s string) string {
	return strip.StripTags(s)
}

func formatTime(format string, t time.Time) string {
	return t.Format(format)
}

func approxDuration(d time.Duration) string {
	if d < time.Second {
		return "less than 1 second"
	}

	ds := int(math.Round(d.Seconds()))
	if ds == 1 {
		return "1 second"
	} else if ds < 60 {
		return fmt.Sprintf("%d seconds", ds)
	}

	dm := int(math.Round(d.Minutes()))
	if dm == 1 {
		return "1 minute"
	} else if dm < 60 {
		return fmt.Sprintf("%d minutes", dm)
	}

	dh := int(math.Round(d.Hours()))
	if dh == 1 {
		return "1 hour"
	} else if dh < 24 {
		return fmt.Sprintf("%d hours", dh)
	}

	dd := int(math.Round(float64(d / day)))
	if dd == 1 {
		return "1 day"
	} else if dd < 365 {
		return fmt.Sprintf("%d days", dd)
	}

	dy := int(math.Round(float64(d / year)))
	if dy == 1 {
		return "1 year"
	}

	return fmt.Sprintf("%d years", dy)
}

func pluralize(count any, singular string, plural string) (string, error) {
	n, err := toInt64(count)
	if err != nil {
		return "", err
	}

	if n == 1 {
		return singular, nil
	}

	return plural, nil
}

func Slugify(s string) string {
	var buf bytes.Buffer

	for _, r := range s {
		switch {
		case r > unicode.MaxASCII:
			continue
		case unicode.IsLetter(r):
			buf.WriteRune(unicode.ToLower(r))
		case unicode.IsDigit(r), r == '_', r == '-':
			buf.WriteRune(r)
		case unicode.IsSpace(r):
			buf.WriteRune('-')
		}
	}

	return buf.String()
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func incr(i any) (int64, error) {
	n, err := toInt64(i)
	if err != nil {
		return 0, err
	}

	n++
	return n, nil
}

func incrBy(i any, incr int) (int64, error) {
	n, err := toInt64(i)
	if err != nil {
		return 0, err
	}

	n += int64(incr)
	return n, nil
}

func decr(i any) (int64, error) {
	n, err := toInt64(i)
	if err != nil {
		return 0, err
	}

	n--
	return n, nil
}

func decrBy(i any, decr int) (int64, error) {
	n, err := toInt64(i)
	if err != nil {
		return 0, err
	}

	n -= int64(decr)
	return n, nil
}

func formatInt(i any) (string, error) {
	n, err := toInt64(i)
	if err != nil {
		return "", err
	}

	return printer.Sprintf("%d", n), nil
}

func formatFloat(f float64, dp int) string {
	format := "%." + strconv.Itoa(dp) + "f"
	return printer.Sprintf(format, f)
}

func yesno(b bool) string {
	if b {
		return "Yes"
	}

	return "No"
}

func urlSetParam(u *url.URL, key string, value any) *url.URL {
	nu := *u
	values := nu.Query()

	values.Set(key, fmt.Sprintf("%v", value))

	nu.RawQuery = values.Encode()
	return &nu
}

func urlDelParam(u *url.URL, key string) *url.URL {
	nu := *u
	values := nu.Query()

	values.Del(key)

	nu.RawQuery = values.Encode()
	return &nu
}

func toInt64(i any) (int64, error) {
	switch v := i.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	// Note: uint64 not supported due to risk of truncation.
	case string:
		return strconv.ParseInt(v, 10, 64)
	}

	return 0, fmt.Errorf("unable to convert type %T to int", i)
}

func RemoveExtension(s string) string {
	if idx := strings.LastIndex(s, "."); idx != -1 {
		return s[:idx]
	}
	return s
}

func GetFileNameFromUrl(url string, extension bool) string {
	// split the url by /
	splitUrl := strings.Split(url, "/")
	// get the last element of the split url
	lastElement := splitUrl[len(splitUrl)-1]

	// if extension is false, remove the extension
	if !extension {
		lastElement = RemoveExtension(lastElement)
	}

	return lastElement
}

// IsHtmxRequest checks if the request is an htmx request by checking the value of the "HX-Request" header.
func IsHtmxRequest(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}

// NotFoundError is a handler that returns a 404 error page.
func NotFoundError(c echo.Context) error {
	return error_pages.NotFoundPage().Render(context.Background(), c.Response().Writer)
}

func ServerFaultError(c echo.Context) error {
	return error_pages.ServerFaultPage().Render(context.Background(), c.Response().Writer)
}

func BadRequestError(c echo.Context) error {
	return error_pages.BadRequestPage().Render(context.Background(), c.Response().Writer)
}

func NormalizedBirthDeathActivity(record *models.Record) string {
	Start := record.GetInt("year_of_birth")
	End := record.GetInt("year_of_death")

	return fmt.Sprintf("%d-%d", Start, End)
}

func GenerateArtistSlug(artist *models.Record) string {
	if artist == nil {
		return ""
	}
	return artist.GetString("slug") + "-" + artist.GetString("id")
}

func GenerateCurrentPageUrl(c echo.Context) string {
	if c == nil || c.Request() == nil {
		return ""
	}
	return c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()
}

// renderSchoolNames takes an instance of PocketBase and a slice of school IDs,
// and returns a string containing the names of the schools corresponding to the given IDs.
// If a school is not found, it logs an error and continues to the next ID.
func RenderSchoolNames(app *pocketbase.PocketBase, schoolIds []string) string {
	var schoolCollector []string

	for _, s := range schoolIds {
		r, err := app.Dao().FindRecordById("schools", s)

		if err != nil {
			app.Logger().Error("school not found", "error", err.Error())
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

	}

	return strings.Join(schoolCollector, ", ")
}

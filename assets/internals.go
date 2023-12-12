package assets

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"log"
	"os"
	"strings"

	"blackfyre.ninja/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
)

//go:embed "reference/*" "views/*"
var InternalFiles embed.FS

type Renderable struct {
	IsHtmx bool
	Page   string
	Block  string
	Data   map[string]any
}

// NewRenderData creates and returns a map containing render data for the given app.
// The render data includes the environment variable "WGA_ENV" and the contents of the "analytics.txt" file.
// If the "renderable:analytics" cache is not available, the file is read and stored in the cache.
// The "Analytics" key in the map contains the contents of the "analytics.txt" file.
func NewRenderData(app *pocketbase.PocketBase) map[string]any {

	//read file ./analytics.txt and append it to the data map

	data := map[string]any{
		"Env": os.Getenv("WGA_ENV"),
	}

	if !app.Store().Has("renderable:analytics") {

		analytics, err := os.ReadFile("./analytics.txt")

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				app.Logger().Warn("analytics.txt file not found, using empty string as default", err)
				analytics = []byte("") // Provide an empty string if file does not exist
			} else {
				app.Logger().Error("Failed to read file", err)
				return nil
			}
		}

		app.Store().Set("renderable:analytics", string(analytics))

		data["Analytics"] = string(analytics)
	} else {
		data["Analytics"] = app.Store().Get("renderable:analytics")
	}

	return data
}

// renderPage renders the given template with the provided data and returns the resulting HTML string.
// The template is parsed from the views directory using the provided template name and the layout.html file.
// If the template cannot be parsed or there is an error rendering it, an error is returned.
func RenderPage(t string, data map[string]any) (string, error) {

	patterns := []string{
		"views/layouts/layout.html",
		"views/partials/*.html",
	}

	patterns = append(patterns, "views/pages/"+t+".html")

	return renderHtml(patterns, "layout", data)
}

func RenderPageWithLayout(t string, layout string, data map[string]any) (string, error) {

	patterns := []string{
		"views/layouts/*.html",
		"views/partials/*.html",
	}

	patterns = append(patterns, "views/pages/"+t+".html")

	return renderHtml(patterns, layout, data)
}

// renderBlock renders a given block of HTML using the provided data and returns the resulting HTML string.
// The function searches for HTML templates in the "views/pages" and "views/partials" directories of the InternalFiles filesystem.
// It uses the utils.TemplateFuncs map to provide additional functions to the templates.
// If an error occurs while parsing or rendering the template, the function returns an empty string and the error.
func RenderBlock(block string, data map[string]any) (string, error) {

	patterns := []string{
		"views/pages/*.html",
		"views/pages/*/*.html",
		"views/partials/*.html",
	}

	return renderHtml(patterns, block, data)
}

// RenderEmail renders an email template with the given data.
// The function takes a string `t` representing the template name and a map `data` containing the data to be rendered.
// It returns a string representing the rendered email and an error if any occurred.
func RenderEmail(t string, data map[string]any) (string, error) {

	patterns := []string{
		"views/emails/*.html",
	}

	return renderHtml(patterns, t, data)
}

// renderHtml renders an HTML template using the provided patterns, name and data.
// It returns the rendered HTML as a string and an error if any occurred.
func renderHtml(patterns []string, name string, data map[string]any) (string, error) {

	ts, err := template.New("").Funcs(utils.TemplateFuncs).ParseFS(
		InternalFiles,
		patterns...,
	)

	if err != nil {
		log.Println("Error parsing template")
		log.Println(err)
		return "", err
	}

	html := new(bytes.Buffer)

	err = ts.ExecuteTemplate(html, name, data)

	if err != nil {
		// or redirect to a dedicated 404 HTML page
		log.Println("Error rendering template")
		log.Println(err)
		return "", apis.NewNotFoundError("", err)
	}

	return html.String(), nil
}

// Render renders a Renderable object and returns the resulting HTML string.
// If the Renderable object is marked as htmx, it renders the block using RenderBlock.
// Otherwise, it renders the page using RenderPage.
func Render(r Renderable) (string, error) {

	if r.IsHtmx {
		return RenderBlock(r.Block, r.Data)
	}

	page := ""

	if r.Page != "" {
		page = r.Page
	} else {
		page = strings.Split(r.Block, ":")[0]
	}

	if page == "" {
		return "", errors.New("Renderable " + page + " not found")
	}

	return RenderPage(page, r.Data)
}

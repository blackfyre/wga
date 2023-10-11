package handlers

import (
	"bytes"
	"html/template"
	"log"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
)

// RegisterHandlers registers all the handlers for the application.
// It takes a pointer to a PocketBase instance and initializes the cache.
// The cache is used to store frequently accessed data for faster access.
// The cache is automatically cleaned up every 30 minutes.
func RegisterHandlers(app *pocketbase.PocketBase) {
	registerGuestbook(app)
	registerArtist(app)
	registerArtists(app)
	registerStatic(app)
	registerHome(app)
}

// isHtmxRequest checks if the request is an htmx request by checking the value of the "HX-Request" header.
func isHtmxRequest(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}

// renderPage renders the given template with the provided data and returns the resulting HTML string.
// The template is parsed from the views directory using the provided template name and the layout.html file.
// If the template cannot be parsed or there is an error rendering it, an error is returned.
func renderPage(t string, data map[string]any) (string, error) {

	patterns := []string{
		"views/layout.html",
		"views/partials/*.html",
	}

	patterns = append(patterns, "views/pages/"+t+".html")

	ts, err := template.New("").Funcs(utils.TemplateFuncs).ParseFS(
		assets.InternalFiles,
		patterns...,
	)

	if err != nil {
		log.Println("Error parsing template")
		log.Println(err)
		return "", err
	}

	html := new(bytes.Buffer)

	err = ts.ExecuteTemplate(html, "layout", data)

	if err != nil {
		// or redirect to a dedicated 404 HTML page
		log.Println("Error rendering template")
		log.Println(err)
		return "", apis.NewNotFoundError("", err)
	}

	return html.String(), nil
}

func renderBlock(block string, data map[string]any) (string, error) {

	patterns := []string{
		"views/pages/*.html",
		"views/partials/*.html",
	}

	ts, err := template.New("").Funcs(utils.TemplateFuncs).ParseFS(
		assets.InternalFiles,
		patterns...,
	)

	if err != nil {
		log.Println("Error parsing template")
		log.Println(err)
		return "", err
	}

	html := new(bytes.Buffer)

	err = ts.ExecuteTemplate(html, block, data)

	if err != nil {
		// or redirect to a dedicated 404 HTML page
		log.Println("Error rendering template")
		log.Println(err)
		return "", apis.NewNotFoundError("", err)
	}

	return html.String(), nil
}

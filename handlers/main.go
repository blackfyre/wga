package handlers

import (
	"bytes"
	"html/template"
	"log"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
)

func RegisterHandlers(app *pocketbase.PocketBase) {
	registerArtist(app)
	registerStatic(app)
	registerHome(app)
}

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

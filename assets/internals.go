package assets

import (
	"bytes"
	"embed"
	"html/template"
	"log"

	"github.com/blackfyre/wga/utils"
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

func RenderPageWithLayout(t string, layout string, data map[string]any) (string, error) {

	patterns := []string{
		"views/layouts/*.html",
		"views/partials/*.html",
	}

	patterns = append(patterns, "views/pages/"+t+".html")

	return renderHtml(patterns, layout, data)
}

// RenderBlock renders a given block of HTML using the provided data and returns the resulting HTML string.
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

package glossary

import (
	"bytes"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	annotationglossary "github.com/blackfyre/wga/internal/utils/glossary"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const glossaryPath = "/glossary"

type glossaryQuery struct {
	Letter string
	Text   string
}

func buildGlossaryQuery(c *core.RequestEvent) glossaryQuery {
	query := c.Request.URL.Query()
	letter := strings.ToUpper(strings.TrimSpace(query.Get("letter")))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		letter = ""
	}

	return glossaryQuery{
		Letter: letter,
		Text:   strings.TrimSpace(query.Get("q")),
	}
}

func findGlossaryTerms(app core.App, query glossaryQuery) ([]pages.GlossaryTerm, error) {
	records, err := app.FindRecordsByFilter(constants.CollectionGlossary, "", "+expression", 0, 0)
	if err != nil {
		return nil, err
	}

	terms := make([]pages.GlossaryTerm, 0, len(records))
	for _, record := range records {
		expression := record.GetString("expression")
		definition := record.GetString("definition")
		terms = append(terms, pages.GlossaryTerm{
			Expression: expression,
			Definition: annotationglossary.SanitizeDefinition(definition),
		})
	}

	sort.Slice(terms, func(i, j int) bool {
		return strings.ToLower(terms[i].Expression) < strings.ToLower(terms[j].Expression)
	})

	return filterGlossaryTerms(terms, query), nil
}

func filterGlossaryTerms(terms []pages.GlossaryTerm, query glossaryQuery) []pages.GlossaryTerm {
	filtered := make([]pages.GlossaryTerm, 0, len(terms))
	for _, term := range terms {
		if matchesGlossaryQuery(term.Expression, term.Definition, query) {
			filtered = append(filtered, term)
		}
	}

	return filtered
}

func glossaryLetters(terms []pages.GlossaryTerm) []string {
	letters := map[string]struct{}{}
	for _, term := range terms {
		for _, character := range strings.TrimSpace(term.Expression) {
			initial := unicode.ToUpper(character)
			if initial >= 'A' && initial <= 'Z' {
				letters[string(initial)] = struct{}{}
			}
			break
		}
	}

	result := make([]string, 0, len(letters))
	for letter := range letters {
		result = append(result, letter)
	}
	sort.Strings(result)

	return result
}

func matchesGlossaryQuery(expression string, definition string, query glossaryQuery) bool {
	if query.Letter != "" && !startsWithLetter(expression, query.Letter) {
		return false
	}

	if query.Text == "" {
		return true
	}

	needle := strings.ToLower(query.Text)
	return strings.Contains(strings.ToLower(expression), needle) || strings.Contains(strings.ToLower(definition), needle)
}

func startsWithLetter(expression string, letter string) bool {
	for _, r := range strings.TrimSpace(expression) {
		return unicode.ToUpper(r) == rune(letter[0])
	}

	return false
}

func renderGlossary(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	query := buildGlossaryQuery(c)
	allTerms, err := findGlossaryTerms(app, glossaryQuery{})
	if err != nil {
		app.Logger().Error("Find glossary terms", "error", err)
		return utils.ServerFaultError(c)
	}

	view := pages.GlossaryView{
		Query:          query.Text,
		SelectedLetter: query.Letter,
		Letters:        glossaryLetters(allTerms),
		Terms:          filterGlossaryTerms(allTerms, query),
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Glossary")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Definitions of terms used in the Web Gallery of Art collection.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl(c.Request.URL.String()))
	c.Response.Header().Set("HX-Push-Url", utils.GenerateCurrentRelativePageUrl(c))

	var buffer bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.GlossaryBlock(view).Render(ctx, &buffer)
	} else {
		err = pages.GlossaryPage(view).Render(ctx, &buffer)
	}
	if err != nil {
		app.Logger().Error("Render glossary page", "error", err)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buffer.String())
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET(glossaryPath, func(c *core.RequestEvent) error {
			return renderGlossary(app, c)
		})

		return se.Next()
	})
}

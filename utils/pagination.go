package utils

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"math"
	"net/url"
	"strconv"
)

type Pagination struct {
	perPage     int
	totalAmount int
	currentPage int
	totalPage   int
	baseUrl     string
	htmxTarget  string
	htmxBaseUrl string

	// render parts
	firstPart  []string
	middlePart []string
	lastPart   []string
}

// constructor
func NewPagination(totalAmount, perPage, currentPage int, baseUrl string, htmxTarget string, htmxUrl string) *Pagination {
	if currentPage == 0 {
		currentPage = 1
	}

	n := int(math.Ceil(float64(totalAmount) / float64(perPage)))
	if currentPage > n {
		currentPage = n
	}

	if htmxUrl == "" {
		htmxUrl = baseUrl
	}

	return &Pagination{
		perPage:     perPage,
		totalAmount: totalAmount,
		currentPage: currentPage,
		totalPage:   int(math.Ceil(float64(totalAmount) / float64(perPage))),
		baseUrl:     baseUrl,
		htmxTarget:  htmxTarget,
		htmxBaseUrl: htmxUrl,
	}
}

// TotalPages returns the total number of pages in the pagination.
func (p *Pagination) TotalPages() int {
	return p.totalPage
}

// HasPages returns true if there are more than one pages in the pagination.
func (p *Pagination) HasPages() bool {
	return p.TotalPages() > 1
}

const tmpl string = `
{{ if .HasPages }}
<ul class="pagination-list">
  {{ .GetPreviousButton "Previous" }}
  {{ range .FirstPart }}
    {{ . }}
  {{ end }}
  {{ if len .MiddlePart }}
    {{ .GetDots }}
	{{ range .MiddlePart }}
	  {{ . }}
	{{ end }}
  {{ end }}
  {{ if len .LastPart }}
	{{ .GetDots }}
	{{ range .LastPart }}
	  {{ . }}
	{{ end }}
  {{ end }}
  {{ .GetNextButton "Next Page" }}
</ul>
{{end}}
`

const onEachSide int = 3

func (p *Pagination) FirstPart() []string {
	return p.firstPart
}
func (p *Pagination) MiddlePart() []string {
	return p.middlePart
}
func (p *Pagination) LastPart() []string {
	return p.lastPart
}
func (p *Pagination) generate() {
	if !p.HasPages() {
		return
	}
	if p.TotalPages() < (onEachSide*2 + 6) {
		p.firstPart = p.getUrlRange(1, p.TotalPages())
	} else {
		window := onEachSide * 2
		lastPage := p.TotalPages()
		if p.currentPage < window {
			p.firstPart = p.getUrlRange(1, window+2)
			p.lastPart = p.getUrlRange(lastPage-1, lastPage)
		} else if p.currentPage > (lastPage - window) {
			p.firstPart = p.getUrlRange(1, 2)
			p.lastPart = p.getUrlRange(lastPage-(window+2), lastPage)
		} else {
			p.firstPart = p.getUrlRange(1, 2)
			p.middlePart = p.getUrlRange(p.currentPage-onEachSide, p.currentPage+onEachSide)
			p.lastPart = p.getUrlRange(lastPage-1, lastPage)
		}
	}
}

func (p *Pagination) getUrlRange(start, end int) []string {
	var ret []string
	for i := start; i <= end; i++ {
		ret = append(ret, p.getUrl(i, strconv.Itoa(i)))
	}
	return ret
}

func (p *Pagination) getUrl(page int, text string) string {
	strPage := strconv.Itoa(page)
	if p.currentPage == page {
		return p.GetActivePageWrapper(strPage)
	} else {
		baseUrl, _ := url.Parse(p.baseUrl)
		params := baseUrl.Query()
		delete(params, "page")
		strParam := ""
		for k, v := range params {
			strParam = strParam + "&" + k + "=" + v[0] // TODO
		}

		href := baseUrl.Path + "?page=" + strPage + strParam

		htmxBase, _ := url.Parse(p.htmxBaseUrl)

		fmt.Printf("htmxBase: %s\n", htmxBase)

		htmxParams := htmxBase.Query()
		delete(htmxParams, "page")
		htmxStrParam := ""
		for k, v := range htmxParams {
			htmxStrParam = htmxStrParam + "&" + k + "=" + v[0] // TODO
		}

		htmxUrl := htmxBase.Path + "?page=" + strPage + htmxStrParam

		return p.GetAvailablePageWrapper(href, text, htmxUrl)
	}
}

func (p *Pagination) GetActivePageWrapper(text string) string {
	return "<li><a class=\"pagination-link is-current\" aria-label=\"Page " + text + "\" aria-current=\"page\">" + text + "</a></li>"
}
func (p *Pagination) GetDisabledPageWrapper(text string) string {
	return "<li><a class=\"pagination-link is-disabled\">" + text + "</a></li>"
}
func (p *Pagination) GetAvailablePageWrapper(href, page, htmxUrl string) string {
	return "<li><a class='pagination-link' aria-label='Goto page " + page + "' hx-get=\"" + htmxUrl + "\" href=\"" + href + "\" hx-target=\"#" + p.htmxTarget + "\">" + page + "</a></li>"
}
func (p *Pagination) GetDots() string {
	return "<li><span class=\"pagination-ellipsis\">&hellip;</span></li>"
}
func (p *Pagination) GetPreviousButton(text string) string {
	if p.currentPage <= 1 {
		return p.GetDisabledPageWrapper(text)
	}

	return p.getUrl(p.currentPage-1, text)
}
func (p *Pagination) GetNextButton(text string) string {
	if p.currentPage == p.TotalPages() {
		return p.GetDisabledPageWrapper(text)
	}
	return p.getUrl(p.currentPage+1, text)
}

// Render generates the HTML for the pagination component and returns it as a template.HTML value.
func (p *Pagination) Render() template.HTML {
	p.generate()

	var out bytes.Buffer
	t := template.Must(template.New("pagination").Parse(tmpl))
	err := t.Execute(&out, p)
	if err != nil {
		return template.HTML(fmt.Sprintf("Error executing pagination template: %s", err))
	}
	return template.HTML(html.UnescapeString(out.String()))
}

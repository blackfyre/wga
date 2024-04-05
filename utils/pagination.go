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
<nav class="flex items-center justify-between border-t px-4 sm:px-0 mt-2">
		<div class="-mt-px flex w-0 flex-1">
			{{ .GetPreviousButton "Previous" }}
		</div>
		<div class="hidden md:-mt-px md:flex">
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
		</div>
		<div class="-mt-px flex w-0 flex-1 justify-end">
			{{ .GetNextButton "Next Page" }}
		</div>
</nav>
{{end}}
`

const onEachSide int = 3

// FirstPart returns the first part of the pagination.
func (p *Pagination) FirstPart() []string {
	return p.firstPart
}

// MiddlePart returns the middle part of the pagination.
func (p *Pagination) MiddlePart() []string {
	return p.middlePart
}

// LastPart returns the last part of the pagination.
func (p *Pagination) LastPart() []string {
	return p.lastPart
}

// generate generates the pagination links based on the current state of the Pagination object.
// It determines the range of pages to display based on the total number of pages and the current page.
// The generated links are stored in the Pagination object's firstPart, middlePart, and lastPart fields.
// If there are no pages to display, the method returns early.
// If the total number of pages is less than (onEachSide*2 + 6), the method generates links for all pages.
// Otherwise, the method generates links based on the current page's position relative to the window size.
// If the current page is within the first window, links are generated for the first window + 2 pages and the last 2 pages.
// If the current page is within the last window, links are generated for the first 2 pages and the last window + 2 pages.
// Otherwise, links are generated for the first 2 pages, the pages within the current page's window, and the last 2 pages.
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

// getUrlRange returns a slice of URLs within the specified range.
// The start and end parameters define the range of URLs to be generated.
// Each URL is generated using the getUrl function with the corresponding index.
func (p *Pagination) getUrlRange(start, end int) []string {
	var ret []string
	for i := start; i <= end; i++ {
		ret = append(ret, p.getUrl(i, strconv.Itoa(i)))
	}
	return ret
}

// getUrl returns the URL for a specific page with the given text.
// It takes the page number and text as parameters and constructs the URL based on the current state of the Pagination object.
// If the current page matches the given page number, it returns the URL wrapped in the active page wrapper.
// Otherwise, it constructs the URL by appending the page number and any query parameters from the base URL and htmx base URL.
// The constructed URL is then wrapped in the available page wrapper along with the given text.
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

// GetActivePageWrapper returns an HTML string representing the active page link for pagination.
// The function takes a `text` parameter, which is the text to be displayed inside the link.
// It generates an HTML string with the active page link using the provided `text`.
func (p *Pagination) GetActivePageWrapper(text string) string {
	return "<a class=\"inline-flex items-center border-t-2 border-primary px-4 pt-4 text-sm font-medium text-primary\" aria-label=\"Page " + text + "\" aria-current=\"page\">" + text + "</a>"
}

// GetDisabledPageWrapper returns a disabled page wrapper HTML element for pagination.
// It takes a `text` parameter representing the text to be displayed within the wrapper.
// The method returns a string containing the disabled page wrapper HTML element.
func (p *Pagination) GetDisabledPageWrapper(text string) string {
	return "<a class=\"inline-flex items-center border-t-2 border-transparent pr-1 pt-4 text-sm font-medium text-gray-500 cursor-not-allowed\">" + text + "</a>"
}

// GetAvailablePageWrapper returns a string representing a pagination link with the given href, page number, and htmxUrl.
// If htmxTarget is specified, it adds the hx-target attribute to the link.
func (p *Pagination) GetAvailablePageWrapper(href, page, htmxUrl string) string {

	str := "<a class='inline-flex items-center border-t-2 border-transparent pr-1 pt-4 text-sm font-medium text-gray-500 hover:border-secondary hover:text-secondary' aria-label='Goto page " + page + "' hx-get='" + htmxUrl + "' href='" + href

	if p.htmxTarget != "" {
		str = str + "' hx-target='#" + p.htmxTarget
	}

	str = str + "'>" + page + "</a>"

	return str
}

// GetDots returns the HTML representation of the dots used for pagination ellipsis.
func (p *Pagination) GetDots() string {
	return "<span class=\"inline-flex items-center border-t-2 border-transparent pr-1 pt-4 text-sm font-medium text-gray-500 cursor-not-allowed \">&hellip;</span>"
}

// GetPreviousButton returns the HTML string for the previous button in the pagination.
// If the current page is the first page, it returns a disabled button.
// Otherwise, it returns a button with the URL for the previous page and the specified text.
func (p *Pagination) GetPreviousButton(text string) string {
	if p.currentPage <= 1 {
		return p.GetDisabledPageWrapper(text)
	}

	return p.getUrl(p.currentPage-1, text)
}

// GetNextButton returns the HTML code for the next button in the pagination.
// If the current page is the last page, it returns the disabled page wrapper.
// Otherwise, it returns the URL for the next page with the specified text.
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

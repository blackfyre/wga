package models

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type StaticPage struct {
	models.BaseModel
	Title   string `json:"title" db:"title"`
	Slug    string `json:"slug" db:"slug"`
	Content string `json:"content" db:"content"`
}

var _ models.Model = (*StaticPage)(nil)

// TableName returns the name of the collection associated with the StaticPage model.
func (m *StaticPage) TableName() string {
	return "static_pages"
}

// StaticPageQuery returns a new dbx.SelectQuery for querying StaticPage models.
func StaticPageQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&StaticPage{})
}

// FindStaticPageBySlug retrieves a StaticPage from the database by its slug.
// It performs a case-insensitive match on the slug parameter.
// Returns a pointer to the StaticPage and an error if any occurred.
func FindStaticPageBySlug(dao *daos.Dao, slug string) (*StaticPage, error) {
	page := &StaticPage{}

	err := StaticPageQuery(dao).
		AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
			"slug": strings.ToLower(slug),
		})).
		Limit(1).
		One(page)

	if err != nil {
		return nil, err
	}

	return page, nil
}

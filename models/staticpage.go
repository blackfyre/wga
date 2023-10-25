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

func (m *StaticPage) TableName() string {
	return "static_pages" // the name of your collection
}

func StaticPageQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&StaticPage{})
}

func FindStaticPageBySlug(dao *daos.Dao, slug string) (*StaticPage, error) {
	page := &StaticPage{}

	err := StaticPageQuery(dao).
		// case insensitive match
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

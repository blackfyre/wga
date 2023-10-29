package models

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type School struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
	Slug string `db:"slug" json:"slug"`
}

var _ models.Model = (*School)(nil)

func (m *School) TableName() string {
	return "schools" // the name of your collection
}

func SchoolQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&School{})
}

func GetSchools(dao *daos.Dao) ([]*School, error) {
	var c []*School
	err := SchoolQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}

func GetSchoolBySlug(dao *daos.Dao, slug string) (*School, error) {
	var c School
	err := SchoolQuery(dao).AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
		"slug": strings.ToLower(slug),
	})).
		Limit(1).
		One(&c)
	return &c, err
}

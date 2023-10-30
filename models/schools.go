package models

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// School represents a school model with its name and slug.
type School struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
	Slug string `db:"slug" json:"slug"`
}

var _ models.Model = (*School)(nil)

// TableName returns the name of the collection for School model.
func (m *School) TableName() string {
	return "schools" // the name of your collection
}

// SchoolQuery returns a new dbx.SelectQuery for the School model.
// It takes a dao object as a parameter and returns a pointer to the new query.
func SchoolQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&School{})
}

// GetSchools retrieves all schools from the database and returns them as a slice of School structs.
// The schools are sorted by name in ascending order.
func GetSchools(dao *daos.Dao) ([]*School, error) {
	var c []*School
	err := SchoolQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}

// GetSchoolBySlug retrieves a school by its slug from the database.
// It takes a dao object and a string slug as input parameters.
// It returns a pointer to a School object and an error object.
func GetSchoolBySlug(dao *daos.Dao, slug string) (*School, error) {
	var c School
	err := SchoolQuery(dao).AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
		"slug": strings.ToLower(slug),
	})).
		Limit(1).
		One(&c)
	return &c, err
}

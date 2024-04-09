package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type Artwork struct {
	models.BaseModel
	Title     string `db:"title" json:"title"`
	Author    string `db:"author" json:"author"`
	Form      string `db:"form" json:"form"`
	Technique string `db:"technique" json:"technique"`
	School    string `db:"school" json:"school"`
	Comment   string `db:"comment" json:"comment"`
	Published bool   `db:"published" json:"published"`
	Image     string `db:"image" json:"image"`
	Type      string `db:"type" json:"type"`
}

var _ models.Model = (*Artwork)(nil)

func (m *Artwork) TableName() string {
	return "artworks" // the name of your collection
}

// ArtworkQuery returns a new dbx.SelectQuery for the Artwork model.
func ArtworkQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Artwork{})
}

// GetArtworks retrieves all artworks from the database.
// It takes a dao object as a parameter and returns a slice of Artwork pointers and an error.
// The artworks are ordered by title in ascending order.
func GetArtworks(dao *daos.Dao) ([]*Artwork, error) {
	var c []*Artwork
	err := ArtworkQuery(dao).OrderBy("title asc").All(&c)
	return c, err
}

// GetRandomArtworks returns a slice of random Artwork objects from the database.
// It takes a dao object and the number of items to retrieve as parameters.
// It returns the slice of Artwork objects and an error, if any.
func GetRandomArtworks(dao *daos.Dao, itemCount int64) ([]*Artwork, error) {
	var c []*Artwork
	err := ArtworkQuery(dao).Where(dbx.NewExp("author != \"\"")).OrderBy("RANDOM()").Limit(itemCount).All(&c)
	return c, err
}

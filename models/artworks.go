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

func GetArtworks(dao *daos.Dao) ([]*Artwork, error) {
	var c []*Artwork
	err := ArtworkQuery(dao).OrderBy("title asc").All(&c)
	return c, err
}

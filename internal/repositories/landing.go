package repositories

import (
	"database/sql"
	"errors"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const recentEligibleArtworkLimit = 4

// landingArtistIdentity is the fail-closed predicate for a home-page artist:
// the encyclopaedic filing form must be present before an artwork byline can
// be rendered. Prior-bootstrap records carry a blank filing_name and are
// denied rather than presented with an empty byline.
const landingArtistIdentity = "TRIM(artist.filing_name) != '' AND TRIM(artist.short_name) != ''"

func landingIdentityFieldsPresent(app core.App) (bool, error) {
	artists, err := app.FindCachedCollectionByNameOrId("artists")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return artists.Fields.GetByName("filing_name") != nil && artists.Fields.GetByName("short_name") != nil, nil
}

type LandingRepository struct {
	app *pocketbase.PocketBase
}

type countRow struct {
	Count int `db:"c"`
}

// LandingArtwork is an artwork with its first, published author. The landing
// page uses the first author when generating its existing canonical links.
type LandingArtwork struct {
	Artwork *core.Record
	Artist  *core.Record
}

type landingArtworkRow struct {
	ArtworkID         string `db:"artwork_id"`
	ArtworkTitle      string `db:"artwork_title"`
	ArtworkYear       string `db:"artwork_year"`
	ArtworkImage      string `db:"artwork_image"`
	ArtworkImageWidth int    `db:"artwork_image_width"`
	ArtistID          string `db:"artist_id"`
	ArtistName        string `db:"artist_name"`
	ArtistFilingName  string `db:"artist_filing_name"`
}

func NewLandingRepository(app *pocketbase.PocketBase) *LandingRepository {
	return &LandingRepository{app: app}
}

func (r *LandingRepository) CountPublishedArtists() (int, error) {
	present, err := landingIdentityFieldsPresent(r.app)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, nil
	}
	row := countRow{}
	err = r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Artists AS artist WHERE artist.published IS true AND " + landingArtistIdentity).One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountPublishedArtworks() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Artworks WHERE published IS true").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountSchools() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Schools").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountEligibleArtworks() (int, error) {
	present, err := landingIdentityFieldsPresent(r.app)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, nil
	}
	row := countRow{}
	err = r.app.DB().NewQuery(`
		SELECT COUNT(*) AS c
		FROM Artworks AS artwork
		INNER JOIN Artists AS artist ON artist.id = json_extract(artwork.author, '$[0]')
		WHERE artwork.published IS true AND artist.published IS true AND ` + landingArtistIdentity + `
	`).One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

// FindEligibleArtworkByOffset returns one published artwork whose first author
// is present and published, ordered by its stable ID. Invalid offsets have no
// matching artwork.
func (r *LandingRepository) FindEligibleArtworkByOffset(offset int) (*LandingArtwork, error) {
	if offset < 0 {
		return nil, nil
	}
	present, err := landingIdentityFieldsPresent(r.app)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}

	rows := []landingArtworkRow{}
	err = r.app.DB().NewQuery(`
		SELECT
			artwork.id AS artwork_id,
			artwork.title AS artwork_title,
			artwork.year AS artwork_year,
			artwork.image AS artwork_image,
			artwork.image_width AS artwork_image_width,
			artist.id AS artist_id,
			artist.name AS artist_name,
			artist.filing_name AS artist_filing_name
		FROM Artworks AS artwork
		INNER JOIN Artists AS artist ON artist.id = json_extract(artwork.author, '$[0]')
		WHERE artwork.published IS true AND artist.published IS true AND ` + landingArtistIdentity + `
		ORDER BY artwork.id ASC
		LIMIT 1 OFFSET {:offset}
	`).Bind(dbx.Params{"offset": offset}).All(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	artworks, err := r.landingArtworks(rows)
	if err != nil {
		return nil, err
	}

	return &artworks[0], nil
}

// ListRecentEligibleArtworks returns at most four published artworks whose
// first authors are present and published, newest first.
func (r *LandingRepository) ListRecentEligibleArtworks() ([]LandingArtwork, error) {
	present, err := landingIdentityFieldsPresent(r.app)
	if err != nil {
		return nil, err
	}
	if !present {
		return []LandingArtwork{}, nil
	}
	rows := []landingArtworkRow{}
	err = r.app.DB().NewQuery(`
		SELECT
			artwork.id AS artwork_id,
			artwork.title AS artwork_title,
			artwork.year AS artwork_year,
			artwork.image AS artwork_image,
			artwork.image_width AS artwork_image_width,
			artist.id AS artist_id,
			artist.name AS artist_name,
			artist.filing_name AS artist_filing_name
		FROM Artworks AS artwork
		INNER JOIN Artists AS artist ON artist.id = json_extract(artwork.author, '$[0]')
		WHERE artwork.published IS true AND artist.published IS true AND ` + landingArtistIdentity + `
		ORDER BY artwork.created DESC, artwork.id ASC
		LIMIT 4
	`).All(&rows)
	if err != nil {
		return nil, err
	}

	return r.landingArtworks(rows)
}

func (r *LandingRepository) landingArtworks(rows []landingArtworkRow) ([]LandingArtwork, error) {
	artworks, err := r.app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return nil, err
	}
	artists, err := r.app.FindCollectionByNameOrId(constants.CollectionArtists)
	if err != nil {
		return nil, err
	}

	result := make([]LandingArtwork, 0, len(rows))
	for _, row := range rows {
		artwork := core.NewRecord(artworks)
		artwork.Id = row.ArtworkID
		artwork.Set("title", row.ArtworkTitle)
		artwork.Set("year", row.ArtworkYear)
		artwork.Set("image", row.ArtworkImage)
		artwork.Set("image_width", row.ArtworkImageWidth)

		artist := core.NewRecord(artists)
		artist.Id = row.ArtistID
		artist.Set("name", row.ArtistName)
		artist.Set("filing_name", row.ArtistFilingName)

		result = append(result, LandingArtwork{Artwork: artwork, Artist: artist})
	}

	return result, nil
}

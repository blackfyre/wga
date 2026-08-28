package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration shares the 1784808002 bootstrap timestamp with the synthetic
// seed migration. Its filename sorts before "seed_synthetic_data" so the
// supplied filing/short identity fields exist before seed.Import records the
// producer values against them.
//
// The fields carry the producer's encyclopaedic filing form and supplied short
// form verbatim and are populated by the seed importer, never derived from the
// display name. They are optional at the schema level so a prior-bootstrap
// database with existing artists can adopt them non-destructively without
// failing record validation.
func init() {
	m.Register(addArtistIdentityFields, removeArtistIdentityFields)
}

// addArtistIdentityFields adds the filing_name and short_name text fields to
// the artists collection, or leaves them intact when an earlier application
// already declared them.
func addArtistIdentityFields(app core.App) error {
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}

	if artists.Fields.GetByName("filing_name") == nil {
		artists.Fields.Add(textField("filing_name", false, false))
	}
	if artists.Fields.GetByName("short_name") == nil {
		artists.Fields.Add(textField("short_name", false, false))
	}

	return app.Save(artists)
}

// removeArtistIdentityFields keeps the identity fields. Both are
// forward-compatible and carry factual producer data, so rollback must not
// remove them; operators disable the affected presentation instead.
func removeArtistIdentityFields(core.App) error {
	return errors.New("artist identity fields migration cannot be rolled back safely")
}

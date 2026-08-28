package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// artworkSourceCommentMaxChars permits the producer's longest source
// commentary (7,577 characters) with forward-compatible headroom. PocketBase
// applies a 5,000-character ceiling to text fields whose Max is left unset,
// which would reject or truncate real source_comment values during a
// validation-bearing import.
const artworkSourceCommentMaxChars = 10000

func init() {
	m.Register(addArtworkFileByteSize, removeArtworkFileByteSize)
}

// addArtworkFileByteSize records the original staged image byte count on the
// artworks collection for later evidence-backed FILE-weight presentation, and
// raises the source-comment ceiling so the real producer commentary imports
// without truncation or failure. The byte count is populated by the seed
// importer from factual file data and is never inferred from name, extension,
// or dimensions.
//
// It shares the 1784808002 bootstrap timestamp and sorts before
// "seed_synthetic_data" so both the field and the raised ceiling exist before
// seed.Import records producer values against them.
func addArtworkFileByteSize(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	if artworks.Fields.GetByName("image_size_bytes") == nil {
		artworks.Fields.Add(numberField("image_size_bytes", false))
	}

	if artworks.Fields.GetByName("source_comment") == nil {
		artworks.Fields.Add(textFieldWithMax("source_comment", false, false, artworkSourceCommentMaxChars))
	} else {
		raiseTextFieldMax(artworks, "source_comment", artworkSourceCommentMaxChars)
	}

	return app.Save(artworks)
}

// raiseTextFieldMax raises a text field's Max ceiling to at least min, never
// lowering an existing higher ceiling. A zero Max (PocketBase's 5,000 default)
// is treated as below any explicit min.
func raiseTextFieldMax(collection *core.Collection, name string, min int) {
	field := collection.Fields.GetByName(name)
	text, ok := field.(*core.TextField)
	if !ok {
		return
	}
	if text.Max < min {
		text.Max = min
	}
}

// removeArtworkFileByteSize keeps both the byte-size field and the raised
// source-comment ceiling. Both are forward-compatible and carry factual data,
// so rollback must not remove them; operators disable the reproduction
// presentation instead.
func removeArtworkFileByteSize(core.App) error {
	return errors.New("artwork file byte size migration cannot be rolled back safely")
}

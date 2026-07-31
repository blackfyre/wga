package migrations

import (
	"database/sql"
	"errors"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const aboutPageSlug = "about"

const aboutPageContent = `<h2>The collection</h2>
<p>The Web Gallery of Art is a searchable archive of European painting, sculpture, decorative arts and architecture from the third century to the early twentieth century.</p>
<p>It was created by Emil Krén and Daniel Marx as an independent resource for study, teaching and curiosity. The collection remains open to everyone.</p>
<h2>How to use the archive</h2>
<h3>Browse and compare</h3>
<p>Start with an artist or artwork, then follow the links between biographies, works, schools and periods. Dual Mode keeps two records open for close comparison.</p>
<h3>Read with the image</h3>
<p>Records combine reproductions with catalogue information and commentary so that an image can remain connected to its context.</p>
<h2>Terms of use</h2>
<p>The collection is protected as a database. Individual reproductions may carry the rights of the holding institution. Please consult the privacy policy for information about the limited data used to operate the service.</p>`

func init() {
	m.Register(seedAboutPage, func(core.App) error {
		return errors.New("seeded static content cannot be rolled back safely")
	})
}

func seedAboutPage(app core.App) error {
	_, err := app.FindFirstRecordByData(constants.CollectionStaticPages, "slug", aboutPageSlug)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	collection, err := app.FindCollectionByNameOrId(constants.CollectionStaticPages)
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("title", "About")
	record.Set("slug", aboutPageSlug)
	record.Set("content", aboutPageContent)

	return app.Save(record)
}

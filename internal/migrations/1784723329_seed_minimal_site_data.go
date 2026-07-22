package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(seedMinimalSiteData, func(app core.App) error {
		// Seed records may have been edited after migration; preserve them on rollback.
		return nil
	})
}

func seedMinimalSiteData(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {
		hasRecords, err := hasApplicationRecords(txApp)
		if err != nil {
			return err
		}
		if hasRecords {
			return nil
		}

		return seedMinimalSiteDataRecords(txApp)
	})
}

func hasApplicationRecords(app core.App) (bool, error) {
	collections, err := app.FindAllCollections()
	if err != nil {
		return false, err
	}

	for _, collection := range collections {
		if collection.System {
			continue
		}

		records, err := app.FindRecordsByFilter(collection.Id, "", "", 1, 0)
		if err != nil {
			return false, err
		}
		if len(records) > 0 {
			return true, nil
		}
	}

	superusers, err := app.FindRecordsByFilter(core.CollectionNameSuperusers, "", "", 1, 0)
	if err != nil {
		return false, err
	}

	return len(superusers) > 0, nil
}

func seedMinimalSiteDataRecords(txApp core.App) error {
	schoolCollection, err := txApp.FindCollectionByNameOrId("schools")
	if err != nil {
		return err
	}
	school := core.NewRecord(schoolCollection)
	school.Set("name", "Demonstration School")
	school.Set("slug", "demonstration-school")
	if err := txApp.Save(school); err != nil {
		return err
	}

	formCollection, err := txApp.FindCollectionByNameOrId("art_forms")
	if err != nil {
		return err
	}
	form := core.NewRecord(formCollection)
	form.Set("name", "Painting")
	form.Set("slug", "painting")
	if err := txApp.Save(form); err != nil {
		return err
	}

	typeCollection, err := txApp.FindCollectionByNameOrId("art_types")
	if err != nil {
		return err
	}
	artType := core.NewRecord(typeCollection)
	artType.Set("name", "Landscape")
	artType.Set("slug", "landscape")
	if err := txApp.Save(artType); err != nil {
		return err
	}

	artistCollection, err := txApp.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}
	artist := core.NewRecord(artistCollection)
	artist.Set("name", "Mara Example")
	artist.Set("slug", "mara-example")
	artist.Set("bio", "<p>Mara Example is a fictional artist included as starter content for local testing.</p>")
	artist.Set("year_of_birth", 1850)
	artist.Set("year_of_death", 1910)
	artist.Set("place_of_birth", "Sampleton")
	artist.Set("place_of_death", "Sampleton")
	artist.Set("exact_year_of_birth", false)
	artist.Set("exact_year_of_death", false)
	artist.Set("profession", "Painter")
	artist.Set("school", []string{school.Id})
	artist.Set("published", true)
	artist.Set("known_place_of_birth", NotApplicable)
	artist.Set("known_place_of_death", NotApplicable)
	if err := txApp.Save(artist); err != nil {
		return err
	}

	artworkCollection, err := txApp.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}
	artwork := core.NewRecord(artworkCollection)
	artwork.Set("title", "Cobalt Horizon")
	artwork.Set("author", []string{artist.Id})
	artwork.Set("form", []string{form.Id})
	artwork.Set("type", []string{artType.Id})
	artwork.Set("school", []string{school.Id})
	artwork.Set("technique", "Oil on canvas")
	artwork.Set("comment", "<p>A fictional landscape included as starter content for testing search and comparison.</p>")
	artwork.Set("published", true)
	if err := txApp.Save(artwork); err != nil {
		return err
	}

	stringsCollection, err := txApp.FindCollectionByNameOrId("strings")
	if err != nil {
		return err
	}
	welcome := core.NewRecord(stringsCollection)
	welcome.Set("name", "welcome")
	welcome.Set("content", "<p>Welcome to this demonstration collection. It contains fictional starter content for exploring artists, artworks, search, and comparison.</p>")
	if err := txApp.Save(welcome); err != nil {
		return err
	}

	glossaryCollection, err := txApp.FindCollectionByNameOrId("glossary")
	if err != nil {
		return err
	}
	glossary := core.NewRecord(glossaryCollection)
	glossary.Set("expression", "landscape")
	glossary.Set("definition", "A composition whose main subject is natural scenery.")
	if err := txApp.Save(glossary); err != nil {
		return err
	}

	staticPagesCollection, err := txApp.FindCollectionByNameOrId("static_pages")
	if err != nil {
		return err
	}
	privacyPage := core.NewRecord(staticPagesCollection)
	privacyPage.Set("title", "Privacy policy")
	privacyPage.Set("slug", "privacy-policy")
	privacyPage.Set("content", "<p>This development instance contains fictional starter content only.</p>")

	return txApp.Save(privacyPage)
}

package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addItineraries, removeItineraries)
}

func addItineraries(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("itineraries"); err == nil {
		return nil
	}

	if err := saveCurrentCollection(app, currentCollection("Itineraries", "itineraries",
		textField("owner", true, false),
		selectField("status", []string{"draft", "pending", "approved", "rejected"}, true),
		textField("token", false, false),
		textFieldWithMax("title", false, true, 80),
		textFieldWithMax("intro", false, false, 400),
		textFieldWithMax("creator", false, false, 40),
		boolField("listed", false),
		dateField("published", false),
		dateField("expires_at", false),
	)); err != nil {
		return err
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		return err
	}
	itineraries.Indexes = append(itineraries.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_token` ON `itineraries` (token) WHERE token != ''",
		"CREATE UNIQUE INDEX `pbx_itinerary_draft_owner` ON `itineraries` (owner) WHERE status = 'draft'",
		"CREATE INDEX `pbx_itinerary_moderation` ON `itineraries` (status, listed, published)",
		"CREATE INDEX `pbx_itinerary_expiry` ON `itineraries` (expires_at)",
	)
	if err := app.Save(itineraries); err != nil {
		return err
	}

	if err := saveCurrentCollection(app, currentCollection("Itinerary_stops", "itinerary_stops",
		relationField("itinerary", "itineraries", 1, 1, true, true),
		relationField("artwork", "artworks", 1, 1, true, false),
		textField("title", false, false),
		numberField("position", false),
		textFieldWithMax("narration", false, false, 600),
	)); err != nil {
		return err
	}

	stops, err := app.FindCollectionByNameOrId("itinerary_stops")
	if err != nil {
		return err
	}
	stops.Indexes = append(stops.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_artwork` ON `itinerary_stops` (itinerary, artwork)",
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_order` ON `itinerary_stops` (itinerary, position)",
	)

	return app.Save(stops)
}

// removeItineraries is a fail-safe rollback: it refuses to drop the itinerary
// collections because doing so would destroy published visitor data. Rollback
// is achieved operationally by disabling the public routes while preserving the
// records for forward cleanup.
func removeItineraries(app core.App) error {
	return fmt.Errorf("itinerary collections cannot be rolled back safely; disable routes instead")
}

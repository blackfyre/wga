package migrations

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addArtworkYear, func(core.App) error {
		return fmt.Errorf("artwork year migration cannot be rolled back safely")
	})
}

func addArtworkYear(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Fields.Add(numberField("year", false))

	if err := app.Save(artworks); err != nil {
		return err
	}

	records, err := app.FindRecordsByFilter("artworks", "", "", 0, 0)
	if err != nil {
		return err
	}

	for _, record := range records {
		year, ok := artworkYear(record.GetString("comment"))
		if !ok {
			continue
		}
		record.Set("year", year)
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

func artworkYear(comment string) (int, bool) {
	dateText := strings.TrimPrefix(strings.SplitN(comment, " · ", 2)[0], "<p>")
	year, err := strconv.Atoi(dateText)
	if err != nil {
		return 0, false
	}

	return year, true
}

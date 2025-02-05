package handlers

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func normalizedBirthDeathActivity(record *core.Record) string {
	Start := record.GetInt("year_of_birth")
	End := record.GetInt("year_of_death")

	return fmt.Sprintf("%d-%d", Start, End)
}

func generateArtistSlug(artist *core.Record) string {
	if artist == nil {
		return ""
	}
	return artist.GetString("slug") + "-" + artist.GetString("id")
}

func generateCurrentPageUrl(c *core.RequestEvent) string {
	if c == nil || c.Request() == nil {
		return ""
	}
	return c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()
}

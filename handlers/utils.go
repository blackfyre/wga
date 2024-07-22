package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

func normalizedBirthDeathActivity(record *models.Record) string {
	Start := record.GetInt("year_of_birth")
	End := record.GetInt("year_of_death")

	return fmt.Sprintf("%d-%d", Start, End)
}

func setHxTrigger(c echo.Context, data map[string]any) {
	hd, err := json.Marshal(data)

	if err != nil {
		log.Fatalln(err)
	}

	c.Response().Header().Set("HX-Trigger", string(hd))
}

func generateArtistSlug(artist *models.Record) string {
	return artist.GetString("slug") + "-" + artist.GetString("id")
}

func generateCurrentPageUrl(c echo.Context) string {
	return c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()
}

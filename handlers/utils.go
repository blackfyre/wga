package handlers

import (
	"fmt"

	"github.com/pocketbase/pocketbase/models"
)

func artistUrl(slug string) string {
	return "/artists/" + slug
}

func normalizedBirthDeathActivity(record *models.Record) string {
	Start := record.GetInt("year_of_birth")
	End := record.GetInt("year_of_death")

	return fmt.Sprintf("%d-%d", Start, End)
}

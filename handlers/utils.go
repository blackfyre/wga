package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

func artistUrl(r *models.Record) string {
	return "/artists/" + r.GetString("slug") + "-" + r.GetString("id")
}

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

// Deprecated: use github.com/blackfyre/wga/utils instead
func sendToastMessage(message string, t string, closeDialog bool, c echo.Context) {
	payload := struct {
		Message     string `json:"message"`
		Type        string `json:"type"`
		CloseDialog bool   `json:"closeDialog"`
	}{
		Message:     message,
		Type:        t,
		CloseDialog: closeDialog,
	}

	m := map[string]any{
		"notification:toast": payload,
	}

	setHxTrigger(c, m)
}

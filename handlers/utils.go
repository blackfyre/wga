package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
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

func generateFileUrl(app *pocketbase.PocketBase, collection string, collectionId string, fileName string) string {

	endPoint := app.Settings().S3.Endpoint

	endPoint = strings.Replace(endPoint, "https://", "https://"+app.Settings().S3.Bucket+".", 1)

	return endPoint + "/" + collection + "/" + collectionId + "/" + fileName
}

func generateThumbUrl(app *pocketbase.PocketBase, collection string, collectionId string, fileName string, thumbSize string) string {

	endPoint := app.Settings().S3.Endpoint

	endPoint = strings.Replace(endPoint, "https://", "https://"+app.Settings().S3.Bucket+".", 1)

	return endPoint + "/" + collection + "/" + collectionId + "/thumb_" + fileName + "/" + thumbSize + "_" + fileName
}

func setHxTrigger(c echo.Context, data map[string]any) {
	hd, err := json.Marshal(data)

	if err != nil {
		log.Fatalln(err)
	}

	c.Response().Header().Set("HX-Trigger", string(hd))
}

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

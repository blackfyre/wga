package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

// artistUrl returns the URL for the artist with the given slug.
func artistUrl(slug string) string {
	return "/artists/" + slug
}

// normalizedBirthDeathActivity returns a string representing the normalized birth and death activity of a record.
// It takes a pointer to a models.Record as input and calculates the start and end years of the record.
// The start year is obtained from the "year_of_birth" field of the record, and the end year is obtained from the "year_of_death" field.
// The function then returns a string in the format "startYear-endYear".
func normalizedBirthDeathActivity(record *models.Record) string {
	Start := record.GetInt("year_of_birth")
	End := record.GetInt("year_of_death")

	return fmt.Sprintf("%d-%d", Start, End)
}

// setHxTrigger sets the "HX-Trigger" header in the response with the provided data.
// It marshals the data into JSON format and sets it as the header value.
// If there is an error during the marshaling process, it logs the error and exits the program.
func setHxTrigger(c echo.Context, data map[string]any) {
	hd, err := json.Marshal(data)

	if err != nil {
		log.Fatalln(err)
	}

	c.Response().Header().Set("HX-Trigger", string(hd))
}

// sendToastMessage sends a toast message to the client.
// It takes a message string, a type string, a closeDialog boolean, and an echo.Context as parameters.
// The message parameter represents the content of the toast message.
// The type parameter represents the type of the toast message.
// The closeDialog parameter determines whether the dialog should be closed after displaying the toast message.
// The function constructs a payload struct with the message, type, and closeDialog values.
// It then creates a map with the "notification:toast" key and the payload as the value.
// Finally, it calls the setHxTrigger function with the echo.Context and the map as arguments.
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

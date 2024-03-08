package utils

import (
	"encoding/json"
	"log"

	"github.com/labstack/echo/v5"
)

func SetHxTrigger(c echo.Context, data map[string]any) {
	hd, err := json.Marshal(data)

	if err != nil {
		log.Fatalln(err)
	}

	c.Response().Header().Set("HX-Trigger", string(hd))
}

func SendToastMessage(message string, t string, closeDialog bool, c echo.Context, trigger string) {
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

	if trigger != "" {
		m[trigger] = trigger
	}

	SetHxTrigger(c, m)
}

package music

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	wgaurl "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	PlayerPath         = "/player"
	PlayerWindowName   = "wga-period-music"
	pocketBaseIDLength = 15
)

var errInvalidPlayerRequest = errors.New("invalid period-music player request")

type playerRequest struct {
	SongID string
}

func parsePlayerRequest(values url.Values) (playerRequest, error) {
	if len(values) != 1 || len(values["song"]) != 1 {
		return playerRequest{}, errInvalidPlayerRequest
	}
	request := playerRequest{SongID: values.Get("song")}
	if !validRecordID(request.SongID) {
		return playerRequest{}, errInvalidPlayerRequest
	}
	return request, nil
}

func validRecordID(value string) bool {
	if len(value) != pocketBaseIDLength {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' {
			continue
		}
		if character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

func renderPlayer(app *pocketbase.PocketBase, event *core.RequestEvent) error {
	request, err := parsePlayerRequest(event.Request.URL.Query())
	if err != nil {
		return event.HTML(http.StatusBadRequest, "Invalid period-music player request.")
	}

	record, err := app.FindRecordById("music_song", request.SongID)
	if err != nil {
		return event.HTML(http.StatusNotFound, "Period-music recording not found.")
	}
	if !record.GetBool("published") {
		return event.HTML(http.StatusNotFound, "Period-music recording not found.")
	}
	title := strings.TrimSpace(record.GetString("title"))
	source := strings.TrimSpace(record.GetString("source"))
	composerIDs := record.GetStringSlice("composer")
	if title == "" || source == "" || len(composerIDs) > 20 {
		return event.HTML(http.StatusNotFound, "Period-music recording not found.")
	}
	composer := ""
	if len(composerIDs) > 0 {
		composerRecord, composerErr := app.FindRecordById("music_composer", composerIDs[0])
		if composerErr == nil {
			if !composerRecord.GetBool("published") {
				return event.HTML(http.StatusNotFound, "Period-music recording not found.")
			}
			composer = strings.TrimSpace(composerRecord.GetString("name"))
		}
	}

	view := pages.MusicPeriodPlayerView{
		SongID:     record.Id,
		Composer:   composer,
		Piece:      title,
		Source:     wgaurl.GenerateFileUrl("music_song", record.Id, source, ""),
		WindowName: PlayerWindowName,
	}

	var output bytes.Buffer
	if err := pages.MusicPeriodPlayerPage(view).Render(context.Background(), &output); err != nil {
		return err
	}

	event.Response.Header().Set("Cache-Control", "no-store")
	return event.HTML(http.StatusOK, output.String())
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(event *core.ServeEvent) error {
		event.Router.GET(PlayerPath, func(request *core.RequestEvent) error {
			return renderPlayer(app, request)
		})
		return event.Next()
	})
}

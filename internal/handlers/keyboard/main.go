package keyboard

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	minimumQueryRunes = 2
	suggestionLimit   = 10
	requestsPerMinute = 30
)

type requestWindow struct {
	started time.Time
	count   int
}

type requestLimiter struct {
	mu      sync.Mutex
	clients map[string]requestWindow
	now     func() time.Time
}

func newRequestLimiter() *requestLimiter {
	return &requestLimiter{clients: map[string]requestWindow{}, now: time.Now}
}

func (l *requestLimiter) allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for name, window := range l.clients {
		if now.Sub(window.started) >= time.Minute {
			delete(l.clients, name)
		}
	}
	window := l.clients[client]
	if now.Sub(window.started) >= time.Minute {
		window = requestWindow{started: now}
	}
	if window.count >= requestsPerMinute {
		return false
	}

	window.count++
	l.clients[client] = window
	return true
}

func validQuery(query string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(query)) >= minimumQueryRunes
}

func suggestionLimitFor(value string) int {
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 {
		return suggestionLimit
	}
	if limit > suggestionLimit {
		return suggestionLimit
	}

	return limit
}

func suggestions(app *pocketbase.PocketBase, query string, limit int) ([]components.KeyboardSuggestion, error) {
	params := dbx.Params{"query": query}
	artists, err := app.FindRecordsByFilter(constants.CollectionArtists, "published = true && name ~ {:query}", "+name,+id", limit, 0, params)
	if err != nil {
		return nil, err
	}

	rows := make([]components.KeyboardSuggestion, 0, limit)
	for _, artist := range artists {
		rows = append(rows, components.KeyboardSuggestion{
			Kind:  "ARTIST",
			Label: artistLabel(artist.GetString("name"), artist.GetInt("year_of_birth"), artist.GetInt("year_of_death")),
			Href:  url.GenerateArtistUrl(url.ArtistUrlDTO{ArtistId: artist.Id, ArtistName: artist.GetString("name")}),
		})
	}
	if len(rows) >= limit {
		return rows, nil
	}

	artworks, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true && author:length > 0 && title ~ {:query}", "+title,+id", limit-len(rows), 0, params)
	if err != nil {
		return nil, err
	}
	for _, artwork := range artworks {
		authorIDs := artwork.GetStringSlice("author")
		artist, err := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
		if err != nil {
			return nil, err
		}
		rows = append(rows, components.KeyboardSuggestion{
			Kind:  "WORK",
			Label: artwork.GetString("title") + " · " + artist.GetString("name"),
			Href: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId: artist.Id, ArtistName: artist.GetString("name"), ArtworkId: artwork.Id, ArtworkTitle: artwork.GetString("title"),
			}),
		})
	}

	return rows, nil
}

func artistLabel(name string, birthYear int, deathYear int) string {
	years := []string{}
	if birthYear > 0 {
		years = append(years, strconv.Itoa(birthYear))
	}
	if deathYear > 0 {
		years = append(years, strconv.Itoa(deathYear))
	}
	if len(years) == 0 {
		return name
	}

	return fmt.Sprintf("%s · %s", name, strings.Join(years, "–"))
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	limiter := newRequestLimiter()
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/keyboard/suggestions", func(c *core.RequestEvent) error {
			if !limiter.allow(c.RealIP()) {
				return c.String(http.StatusTooManyRequests, "Too many keyboard suggestions requested.")
			}

			query := strings.TrimSpace(c.Request.URL.Query().Get("q"))
			if !validQuery(query) {
				return c.HTML(http.StatusOK, "")
			}

			rows, err := suggestions(app, query, suggestionLimitFor(c.Request.URL.Query().Get("limit")))
			if err != nil {
				app.Logger().Error("Keyboard suggestion lookup failed", "error", err)
				return c.InternalServerError("Unable to look up suggestions.", nil)
			}

			var output bytes.Buffer
			if err := components.KeyboardSuggestionRows(rows).Render(context.Background(), &output); err != nil {
				return c.InternalServerError("Unable to render suggestions.", nil)
			}

			return c.HTML(http.StatusOK, output.String())
		})
		return se.Next()
	})
}

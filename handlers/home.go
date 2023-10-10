package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jellydator/ttlcache/v3"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Content struct {
	FieldName string `db:"name" json:"name"`
	Content   string `db:"content" json:"content"`
}

type counter struct {
	C int `db:"c" json:"c"`
}

func getWelcomContent(app *pocketbase.PocketBase, cache *ttlcache.Cache[string, string]) (string, error) {

	found := cache.Has("strings:welcome")

	if found {
		return cache.Get("strings:welcome").Value(), nil
	}

	record, err := app.Dao().FindFirstRecordByData("strings", "name", "welcome")

	if err != nil {
		log.Println(err)
		return "", err
	}

	result := record.Get("content")

	cache.Set("strings:welcome", result.(string), ttlcache.DefaultTTL)

	return result.(string), nil

}

func getArtistCount(app *pocketbase.PocketBase, cache *ttlcache.Cache[string, string]) (string, error) {

	key := "count:artists"

	found := cache.Has(key)

	if found {
		return cache.Get(key).Value(), nil
	}

	c := counter{}

	err := app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artists WHERE published IS true").One(&c)

	if err != nil {
		log.Println(err)
		return "0", err
	}

	result := fmt.Sprintf("%d", c.C)

	cache.Set(key, result, ttlcache.DefaultTTL)

	return result, nil

}

func getArtworkCount(app *pocketbase.PocketBase, cache *ttlcache.Cache[string, string]) (string, error) {

	key := "count:artworks"

	found := cache.Has(key)

	if found {
		return cache.Get(key).Value(), nil
	}

	c := counter{}

	err := app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artworks WHERE published IS true").One(&c)

	if err != nil {
		log.Println(err)
		return "0", err
	}

	result := fmt.Sprintf("%d", c.C)

	cache.Set(key, result, ttlcache.DefaultTTL)

	return result, nil

}

func registerHome(app *pocketbase.PocketBase, cache *ttlcache.Cache[string, string]) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/", func(c echo.Context) error {

			welcomeText, err := getWelcomContent(app, cache)

			if err != nil {
				fmt.Println(err)
			}

			artistCount, err := getArtistCount(app, cache)

			if err != nil {
				fmt.Println(err)
			}

			artworkCount, err := getArtworkCount(app, cache)

			if err != nil {
				fmt.Println(err)
			}

			isHtmx := isHtmxRequest(c)

			html := ""

			if isHtmx {

			} else {
				html, err = renderPage("home", map[string]any{
					"Content":      welcomeText,
					"ArtistCount":  artistCount,
					"ArtworkCount": artworkCount,
				})
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"blackfyre.ninja/wga/assets"
	shape "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Song struct {
	Title  string
	URL    string
	Source []string
}

type Composer struct {
	Name     string
	Date     string
	Language string
	Songs    []Song
}

type Century struct {
	Century   string
	Composers []Composer
}

func registerMusicHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("musics", func(c echo.Context) error {
			html, err := "", error(nil)
			isHtmx := utils.IsHtmxRequest(c)
			cacheKey := "musics"

			setUrl(c, "")

			found := app.Cache().Has(cacheKey)

			// TODO: implement data getter
			musicList, err := newGetComposers(app)
			// musicList := GetMusics()

			if err != nil {
				fmt.Println("Error:", err)
				return apis.NewNotFoundError("", err)
			}

			years := []string{}
			for _, century := range musicList {
				years = append(years, century.Century)
			}
			if found {
				html = app.Cache().Get(cacheKey).(string)
			} else {
				data := map[string]any{
					"Centuries": years,
					"MusicList": musicList,
				}

				if isHtmx {
					html, err = assets.RenderBlock("musics:content", data)
				} else {
					html, err = assets.RenderPageWithLayout("musics", "noLayout", data)
				}

				if err != nil {
					// or redirect to a dedicated 404 HTML page
					return apis.NewNotFoundError("", err)
				}
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.GET("musics/:name", func(c echo.Context) error {
			isHtmx := utils.IsHtmxRequest(c)
			slug := c.PathParam("name")
			cacheKey := "music:" + slug

			if isHtmx {
				cacheKey = cacheKey + "-htmx"
			}

			html := ""
			err := error(nil)

			data := map[string]any{
				"Title":    "Gregorian Chants",
				"Composer": "Anonymus",
				"Date":     "1123",
				"Source":   "anonymous_conductus.mp3",
			}

			if isHtmx {
				html, err = assets.RenderBlock("music:content", data)
			} else {
				html, err = assets.RenderPageWithLayout("musics/music", "noLayout", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			app.Cache().Set(cacheKey, html)

			c.Response().Header().Set("HX-Push-Url", "/musics/"+slug)

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

func GetMusics() []Century {
	var data []Century

	fileData, err := os.ReadFile("./assets/reference/musics.json")

	if err != nil {
		fmt.Println("Error reading file:", err)
	}

	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
	}

	return data
}

func GetParsedMusics([]Century) ([]shape.Composer) {
	var composers []shape.Composer

	for _, century := range GetMusics() {
		for _, composer := range century.Composers {

			songs := make([]shape.Song, len(composer.Songs))
			for i, song := range composer.Songs {
				songs[i] = shape.Song{
					Title: song.Title,
					URL: song.URL,
					Source: song.Source,
				}
			}
		
			composers = append(composers, shape.Composer{
				Name: composer.Name,
				Date: composer.Date,
				Language: composer.Language,
				Century: century.Century,
				Songs: songs,
			})
		}
	}

	return composers
}

func newGetComposers(app *pocketbase.PocketBase) ([]shape.Composer, error) {
	// q := app.Dao().DB().NewQuery("SELECT * FROM music_composers INNER JOIN music_songs ON music_composers.name = music_songs.music_composer_name")

	composers, err := app.Dao().FindRecordsByFilter(
		"music_composers",
		"id != null",
		"+name",
		0,
		0,
		dbx.Params{},
	)

	if err != nil {
        return nil, err
    }

	songs, err := app.Dao().FindRecordsByFilter(
		"music_songs",
		"id != null",
		"+title",
		0,
		0,
		dbx.Params{},
	)

	if err != nil {
        return nil, err
    }

	var data []shape.Composer

	// i need to group songs by composer

	for _, composer := range composers {
		var songsByComposer []shape.Song
		for _, song := range songs {
			fmt.Println(song.GetString("music_composer_name"))
			if song.GetString("music_composer_name") == composer.GetString("name") {
				fmt.Println(song.GetString("title"))
				songsByComposer = append(songsByComposer, shape.Song{
					Title: song.GetString("title"),
					URL: song.GetString("url"),
					Source: song.GetStringSlice("source"),
				})
			}
		}
		data = append(data, shape.Composer{
			Name: composer.GetString("name"),
			Date: composer.GetString("date"),
			Language: composer.GetString("language"),
			Century: composer.GetString("century"),
			Songs: songsByComposer,
		})
	}

	for _, row := range data {
		fmt.Println(row)
	}

	mockComposer := []shape.Composer{}

    return mockComposer, nil
}

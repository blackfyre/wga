package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"blackfyre.ninja/wga/assets"
	shape "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Century struct {
	Century   string
	Composers []Composer_source
}

type Composer_source struct {
	ID	     string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Date     string `db:"date" json:"date"`
	Language string `db:"language" json:"language"`
	Century  string `db:"century" json:"century"`
	Songs    []Song_source `db:"songs" json:"songs"`
}

type Song_source struct {
	Title  		string `db:"title" json:"title"`
	URL    		string `db:"url" json:"url"`
	Source 		[]string `db:"source" json:"source"`
	ComposerID  string `db:"composer_id" json:"composer_id"` // foreign key
}

type Composer_seed struct {
	ID	     string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Date     string `db:"date" json:"date"`
	Language string `db:"language" json:"language"`
	Century  string `db:"century" json:"century"`
	Songs    []Song_seed `db:"songs" json:"songs"`
}


type Song_seed struct {
	Title  		string `db:"title" json:"title"`
	URL    		string `db:"url" json:"url"`
	Source 		string `db:"source" json:"source"`
	ComposerID  string `db:"composer_id" json:"composer_id"` // foreign key
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
			musicList, err := newGetComposers(app, c)
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

func GetParsedMusics([]Century) ([]Composer_seed) {
	var composers []Composer_seed

	for _, century := range GetMusics() {
		for _, composer := range century.Composers {
			id := uuid.New().String()

			songs := make([]Song_seed, len(composer.Songs))
			for i, song := range composer.Songs {
				newSource := []string{}
				for _, source := range song.Source {
					newSource = append(newSource, utils.GetFileNameFromUrl(source, true))
				}
				newSourceStr := strings.Join(newSource, ",")
				songs[i] = Song_seed{
					Title: song.Title,
					URL: song.URL,
					Source: newSourceStr,
					ComposerID: id,
				}
			}
		
			composers = append(composers, Composer_seed{
				ID: id,
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

func newGetComposers(app *pocketbase.PocketBase, c echo.Context) ([]shape.Music_composer, error) {
	
	// Then you can create an Expression like this:
	// expression := MyExpression("music_composer.id = music_song.composer_id")
	composers := []shape.Music_composer{}
	// i want to build this expression music_composer.id = music_song.composer_id
	// expression := Expression.Build(app.Dao(), Params{})
	// query := app.Dao().DB().Select().InnerJoin("music_song", dbx.NewExp("music_composer.id = music_song.composer_id")).From("music_composer", "music_song")
	// query := app.Dao().DB().Select()
	err := app.Dao().DB().NewQuery("SELECT * FROM music_song INNER JOIN music_composer ON music_composer.id = music_song.composer_id").All(&composers)

	// err := query.All(&composers)

	// TODO: get composers
	// TODO: get songs for each composer


	if err != nil {
		log.Println("Error executing query", err)
		return nil, errors.New("Error executing query")
	}

	for _, row := range composers {
		fmt.Println(row)
		for _, song := range row.Songs {
			fmt.Println(song.Title)
		}
	}


	// composers, err := app.Dao().FindRecordsByFilter(
	// 	"music_composers",
	// 	"id != null",
	// 	"+name",
	// 	0,
	// 	0,
	// 	dbx.Params{},
	// )

	// if err != nil {
    //     return nil, err
    // }

	// songs, err := app.Dao().FindRecordsByFilter(
	// 	"music_songs",
	// 	"id != null",
	// 	"+title",
	// 	0,
	// 	0,
	// 	dbx.Params{},
	// )

	// if err != nil {
    //     return nil, err
    // }

	// var data []shape.Music_composer

	// // i need to group songs by composer

	// for _, composer := range composers {
	// 	var songsByComposer []shape.Song
	// 	for _, song := range songs {
	// 		fmt.Println(song.GetString("music_composer_name"))
	// 		if song.GetString("music_composer_name") == composer.GetString("name") {
	// 			fmt.Println(song.GetString("title"))
	// 			songsByComposer = append(songsByComposer, shape.Song{
	// 				Title: song.GetString("title"),
	// 				URL: song.GetString("url"),
	// 				Source: song.GetStringSlice("source"),
	// 			})
	// 		}
	// 	}
	// 	data = append(data, shape.Music_composer{
	// 		Name: composer.GetString("name"),
	// 		Date: composer.GetString("date"),
	// 		Language: composer.GetString("language"),
	// 		Century: composer.GetString("century"),
	// 		Songs: songsByComposer,
	// 	})
	// }

	// for _, row := range data {
	// 	fmt.Println(row)
	// }

	mockComposer := []shape.Music_composer{}

    return mockComposer, nil
}

func MyExpression(me string) dbx.Expression {
	// Logic to convert the expression into a SQL fragment goes here.
	// This is just a placeholder.
	myNewExpression := dbx.NewExp(me)
	return myNewExpression
}

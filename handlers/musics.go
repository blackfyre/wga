package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"blackfyre.ninja/wga/assets"
	shape "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

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

type Grouped_music_list struct {
	Century string
	Composers []shape.Music_composer
}


func registerMusicHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("musics", func(c echo.Context) error {
			html, err := "", error(nil)
			isHtmx := utils.IsHtmxRequest(c)
			cacheKey := "musics"

			setUrl(c, "")

			found := app.Cache().Has(cacheKey)

			musicList, err := newGetComposers(app, c)

			if err != nil {
				fmt.Println("Error:", err)
				return apis.NewNotFoundError("", err)
			}

			years := []string{}
			seen := make(map[string]struct{})

			// make years unique
			for _, musicList := range musicList {
				if _, ok := seen[musicList.Century]; !ok {
					years = append(years, musicList.Century)
					seen[musicList.Century] = struct{}{}
				}
			}

			groupedMusicListByCenturies := GroupAndSortMusicByCentury(musicList)

			if found {
				html = app.Cache().Get(cacheKey).(string)
			} else {
				data := map[string]any{
					"Centuries": years,
					"MusicList": groupedMusicListByCenturies,
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
    composers := []shape.Music_composer{}
    err := app.Dao().DB().NewQuery("SELECT * FROM music_composer").All(&composers)
    if err != nil {
        log.Println("error executing query", err)
        return nil, errors.New("error executing query")
    }

    for i, composer := range composers {
        songs := []shape.Music_song{}

		query := fmt.Sprintf("SELECT * FROM music_song WHERE composer_id = '%s'", composer.ID)
        err := app.Dao().DB().NewQuery(query).All(&songs)
        if err != nil {
            return nil, errors.New("error executing query")
        }

        composers[i].Songs = songs
    }

    return composers, nil
}

func GroupAndSortMusicByCentury(musicList []shape.Music_composer) []Grouped_music_list {
    groupedMusicListItemsByCenturies := make(map[string][]shape.Music_composer)
    for _, music := range musicList {
        groupedMusicListItemsByCenturies[music.Century] = append(groupedMusicListItemsByCenturies[music.Century], music)
    }

    groupedMusicListByCenturies := make([]Grouped_music_list, 0, len(groupedMusicListItemsByCenturies))
    for century, composers := range groupedMusicListItemsByCenturies {
        groupedMusicListByCenturies = append(groupedMusicListByCenturies, Grouped_music_list{
            Century:   century,
            Composers: composers,
        })
    }

    sort.Slice(groupedMusicListByCenturies, func(i, j int) bool {
        return groupedMusicListByCenturies[i].Century < groupedMusicListByCenturies[j].Century
    })

    return groupedMusicListByCenturies
}

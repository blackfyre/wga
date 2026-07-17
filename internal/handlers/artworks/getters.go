package artworks

import (
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
)

const artworkSearchOptionsTTL = 6 * time.Hour

const (
	artTypesCacheKey    = "artworks:search:art-types"
	artFormsCacheKey    = "artworks:search:art-forms"
	artSchoolsCacheKey  = "artworks:search:art-schools"
	artistNamesCacheKey = "artworks:search:artist-names"
)

// getArtTypesOptions returns a map of art type slugs and their corresponding names.
// It retrieves the art types from the database using the provided PocketBase app instance.
func getArtTypesOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artTypesCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionArtTypes, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artTypesCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

// getArtFormOptions returns a map of art form slugs to their corresponding names.
// It retrieves the art forms from the database using the provided PocketBase app instance.
func getArtFormOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artFormsCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionArtForms, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artFormsCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

// getArtSchoolOptions returns a map of art school options where the key is the slug and the value is the name.
func getArtSchoolOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artSchoolsCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionSchools, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artSchoolsCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

func GetArtistNameList(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artistNamesCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	names := make(map[string]string) // Initialize the names map
	c, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		"published = true",
		"+name",
		0,
		0,
	)

	if err != nil {
		return names, err
	}

	for _, v := range c {
		names[url.GenerateArtistUrl(url.ArtistUrlDTO{
			ArtistId:   v.GetString("id"),
			ArtistName: v.GetString("name"),
		})] = v.GetString("name")
	}

	utils.SetCachedValue(app, artistNamesCacheKey, cloneStringMap(names), artworkSearchOptionsTTL)

	return names, nil
}

func cloneStringMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))

	for key, value := range source {
		clone[key] = value
	}

	return clone
}

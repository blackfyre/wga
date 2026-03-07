package artworks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
)

// getArtTypesOptions returns a map of art type slugs and their corresponding names.
// It retrieves the art types from the database using the provided PocketBase app instance.
func getArtTypesOptions(app *pocketbase.PocketBase) (map[string]string, error) {
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

	return options, nil
}

// getArtFormOptions returns a map of art form slugs to their corresponding names.
// It retrieves the art forms from the database using the provided PocketBase app instance.
func getArtFormOptions(app *pocketbase.PocketBase) (map[string]string, error) {
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

	return options, nil
}

// getArtSchoolOptions returns a map of art school options where the key is the slug and the value is the name.
func getArtSchoolOptions(app *pocketbase.PocketBase) (map[string]string, error) {
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

	return options, nil
}

func GetArtistNameList(app *pocketbase.PocketBase) (map[string]string, error) {
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

	return names, nil
}

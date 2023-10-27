package search

import (
	"blackfyre.ninja/wga/models"
	"github.com/pocketbase/pocketbase"
)

// getArtTypesOptions returns a map of art type slugs and their corresponding names.
// It retrieves the art types from the database using the provided PocketBase app instance.
func getArtTypesOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	options := map[string]string{}
	c, err := models.GetArtTypes(app.Dao())

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

// getArtFormOptions returns a map of art form slugs to their corresponding names.
// It retrieves the art forms from the database using the provided PocketBase app instance.
func getArtFormOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	options := map[string]string{}
	c, err := models.GetArtForms(app.Dao())

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

// getArtSchoolOptions returns a map of art school options where the key is the slug and the value is the name.
func getArtSchoolOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	options := map[string]string{}
	c, err := models.GetSchools(app.Dao())

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

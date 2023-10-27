package search

import (
	"blackfyre.ninja/wga/models"
	"github.com/pocketbase/pocketbase"
)

// getArtTypesOptions returns a map of art type slugs and their corresponding names.
// It retrieves the art types from the database using the provided PocketBase app instance.
func getArtTypesOptions(app *pocketbase.PocketBase) (map[string]string, error) {

	c, err := models.GetArtTypes(app.Dao())

	if err != nil {
		return nil, err
	}

	options := make(map[string]string)

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

// getArtFormOptions returns a map of art form slugs to their corresponding names.
// It retrieves the art forms from the database using the provided PocketBase app instance.
func getArtFormOptions(app *pocketbase.PocketBase) (map[string]string, error) {

	c, err := models.GetArtForms(app.Dao())

	if err != nil {
		return nil, err
	}

	options := make(map[string]string)

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

func getArtSchoolOptions(app *pocketbase.PocketBase) (map[string]string, error) {

	c, err := models.GetSchools(app.Dao())

	if err != nil {
		return nil, err
	}

	options := make(map[string]string)

	for _, v := range c {
		options[v.Slug] = v.Name
	}

	return options, nil
}

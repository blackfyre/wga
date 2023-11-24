package seed

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func SeedImages(app *pocketbase.PocketBase) error {

	portraitLocal, err := assets.InternalFiles.ReadFile("reference/wga_placeholder_portrait.jpg")

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	landscapeLocal, err := assets.InternalFiles.ReadFile("reference/wga_placeholder_landscape.jpg")

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	rfs, err := app.NewFilesystem()

	if err != nil {
		return err
	}

	defer func() {
		err = rfs.Close()

		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}()

	artworks, err := models.GetArtowrks(app.Dao())

	if err != nil {
		return err
	}

	log.Printf("Found %d artworks", len(artworks))

	// start overall timer here
	startTime := time.Now()

	for i, artwork := range artworks {
		// timer start here

		uploadKey := fmt.Sprintf("artworks/%s/%s", artwork.Id, artwork.Image)

		var img []byte

		// Randomly generate a number between 1 and 10
		randomNumber := rand.Intn(10) + 1

		// If the number is even, use the portrait image
		if randomNumber%2 == 0 {
			img = portraitLocal
		} else {
			img = landscapeLocal
		}

		err = rfs.Upload(img, uploadKey)

		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		err = generateThunmbnail(artwork, rfs, "100x100")

		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		err = generateThunmbnail(artwork, rfs, "320x240")

		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		if i%200 == 0 {
			log.Printf("Uploaded %d images", i)
			log.Printf("Elapsed time: %s", time.Since(startTime))
			log.Printf("Estimated time remaining: %s", time.Duration((len(artworks)-i)/200)*time.Minute)
		}
	}

	// end overall timer here
	endTime := time.Now()

	// calculate overall time here
	elapsed := endTime.Sub(startTime)

	log.Printf("Elapsed time: %s", elapsed)

	return nil
}

// generateThumbnail generates a thumbnail for the given artwork.
// It takes an Artwork pointer, a System pointer, and a size string as parameters.
// The uploadKey is generated using the artwork's ID and image name.
// It creates a thumbnail using the CreateThumb method of the System object.
// The thumbnail is saved with a filename that includes the size and original image name.
// If an error occurs during the thumbnail generation, it is printed and returned.
// Otherwise, nil is returned.
func generateThunmbnail(aw *models.Artwork, rfs *filesystem.System, size string) error {

	uploadKey := fmt.Sprintf("artworks/%s/%s", aw.Id, aw.Image)

	err := rfs.CreateThumb(uploadKey, fmt.Sprintf("artworks/%s/thumb_%s/%s_%s", aw.Id, aw.Image, size, aw.Image), size)

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil

}

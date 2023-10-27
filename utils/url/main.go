package url

import (
	"strings"

	"github.com/pocketbase/pocketbase"
)

func GenerateFileUrl(app *pocketbase.PocketBase, collection string, collectionId string, fileName string) string {

	endPoint := app.Settings().S3.Endpoint

	endPoint = strings.Replace(endPoint, "https://", "https://"+app.Settings().S3.Bucket+".", 1)

	return endPoint + "/" + collection + "/" + collectionId + "/" + fileName
}

func GenerateThumbUrl(app *pocketbase.PocketBase, collection string, collectionId string, fileName string, thumbSize string) string {

	endPoint := app.Settings().S3.Endpoint

	endPoint = strings.Replace(endPoint, "https://", "https://"+app.Settings().S3.Bucket+".", 1)

	return endPoint + "/" + collection + "/" + collectionId + "/thumb_" + fileName + "/" + thumbSize + "_" + fileName
}

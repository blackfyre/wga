package url

import (
	"fmt"
	"net/url"
)

func GenerateFileUrl(collection string, collectionId string, fileName string, token string) string {

	return fmt.Sprintf(
		"/api/files/%s/%s/%s?token=%s",
		collection,
		collectionId,
		fileName,
		url.QueryEscape(token),
	)
}

func GenerateThumbUrl(collection string, collectionId string, fileName string, thumbSize string, token string) string {

	return fmt.Sprintf(
		"/api/files/%s/%s/%s?token=%s&thumb=%s",
		collection,
		collectionId,
		fileName,
		url.QueryEscape(token),
		thumbSize,
	)
}

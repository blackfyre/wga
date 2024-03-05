package utils

import (
	"context"
)

type ContextKey string

var TitleKey ContextKey = "title"
var DescriptionKey ContextKey = "description"
var EnvironmentKey ContextKey = "environment"
var OgTitleKey ContextKey = "og:title"
var OgDescriptionKey ContextKey = "og:description"
var OgImageKey ContextKey = "og:image"
var OgUrlKey ContextKey = "og:url"
var OgTypeKey ContextKey = "og:type"
var OgSiteNameKey ContextKey = "og:site_name"
var TwitterCardKey ContextKey = "twitter:card"
var TwitterSiteKey ContextKey = "twitter:site"
var TwitterCreatorKey ContextKey = "twitter:creator"
var TwitterTitleKey ContextKey = "twitter:title"
var TwitterDescriptionKey ContextKey = "twitter:description"
var TwitterImageKey ContextKey = "twitter:image"

var ctx context.Context

// GetTitle retrieves the title from the context.
// If the title is found, it returns the title as a string.
// If the title is not found, it returns an empty string.
func GetTitle(c context.Context) string {
	if v, ok := c.Value(TitleKey).(string); ok {
		return v + " - WGA"
	}

	return "Web Gallery of Art"
}

// GetDescription retrieves the description value from the context.
// If the value is found and is of type string, it is returned.
// Otherwise, an empty string is returned.
func GetDescription(c context.Context) string {
	if v, ok := c.Value(DescriptionKey).(string); ok {
		return v
	}

	return ""
}

// GetEnvironment returns the environment value from the given context.
// If the environment value is not found in the context, it returns "dev" as the default value.
func GetEnvironment(c context.Context) string {
	if v, ok := c.Value(EnvironmentKey).(string); ok {
		return v
	}

	return "dev"
}

func GetOpenGraphTags(c context.Context) map[string]string {
	ogTags := make(map[string]string)

	if v, ok := c.Value(OgTitleKey).(string); ok {
		ogTags["og:title"] = v
	}

	if v, ok := c.Value(OgDescriptionKey).(string); ok {
		ogTags["og:description"] = v
	}

	if v, ok := c.Value(OgImageKey).(string); ok {
		ogTags["og:image"] = v
	}

	if v, ok := c.Value(OgUrlKey).(string); ok {
		ogTags["og:url"] = v
	}

	if v, ok := c.Value(OgTypeKey).(string); ok {
		ogTags["og:type"] = v
	}

	if v, ok := c.Value(OgSiteNameKey).(string); ok {
		ogTags["og:site_name"] = v
	}

	return ogTags
}

func GetTwitterTags(c context.Context) map[string]string {
	twitterTags := make(map[string]string)

	if v, ok := c.Value(TwitterCardKey).(string); ok {
		twitterTags["twitter:card"] = v
	}

	if v, ok := c.Value(TwitterSiteKey).(string); ok {
		twitterTags["twitter:site"] = v
	}

	if v, ok := c.Value(TwitterCreatorKey).(string); ok {
		twitterTags["twitter:creator"] = v
	}

	if v, ok := c.Value(TwitterTitleKey).(string); ok {
		twitterTags["twitter:title"] = v
	}

	if v, ok := c.Value(TwitterDescriptionKey).(string); ok {
		twitterTags["twitter:description"] = v
	}

	if v, ok := c.Value(TwitterImageKey).(string); ok {
		twitterTags["twitter:image"] = v
	}

	return twitterTags
}

// DecorateContext decorates the given context with a key-value pair.
// It returns a new context with the provided key-value pair added.
func DecorateContext(c context.Context, k ContextKey, v string) context.Context {

	if k == TitleKey || k == OgTitleKey || k == TwitterTitleKey {
		cwv := context.WithValue(c, TitleKey, v)
		cwv = context.WithValue(cwv, OgTitleKey, v)
		cwv = context.WithValue(cwv, TwitterTitleKey, v)
		return cwv
	}

	if k == DescriptionKey || k == OgDescriptionKey || k == TwitterDescriptionKey {
		if len(v) > 160 {
			v = v[:160]
		}

		cwv := context.WithValue(c, DescriptionKey, v)
		cwv = context.WithValue(cwv, OgDescriptionKey, v)
		cwv = context.WithValue(cwv, TwitterDescriptionKey, v)
		return cwv
	}

	if k == OgImageKey || k == TwitterImageKey {
		cwv := context.WithValue(c, OgImageKey, v)
		cwv = context.WithValue(cwv, TwitterImageKey, v)
		return cwv
	}

	return context.WithValue(c, k, v)
}

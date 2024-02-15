package utils

import (
	"context"
)

type ContextKey string

var TitleKey ContextKey = "title"
var DescriptionKey ContextKey = "description"
var EnvironmentKey ContextKey = "environment"

// GetTitle retrieves the title from the context.
// If the title is found, it returns the title as a string.
// If the title is not found, it returns an empty string.
func GetTitle(c context.Context) string {
	if v, ok := c.Value(TitleKey).(string); ok {
		return v
	}

	return ""
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

// DecorateContext decorates the given context with a key-value pair.
// It returns a new context with the provided key-value pair added.
func DecorateContext(c context.Context, k ContextKey, v string) context.Context {
	return context.WithValue(c, k, v)
}

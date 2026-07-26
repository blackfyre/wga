package contributors

import "context"

const (
	SnapshotSourceCache        Source = "cache"
	SnapshotSourceFileFallback Source = "file_fallback"
)

type Source string

type Contributor struct {
	Login         string `json:"login"`
	AvatarURL     string `json:"avatar_url"`
	HTMLURL       string `json:"html_url"`
	Contributions int    `json:"contributions"`
}

type Snapshot struct {
	Contributors []Contributor
	Source       Source
}

type Reader interface {
	Current(context.Context) (Snapshot, error)
}

type Provider interface {
	Fetch(context.Context) ([]Contributor, error)
}

type RefreshJob interface {
	Run(context.Context, string) error
}

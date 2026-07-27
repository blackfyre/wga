package contributors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const githubContributorsURL = "https://api.github.com/repos/blackfyre/wga/contributors"

type ProviderErrorKind string

const (
	ProviderErrorTimeout     ProviderErrorKind = "timeout"
	ProviderErrorUnavailable ProviderErrorKind = "unavailable"
	ProviderErrorContract    ProviderErrorKind = "contract"
)

type ProviderError struct {
	Kind      ProviderErrorKind
	Retryable bool
}

func (e *ProviderError) Error() string {
	return string(e.Kind)
}

type githubProvider struct {
	client *http.Client
	url    string
}

func NewGitHubProvider(client *http.Client) Provider {
	return newGitHubProvider(client, githubContributorsURL)
}

func newGitHubProvider(client *http.Client, url string) Provider {
	return &githubProvider{client: client, url: url}
}

func (p *githubProvider) Fetch(ctx context.Context) ([]Contributor, error) {
	nextURL := p.url
	seen := map[string]bool{}
	var contributors []Contributor

	for nextURL != "" {
		if seen[nextURL] {
			return nil, &ProviderError{Kind: ProviderErrorContract}
		}
		seen[nextURL] = true

		page, next, err := p.fetchPage(ctx, nextURL)
		if err != nil {
			return nil, err
		}
		contributors = append(contributors, page...)
		nextURL = next
	}

	if len(contributors) == 0 {
		return nil, &ProviderError{Kind: ProviderErrorContract}
	}

	return contributors, nil
}

func (p *githubProvider) fetchPage(ctx context.Context, endpoint string) ([]Contributor, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", &ProviderError{Kind: ProviderErrorContract}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "blackfyre/wga")

	resp, err := p.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, "", &ProviderError{Kind: ProviderErrorTimeout, Retryable: true}
		}
		return nil, "", &ProviderError{Kind: ProviderErrorUnavailable, Retryable: true}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return nil, "", &ProviderError{Kind: ProviderErrorUnavailable, Retryable: true}
		}
		return nil, "", &ProviderError{Kind: ProviderErrorContract}
	}

	var contributors []Contributor
	if err := json.NewDecoder(resp.Body).Decode(&contributors); err != nil {
		return nil, "", &ProviderError{Kind: ProviderErrorContract}
	}
	next, err := githubNextPage(resp.Header.Get("Link"), endpoint)
	if err != nil {
		return nil, "", &ProviderError{Kind: ProviderErrorContract}
	}

	return contributors, next, nil
}

func githubNextPage(linkHeader string, currentURL string) (string, error) {
	for _, link := range strings.Split(linkHeader, ",") {
		parts := strings.Split(link, ";")
		isNext := false
		for _, attribute := range parts[1:] {
			if strings.TrimSpace(attribute) == `rel="next"` {
				isNext = true
				break
			}
		}
		if !isNext {
			continue
		}

		value := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(value, "<") || !strings.HasSuffix(value, ">") {
			return "", errors.New("invalid next page link")
		}
		next, err := url.Parse(strings.TrimSuffix(strings.TrimPrefix(value, "<"), ">"))
		if err != nil {
			return "", err
		}
		base, err := url.Parse(currentURL)
		if err != nil {
			return "", err
		}
		if next.IsAbs() && (next.Scheme != base.Scheme || next.Host != base.Host) {
			return "", errors.New("next page host differs from provider")
		}

		return base.ResolveReference(next).String(), nil
	}

	return "", nil
}

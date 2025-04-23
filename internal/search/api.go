package search

import "context"

type SearchEngine interface {
	Search(ctx context.Context, query string, queryParams QueryParams) (*SearchResult, error)
}

type QueryParams struct{}

type SearchResult struct {
	Items []SearchResultItems
}

type SearchResultItems struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

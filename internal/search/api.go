package search

import "context"

type SearchEngine interface {
	Search(ctx context.Context, query string, queryParams QueryParams) (*SearchResult, error)
}

type QueryParams struct{}

type SearchResult struct {
	Items []struct {
		Kind    string `json:"kind"`
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
	SearchInformation struct {
		TotalResults string `json:"totalResults"`
	} `json:"searchInformation"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

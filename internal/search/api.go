package search

import "context"

type SearchEngine interface {
	Search(ctx context.Context, query string, queryParams QueryParams) (SearchResult, error)
}

type QueryParams struct{}

type SearchResult struct{}

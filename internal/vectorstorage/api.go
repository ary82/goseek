package vectorstorage

import "context"

type VectorStore interface {
	UpsertRecords(ctx context.Context, records any, ns string) error
	SearchTopK(ctx context.Context, query string, k int, ns string) (any, error)
}

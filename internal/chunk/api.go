package chunk

import (
	"context"
)

type Chunker interface {
	Chunk(ctx context.Context, str string) ([]Chunk, error)
}

type Chunk struct {
	Content    string
	TokenCount int
}

package chunk

import (
	"context"
)

type Chunker interface {
	Chunk(ctx context.Context, link string, str string) ([]Chunk, error)
}

type Chunk struct {
	Link       string
	Content    string
	TokenCount int
}

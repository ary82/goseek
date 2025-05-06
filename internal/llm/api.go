package llm

import "context"

type LLM interface {
	GenerateContent(ctx context.Context, prompt string) (*string, error)
}

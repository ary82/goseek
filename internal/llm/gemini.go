package llm

import (
	"context"
	"log"

	"google.golang.org/genai"
)

type Gemini struct {
	client *genai.Client
}

func NewGeminiLLM(ctx context.Context, key string) (LLM, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  key,
	})
	if err != nil {
		return nil, err
	}
	return &Gemini{
		client: client,
	}, nil
}

func (g *Gemini) GenerateContent(ctx context.Context, prompt string) (*string, error) {
	result, err := g.client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text(prompt),
		&genai.GenerateContentConfig{},
	)
	if err != nil {
		return nil, err
	}

	res := result.Text()
	log.Printf("generation succeeded")
	return &res, nil
}

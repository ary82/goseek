package scrape

import "context"

type Scraper interface {
	Scrape(ctx context.Context, urls []string) (map[string]ScrapedContent, error)
}

type ScrapedContent struct {
	Content string
	URL     string
	Error   error
}

type Content struct{}

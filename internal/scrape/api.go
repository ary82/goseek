package scrape

import "context"

type Scraper interface {
	Scrape(ctx context.Context, urls []string) (ScrapedContent, error)
}

type ScrapedContent struct{}

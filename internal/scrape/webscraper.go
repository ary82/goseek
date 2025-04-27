package scrape

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type webScraper struct {
	client     *http.Client
	userAgent  string
	maxWorkers int
}

func NewWebScraper() Scraper {
	return &webScraper{}
}

func (w *webScraper) Scrape(ctx context.Context, urls []string) (map[string]ScrapedContent, error) {
	results := make(map[string]ScrapedContent)
	resultsMu := sync.Mutex{}

	// Create worker pool
	workCh := make(chan string)
	wg := sync.WaitGroup{}

	// Start workers
	for range w.maxWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range workCh {
				content, err := w.scrapeURL(ctx, url)
				resultsMu.Lock()
				results[url] = ScrapedContent{
					URL:     url,
					Content: content,
					Error:   err,
				}
				resultsMu.Unlock()
			}
		}()
	}

	// Feed URLs to workers
	for _, url := range urls {
		select {
		case workCh <- url:
		case <-ctx.Done():
			close(workCh)
			return nil, ctx.Err()
		}
	}
	close(workCh)
	wg.Wait()

	// Filter out errors
	validResults := make(map[string]ScrapedContent)
	for url, result := range results {
		if result.Error == nil && result.Content != "" {
			validResults[url] = result
		}
	}

	return validResults, nil
}

func (w *webScraper) scrapeURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", w.userAgent)
	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("error closing response body: %w", err)
	}

	// Extract body text
	bodyText := doc.Find("body").Text()
	bodyText = strings.Join(strings.Fields(bodyText), " ")

	if len(bodyText) < 100 {
		return "", fmt.Errorf("body text too short (%d chars)", len(bodyText))
	}

	return bodyText, nil
}

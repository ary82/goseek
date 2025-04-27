package scrape

import (
	"context"
	"net/http"
	"testing"
)

func Test_webScraper_scrapeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "test single scrape",
			url:     "https://www.pnnl.gov/explainer-articles/nanomaterials",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := webScraper{
				client:    &http.Client{},
				userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
			}
			got, gotErr := w.scrapeURL(context.Background(), tt.url)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("scrapeURL() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("scrapeURL() succeeded unexpectedly")
			}
			t.Logf("scrapeURL(): url: %v, length: %v", tt.url, len(got))
		})
	}
}

func Test_webScraper_Scrape(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		wantErr bool
	}{
		{
			name: "test multithread scrape",
			urls: []string{
				"https://revolutionized.com/nanomaterials/",
				"https://en.wikipedia.org/wiki/Nanomaterials",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := webScraper{
				client:     &http.Client{},
				userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
				maxWorkers: 4,
			}
			got, gotErr := w.Scrape(context.Background(), tt.urls)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Scrape() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Scrape() succeeded unexpectedly")
			}
			for i, v := range got {
				t.Logf("Scrape() result: {url: %v, content_url: %v, content_len: %v, err: %v}", i, v.URL, len(v.Content), v.Error)
			}
		})
	}
}

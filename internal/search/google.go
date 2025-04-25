package search

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

type googleSearchEngine struct {
	Url string
	Key string
	Cx  string
}

func NewGoogleSearchEngine(url string, key string) (SearchEngine, error) {
	return &googleSearchEngine{
		Url: url,
		Key: key,
	}, nil
}

func (g *googleSearchEngine) Search(ctx context.Context, query string, queryParams QueryParams) (*SearchResult, error) {
	url, err := url.Parse(g.Url)
	if err != nil {
		return nil, err
	}
	v := url.Query()
	v.Set("key", g.Key)
	v.Set("cx", g.Cx)
	v.Set("q", query)

	url.RawQuery = v.Encode()

	log.Println(url.String())
	res, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}

	var sr SearchResult
	err = json.NewDecoder(res.Body).Decode(&sr)
	if err != nil {
		return nil, err
	}

	return &sr, nil
}

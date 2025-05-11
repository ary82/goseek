package vectorstorage

import (
	"context"
	"fmt"
	"log"

	"github.com/pinecone-io/go-pinecone/v3/pinecone"
)

type PineconeStorage struct {
	Pc   *pinecone.Client
	Idx  string
	Host string
}

func NewPineconeStorage(key string, host string) (VectorStore, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Client: %v", err)
	}

	return &PineconeStorage{
		Pc:   pc,
		Idx:  "goseek",
		Host: host,
	}, nil
}

func (ps *PineconeStorage) UpsertRecords(ctx context.Context, records any, ns string) error {
	idxConnection, err := ps.Pc.Index(pinecone.NewIndexConnParams{Host: ps.Host, Namespace: ns})
	if err != nil {
		return fmt.Errorf("failed to create IndexConnection for Host: %v: %v", ps.Host, err)
	}

	err = idxConnection.UpsertRecords(ctx, records.([]*pinecone.IntegratedRecord))
	if err != nil {
		return fmt.Errorf("failed to upsert vectors: %v", err)
	}

	log.Printf("upsert succeeded")
	return nil
}

func (ps *PineconeStorage) SearchTopK(ctx context.Context, query string, k int, ns string) (any, error) {
	idxConnection, err := ps.Pc.Index(pinecone.NewIndexConnParams{Host: ps.Host, Namespace: ns})
	if err != nil {
		return nil, fmt.Errorf("failed to create IndexConnection for Host: %v: %v", ps.Host, err)
	}

	res, err := idxConnection.SearchRecords(ctx, &pinecone.SearchRecordsRequest{
		Query: pinecone.SearchRecordsQuery{
			TopK: int32(k),
			Inputs: &map[string]any{
				"text": query,
			},
		},
		Fields: &[]string{"text", "link"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %v", err)
	}

	log.Printf("vectorsearch succeeded with %v results", len(res.Result.Hits))
	return res, err
}

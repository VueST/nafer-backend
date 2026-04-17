package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/meilisearch/meilisearch-go"

	"nafer/search/internal/domain"
)

const indexName = "media"

// SearchService manages Meilisearch indexing and querying.
// It has no database — Meilisearch IS the data store for search.
type SearchService struct {
	client *meilisearch.Client
	log    *slog.Logger
}

// NewSearchService constructs the service and configures the Meilisearch index.
func NewSearchService(client *meilisearch.Client, log *slog.Logger) (*SearchService, error) {
	svc := &SearchService{client: client, log: log}
	if err := svc.ensureIndex(context.Background()); err != nil {
		return nil, err
	}
	return svc, nil
}

// ensureIndex creates the media index and configures filterable / searchable attributes.
func (s *SearchService) ensureIndex(ctx context.Context) error {
	_, err := s.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexName,
		PrimaryKey: "id",
	})
	// Meilisearch returns an error if the index already exists — that's OK to ignore.
	if err != nil {
		s.log.Info("index may already exist, continuing", "index", indexName)
	}

	index := s.client.Index(indexName)

	// Set searchable attributes (which fields Meilisearch full-text searches)
	if _, err := index.UpdateSearchableAttributes(&[]string{"title", "description", "tags"}); err != nil {
		return fmt.Errorf("setting searchable attributes: %w", err)
	}

	// Set filterable attributes (usable in filters)
	if _, err := index.UpdateFilterableAttributes(&[]string{"media_type", "uploader_id"}); err != nil {
		return fmt.Errorf("setting filterable attributes: %w", err)
	}

	// Set sortable attributes
	if _, err := index.UpdateSortableAttributes(&[]string{"created_at"}); err != nil {
		return fmt.Errorf("setting sortable attributes: %w", err)
	}

	return nil
}

// IndexMedia adds or updates a media document in the search index.
func (s *SearchService) IndexMedia(ctx context.Context, media *domain.IndexedMedia) error {
	index := s.client.Index(indexName)
	_, err := index.AddDocuments([]domain.IndexedMedia{*media}, "id")
	if err != nil {
		return fmt.Errorf("indexing media %s: %w", media.ID, err)
	}
	return nil
}

// DeleteMedia removes a media document from the search index.
func (s *SearchService) DeleteMedia(ctx context.Context, mediaID string) error {
	index := s.client.Index(indexName)
	_, err := index.DeleteDocument(mediaID)
	return err
}

// Search performs a full-text query against the media index.
func (s *SearchService) Search(ctx context.Context, query string, mediaType string, page, hitsPerPage int) (*domain.SearchResult, error) {
	if hitsPerPage <= 0 || hitsPerPage > 100 {
		hitsPerPage = 20
	}
	if page <= 0 {
		page = 1
	}

	req := &meilisearch.SearchRequest{
		HitsPerPage: int64(hitsPerPage),
		Page:        int64(page),
		Sort:        []string{"created_at:desc"},
	}
	if mediaType != "" {
		req.Filter = fmt.Sprintf("media_type = '%s'", mediaType)
	}

	index := s.client.Index(indexName)
	raw, err := index.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("searching: %w", err)
	}

	hits := make([]domain.IndexedMedia, 0, len(raw.Hits))
	for _, h := range raw.Hits {
		// Meilisearch returns hits as map[string]interface{} — re-serialize to get typed struct
		m, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		item := domain.IndexedMedia{
			ID:          stringVal(m, "id"),
			Title:       stringVal(m, "title"),
			Description: stringVal(m, "description"),
			UploaderID:  stringVal(m, "uploader_id"),
			MediaType:   stringVal(m, "media_type"),
			CreatedAt:   stringVal(m, "created_at"),
		}
		if tags, ok := m["tags"].([]interface{}); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					item.Tags = append(item.Tags, s)
				}
			}
		}
		hits = append(hits, item)
	}

	return &domain.SearchResult{
		Hits:        hits,
		Query:       query,
		TotalHits:   raw.TotalHits,
		HitsPerPage: hitsPerPage,
		Page:        page,
	}, nil
}

func stringVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

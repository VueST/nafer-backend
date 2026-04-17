package domain

// IndexedMedia represents a media document stored in Meilisearch.
// This is the shape of each searchable document.
type IndexedMedia struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	UploaderID  string   `json:"uploader_id"`
	MediaType   string   `json:"media_type"` // "video" | "image" | "audio"
	CreatedAt   string   `json:"created_at"` // ISO 8601 string for Meilisearch
}

// SearchResult wraps a list of matched media with pagination metadata.
type SearchResult struct {
	Hits        []IndexedMedia `json:"hits"`
	Query       string         `json:"query"`
	TotalHits   int64          `json:"total_hits"`
	HitsPerPage int            `json:"hits_per_page"`
	Page        int            `json:"page"`
}

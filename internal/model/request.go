package model

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenBatchRequest []ShortenBatchRequestItem

type ShortenBatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

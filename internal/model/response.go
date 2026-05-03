package model

type ShortenResponse struct {
	Result string `json:"result" validate:"required,url"`
}

type ShortenBatchResponse []ShortenBatchResponseItem

type ShortenBatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

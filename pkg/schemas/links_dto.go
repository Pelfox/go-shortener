package schemas

// CreateShortLink описывает тело запроса для создания короткой ссылки.
type CreateShortLink struct {
	// URL это исходный URL, который нужно сократить.
	URL string `json:"url"`
}

// ShortLinkResponse описывает ответ с результатом создания короткой ссылки.
type ShortLinkResponse struct {
	// Result это сгенерированная короткая ссылка.
	Result string `json:"result"`
}

// BatchedLinkRequest описывает одну ссылку для сокращения.
type BatchedLinkRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchedLinkResponse описывает одну сокращённую ссылку.
type BatchedLinkResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

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

package schemas

type CreateShortLink struct {
	URL string `json:"url"`
}

type ShortLinkResponse struct {
	Result string `json:"result"`
}

package linkdto

type CreateLinkRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type GetLinkRequest struct {
	ShortCode string `json:"short_code"`
}

type ListLinksRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type DeleteLinkRequest struct {
	ShortCode string `json:"short_code"`
}

type GetLinkStatsRequest struct {
	ShortCode string `json:"short_code"`
}

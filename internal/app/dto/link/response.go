package linkdto

import "time"

type CreateLinkResponse struct {
	ShortCode string `json:"short_code"`
}

type LinkResponse struct {
	ID          int       `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	Visits      int       `json:"visits"`
}

type ListLinksResponse struct {
	Items []*LinkResponse `json:"items"`
	Total int             `json:"total"`
}

type LinkStatsResponse struct {
	ShortCode string    `json:"short_code"`
	URL       string    `json:"url"`
	Visits    int       `json:"visits"`
	CreatedAt time.Time `json:"created_at"`
}

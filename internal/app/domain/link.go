package domain

import "time"

type Link struct {
	ID          int       `db:"id" json:"id"`
	ShortCode   string    `db:"short_code" json:"short_code"`
	OriginalURL string    `db:"original_url" json:"original_url"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	Visits      int       `db:"visits" json:"visits"`
}

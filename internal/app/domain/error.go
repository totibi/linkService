package domain

import "errors"

var (
	ErrNotFound   = errors.New("link not found")
	ErrInvalidURL = errors.New("invalid url")
	ErrInternal   = errors.New("internal error")
)

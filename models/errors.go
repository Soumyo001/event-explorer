package models

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrNotFound            = errors.New("resource not found")
	ErrUpstreamUnavailable = errors.New("upstream provider unavailable")
	ErrInvalidTicketURL    = errors.New("ticket url failed validation")
	ErrNoCityComponent     = errors.New("place has no city component")
)

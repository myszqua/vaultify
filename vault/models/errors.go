package models

import "errors"

var (
	// ErrEmptyENVField is returned when a required environment variable is empty.
	ErrEmptyENVField = errors.New("environment variable is empty")
)

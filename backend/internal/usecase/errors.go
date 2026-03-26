package usecase

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountLocked       = errors.New("account locked")
	ErrAccountDisabled     = errors.New("account disabled")
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenInvalid        = errors.New("token invalid")
	ErrArticleNotFound     = errors.New("article not found")
	ErrAlreadyBookmarked   = errors.New("already bookmarked")
	ErrBookmarkNotFound    = errors.New("bookmark not found")
	ErrFeedbackNotFound    = errors.New("feedback not found")
	ErrInvalidFeedbackType = errors.New("invalid feedback type")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrValidation          = errors.New("validation error")
	ErrSourceNotFound      = errors.New("source not found")
	ErrFeedURLAlreadyExists = errors.New("feed_url already exists")
	ErrIngestionJobNotFound = errors.New("ingestion job not found")
)

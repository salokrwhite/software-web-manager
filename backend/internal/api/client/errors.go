package client

import "errors"

// Client-ingest scope/status sentinel errors.
var (
	ErrInsufficientScope = errors.New("insufficient scope")
	ErrAppPending        = errors.New("app_pending_review")
	ErrAppRejected       = errors.New("app_rejected")
)

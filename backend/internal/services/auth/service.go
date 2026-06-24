package auth

import (
	"software-web-manager/backend/internal/config"

	"gorm.io/gorm"
)

// Service exposes authentication commands (login, register, token refresh,
// register email codes) that are independent of the HTTP layer (no gin, no
// response writing). Failures are reported via typed errors (see Error,
// UserNotActiveError, OrgNotActiveError) so the HTTP layer can map them to the
// right status code, message, and code.
type Service struct {
	DB  *gorm.DB
	Cfg config.Config
}

// NewService builds an auth service from a database handle and config.
func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{DB: db, Cfg: cfg}
}

// Error is a typed error carrying an HTTP status, a user-facing message, and an
// optional machine-readable code.
type Error struct {
	Status  int
	Message string
	Code    string
}

func (e *Error) Error() string { return e.Message }

func newError(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

func newErrorCode(status int, message, code string) *Error {
	return &Error{Status: status, Message: message, Code: code}
}

// UserNotActiveError signals an inactive user. The raw status lets the HTTP layer
// compute the status code via middleware.UserStatusCode.
type UserNotActiveError struct {
	Status string
}

func (e *UserNotActiveError) Error() string { return "user not active" }

// OrgNotActiveError signals an inactive org. The raw status lets the HTTP layer
// compute the status code via middleware.OrgStatusCode.
type OrgNotActiveError struct {
	Status string
}

func (e *OrgNotActiveError) Error() string { return "org not active" }

package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrorKind classifies a failure so both the UI and a future MCP adapter can
// react without parsing message text.
type ErrorKind string

// Recognised error kinds.
const (
	ErrorNetwork      ErrorKind = "network"
	ErrorTimeout      ErrorKind = "timeout"
	ErrorCancelled    ErrorKind = "cancelled"
	ErrorUnauthorized ErrorKind = "unauthorized"
	ErrorForbidden    ErrorKind = "forbidden"
	ErrorNotFound     ErrorKind = "not_found"
	ErrorRateLimited  ErrorKind = "rate_limited"
	ErrorConflict     ErrorKind = "conflict"
	ErrorValidation   ErrorKind = "validation"
	ErrorServer       ErrorKind = "server"
	ErrorDecode       ErrorKind = "decode"
	ErrorUnsupported  ErrorKind = "unsupported"
	ErrorConfig       ErrorKind = "config"
	ErrorUnknown      ErrorKind = "unknown"
)

// Error is a user-facing failure. Every field is safe to render: the layer
// that constructs it is responsible for keeping credentials and raw response
// bodies out.
type Error struct {
	Kind ErrorKind
	// Title is a short headline, e.g. "Permission denied".
	Title string
	// Message explains the problem in plain language.
	Message string
	// Detail is optional technical context shown behind "view details".
	Detail string
	// Suggestion is a concrete next step the user can take.
	Suggestion string
	// Retryable indicates the same request may succeed if repeated.
	Retryable bool
	// RetryAfter is set for rate limiting when the server told us how long to wait.
	RetryAfter time.Duration
	// Operation names what failed, e.g. "list applications".
	Operation string
	// StatusCode is the HTTP status, 0 when the failure was not an HTTP response.
	StatusCode int

	wrapped error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	return e.Message
}

// Unwrap exposes the underlying cause for errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.wrapped }

// WithOperation returns a copy labelled with the operation that failed. It is
// how a service layer adds context without losing the classification.
func (e *Error) WithOperation(op string) *Error {
	if e == nil {
		return nil
	}
	c := *e
	c.Operation = op
	return &c
}

// Wrap attaches a cause without changing the user-facing text.
func (e *Error) Wrap(err error) *Error {
	if e == nil {
		return nil
	}
	c := *e
	c.wrapped = err
	return &c
}

// NewError builds an Error with the default copy for its kind.
func NewError(kind ErrorKind, cause error) *Error {
	e := &Error{Kind: kind, wrapped: cause}
	e.Title, e.Message, e.Suggestion, e.Retryable = defaultCopy(kind)
	return e
}

// AsError extracts a *Error from an error chain, classifying anything else as
// ErrorUnknown so the UI always has structured information to show.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	e := NewError(ErrorUnknown, err)
	e.Detail = err.Error()
	return e
}

// Is lets callers match on kind with errors.Is(err, domain.ErrorForbidden).
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return t.Kind == e.Kind
	}
	return false
}

// IsKind reports whether err is a domain error of the given kind.
func IsKind(err error, kind ErrorKind) bool {
	var de *Error
	return errors.As(err, &de) && de.Kind == kind
}

// defaultCopy returns the standard title, message, suggestion and retryability
// for a kind. Keeping the wording here means the same failure reads the same
// way in the TUI, in `doctor` and in a future MCP response.
func defaultCopy(kind ErrorKind) (title, message, suggestion string, retryable bool) {
	switch kind {
	case ErrorNetwork:
		return "Cannot reach Coolify",
			"The Coolify instance did not answer.",
			"Check the instance URL, your network connection and whether Coolify is running.",
			true
	case ErrorTimeout:
		return "Request timed out",
			"Coolify did not respond in time.",
			"The server may be busy. Press R to retry.",
			true
	case ErrorCancelled:
		return "Request cancelled",
			"The request was cancelled before it completed.",
			"", false
	case ErrorUnauthorized:
		return "Authentication failed",
			"Coolify rejected the API token.",
			"The token may be revoked or expired. Run `cooldeck auth add <instance>` to store a new one.",
			false
	case ErrorForbidden:
		return "Permission denied",
			"The API token is not allowed to perform this operation.",
			"Create a token with the required permission in Coolify under Keys & Tokens.",
			false
	case ErrorNotFound:
		return "Not found",
			"Coolify does not know about this resource.",
			"It may have been deleted. Press R to refresh.",
			false
	case ErrorRateLimited:
		return "Rate limit reached",
			"Coolify is throttling requests from this token.",
			"Increase refresh_interval in your config to poll less often.",
			true
	case ErrorConflict:
		return "Operation conflicts with current state",
			"Coolify refused the operation because the resource is busy or already in that state.",
			"Wait for the running operation to finish and try again.",
			false
	case ErrorValidation:
		return "Request rejected",
			"Coolify rejected the request as invalid.",
			"", false
	case ErrorServer:
		return "Coolify server error",
			"The Coolify instance returned an internal error.",
			"Check the Coolify server logs. Press R to retry.",
			true
	case ErrorDecode:
		return "Unexpected response",
			"The response from Coolify could not be understood.",
			"This usually means the Coolify version is not supported. Run `cooldeck doctor`.",
			false
	case ErrorUnsupported:
		return "Not supported",
			"This Coolify instance does not offer the requested capability.",
			"", false
	case ErrorConfig:
		return "Configuration problem",
			"The cooldeck configuration is incomplete or invalid.",
			"Run `cooldeck config validate` to see the details.",
			false
	default:
		return "Something went wrong",
			"An unexpected error occurred.",
			"", false
	}
}

package voicetel

import "fmt"

// ErrorKind classifies a VoiceTel API error so callers can switch on it
// without having to inspect HTTP status codes.
type ErrorKind int

const (
	// KindUnknown is the catch-all for unmapped statuses or transport failures.
	KindUnknown ErrorKind = iota
	// KindBadRequest — HTTP 400, server-side validation failure.
	KindBadRequest
	// KindAuthentication — HTTP 401, bearer token missing, expired, or invalid.
	KindAuthentication
	// KindPermissionDenied — HTTP 403, authenticated but not allowed.
	KindPermissionDenied
	// KindNotFound — HTTP 404, resource does not exist.
	KindNotFound
	// KindConflict — HTTP 409, request conflicts with current state.
	KindConflict
	// KindRateLimit — HTTP 429, exceeded the 6/hour/IP cap on account/* endpoints.
	KindRateLimit
	// KindServer — any HTTP 5xx.
	KindServer
)

// APIError is returned whenever the VoiceTel API responds with a non-2xx status,
// or when the transport itself fails (in which case StatusCode is 0).
//
// The Body field preserves the raw response payload — useful for 409 conflicts
// where the server returns structured detail about partial successes (see
// AclConflictData and AuthPutConflictData).
type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	Message    string
	Body       any
	cause      error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.cause != nil && e.Message == "" {
		return fmt.Sprintf("voicetel: %s", e.cause)
	}
	if e.Code != "" {
		return fmt.Sprintf("voicetel: HTTP %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("voicetel: HTTP %d: %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying transport error, if any.
func (e *APIError) Unwrap() error { return e.cause }

// IsRateLimit returns true when err is an APIError with KindRateLimit.
// Equivalent to checking e.Kind == KindRateLimit.
func IsRateLimit(err error) bool { return kindOf(err) == KindRateLimit }

// IsNotFound returns true when err is an APIError with KindNotFound.
func IsNotFound(err error) bool { return kindOf(err) == KindNotFound }

// IsAuthentication returns true when err is an APIError with KindAuthentication.
func IsAuthentication(err error) bool { return kindOf(err) == KindAuthentication }

// IsConflict returns true when err is an APIError with KindConflict.
func IsConflict(err error) bool { return kindOf(err) == KindConflict }

func kindOf(err error) ErrorKind {
	if e, ok := err.(*APIError); ok {
		return e.Kind
	}
	return KindUnknown
}

func errorFromStatus(status int, code, message string, body any) *APIError {
	return &APIError{
		Kind:       kindFromStatus(status),
		StatusCode: status,
		Code:       code,
		Message:    message,
		Body:       body,
	}
}

func kindFromStatus(status int) ErrorKind {
	switch {
	case status == 400:
		return KindBadRequest
	case status == 401:
		return KindAuthentication
	case status == 403:
		return KindPermissionDenied
	case status == 404:
		return KindNotFound
	case status == 409:
		return KindConflict
	case status == 429:
		return KindRateLimit
	case status >= 500 && status < 600:
		return KindServer
	}
	return KindUnknown
}

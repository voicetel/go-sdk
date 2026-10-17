package voicetel

import (
	"encoding/json"
	"fmt"
)

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
	// KindRateLimit — HTTP 429, exceeded a rate cap (6/hour/IP on account,
	// account/cdr, account/recurring-charges, and account/api-key; 30/hour/account
	// on messaging health checks).
	KindRateLimit
	// KindServer — any HTTP 5xx.
	KindServer
)

// APIError is returned whenever the VoiceTel API responds with a non-2xx status,
// or when the transport itself fails (in which case StatusCode is 0).
//
// Code and Message come from the spec error envelope
// {"status":"error","error":{"code":<int>,"message":<string>}}; Code is 0 when
// the server sent no machine-readable code.
//
// The Body field preserves the decoded response payload (a map[string]any for
// JSON error bodies, or the raw string otherwise). For 409 conflicts the body
// carries structured detail about partial successes — use the AclConflict and
// AuthPutConflict helpers to decode it.
type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Code       int
	Message    string
	Body       any
	cause      error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.cause != nil && e.Message == "" {
		return fmt.Sprintf("voicetel: %s", e.cause)
	}
	if e.Code != 0 {
		return fmt.Sprintf("voicetel: HTTP %d (code %d): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("voicetel: HTTP %d: %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying transport error, if any.
func (e *APIError) Unwrap() error { return e.cause }

// AclConflict decodes the structured conflict body a 409 from POST or DELETE
// /v2.2/acl carries: the entries that succeeded alongside the ones that failed.
// ok is false unless this error is a 409 with the ACL conflict envelope.
func (e *APIError) AclConflict() (data *AclConflictData, ok bool) {
	if e.Kind != KindConflict {
		return nil, false
	}
	var env struct {
		Data *AclConflictData `json:"data"`
	}
	if !e.decodeBodyInto(&env) || env.Data == nil {
		return nil, false
	}
	return env.Data, true
}

// AuthPutConflict decodes the structured conflict body a 409 from PUT
// /v2.2/auth carries: the fields applied before the conflict plus the
// per-field rejections. ok is false unless this error is a 409 with the auth
// conflict envelope.
func (e *APIError) AuthPutConflict() (data *AuthPutConflictData, ok bool) {
	if e.Kind != KindConflict {
		return nil, false
	}
	var out AuthPutConflictData
	if !e.decodeBodyInto(&out) || (len(out.Updated) == 0 && len(out.Errors) == 0) {
		return nil, false
	}
	return &out, true
}

// decodeBodyInto re-encodes the preserved Body map and unmarshals it into v.
// Returns false when Body does not hold a decoded JSON object or the shape
// does not match.
func (e *APIError) decodeBodyInto(v any) bool {
	m, ok := e.Body.(map[string]any)
	if !ok {
		return false
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

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

func errorFromStatus(status int, code int, message string, body any) *APIError {
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

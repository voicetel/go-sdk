package voicetel

import "context"

// AuthenticationService manages SIP/HTTP authentication settings (mode + password).
type AuthenticationService struct{ c *Client }

// Auth-mode constants for AuthPutRequest.AuthType and AuthGetData.AuthType.
//
//	0 = Digest, 1 = IP Auth, 2 = Digest OR IP, 3 = Digest AND IP.
const (
	AuthTypeDigest      = 0
	AuthTypeIPAuth      = 1
	AuthTypeDigestOrIP  = 2
	AuthTypeDigestAndIP = 3
)

// AuthPutRequest is the body for PUT /v2.2/auth. Pointer fields are optional.
type AuthPutRequest struct {
	AuthType *int    `json:"authType,omitempty"`
	Password *string `json:"password,omitempty"` // 6-10 alphanumeric chars; at least one letter and one number
}

// AuthGetData is the response data for GET /v2.2/auth.
type AuthGetData struct {
	AuthType            int         `json:"authType"`
	AuthTypeDescription string      `json:"authTypeDescription"`
	ACL                 []CidrEntry `json:"acl"`
}

// AuthUpdatedEntry records one field's change, returned by PUT /v2.2/auth.
type AuthUpdatedEntry struct {
	Field string `json:"field"`           // "authType" or "password"
	Value int    `json:"value,omitempty"` // present when echoing is safe (authType); omitted for password
}

// AuthPutData is the response data for PUT /v2.2/auth.
type AuthPutData struct {
	Updated []AuthUpdatedEntry `json:"updated"`
}

// AuthPutConflictData is the data payload returned in a 409 from PUT /v2.2/auth.
type AuthPutConflictData struct {
	Updated []AuthUpdatedEntry `json:"updated,omitempty"`
}

// Get returns the current auth mode + allowlist.
func (s *AuthenticationService) Get(ctx context.Context) (*AuthGetData, error) {
	var out AuthGetData
	if err := s.c.t.request(ctx, "GET", "/v2.2/auth", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update sets the auth mode and/or password.
func (s *AuthenticationService) Update(ctx context.Context, body AuthPutRequest) (*AuthPutData, error) {
	var out AuthPutData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/auth", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

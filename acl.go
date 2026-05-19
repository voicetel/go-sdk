package voicetel

import "context"

// ACLService manages the IP allowlist (CIDR entries) bound to the account.
type ACLService struct{ c *Client }

// AclModifyRequest is the body for POST /v2.2/acl (add) and DELETE /v2.2/acl (remove).
type AclModifyRequest struct {
	ACL []CidrEntry `json:"acl"`
}

// AclListData is the response data for GET /v2.2/acl.
type AclListData struct {
	ACL []CidrEntry `json:"acl"`
}

// AclAddData is the response data for POST /v2.2/acl.
type AclAddData struct {
	Added []CidrEntry `json:"added"`
}

// AclRemoveData is the response data for DELETE /v2.2/acl.
type AclRemoveData struct {
	Removed []CidrEntry `json:"removed"`
}

// AclFailedEntry is a CIDR that was rejected, with the reason.
//
// Reason is one of:
//   - "DB Insert failed"
//   - "DB delete failed"
//   - "Invalid mask: must be /8, /16, /24, or /32"
//   - "CIDR range must be routable"
type AclFailedEntry struct {
	CIDR   string `json:"cidr"`
	Reason string `json:"reason"`
}

// AclConflictData is the data payload included in a 409 response from
// POST/DELETE /v2.2/acl. It surfaces partial success: entries that succeeded
// alongside the ones that failed.
type AclConflictData struct {
	Added   []CidrEntry      `json:"added,omitempty"`
	Removed []CidrEntry      `json:"removed,omitempty"`
	Failed  []AclFailedEntry `json:"failed,omitempty"`
}

// List returns the current allowlist.
func (s *ACLService) List(ctx context.Context) (*AclListData, error) {
	var out AclListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/acl", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Add appends one or more CIDR entries to the allowlist.
func (s *ACLService) Add(ctx context.Context, body AclModifyRequest) (*AclAddData, error) {
	var out AclAddData
	if err := s.c.t.request(ctx, "POST", "/v2.2/acl", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove removes one or more CIDR entries from the allowlist.
func (s *ACLService) Remove(ctx context.Context, body AclModifyRequest) (*AclRemoveData, error) {
	var out AclRemoveData
	if err := s.c.t.request(ctx, "DELETE", "/v2.2/acl", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

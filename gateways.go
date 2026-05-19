package voicetel

import (
	"context"
	"strconv"
)

// GatewaysService manages outbound termination gateways on the account.
type GatewaysService struct{ c *Client }

// GatewayAddRequest is the body for POST /v2.2/gateways.
type GatewayAddRequest struct {
	Gateway string  `json:"gateway"`          // IP/hostname with optional :port; must be routable public IPv4
	Prefix  *string `json:"prefix,omitempty"` // digits to prepend on outbound calls
	Limit   *int    `json:"limit,omitempty"`  // max concurrent calls; default 23, range 1..1000
}

// GatewayUpdateRequest is the body for PUT /v2.2/gateways/{id}.
type GatewayUpdateRequest struct {
	Gateway *string `json:"gateway,omitempty"`
	Prefix  *string `json:"prefix,omitempty"`
	Limit   *int    `json:"limit,omitempty"`
}

// GatewayEntry is a single gateway row. Limit is *int to distinguish
// "unset on system routes" (nil) from "0" (which would never be valid).
type GatewayEntry struct {
	ID      int    `json:"id,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	Limit   *int   `json:"limit,omitempty"` // null for system routes
	System  bool   `json:"system,omitempty"`
}

// GatewayNumberSummary is one number bound to a gateway, as returned by
// GET /v2.2/gateways/{id}/numbers.
type GatewayNumberSummary struct {
	Number     string  `json:"number"`
	Translated string  `json:"translated"`
	Forward    bool    `json:"forward"`
	ForwardTo  *string `json:"forwardTo"` // nullable
	CNAM       bool    `json:"cnam"`
	Carrier    int     `json:"carrier"` // outbound messaging carrier id; 0 = none
	SMSEnabled bool    `json:"smsEnabled"`
	FaxEnabled bool    `json:"faxEnabled"`
}

// GatewaysListData is the response data for GET /v2.2/gateways.
type GatewaysListData struct {
	Gateways []GatewayEntry `json:"gateways"`
}

// GatewayNumbersData is the response data for GET /v2.2/gateways/{id}/numbers.
type GatewayNumbersData struct {
	Numbers []GatewayNumberSummary `json:"numbers"`
}

// List returns every gateway on the account.
func (s *GatewaysService) List(ctx context.Context) (*GatewaysListData, error) {
	var out GatewaysListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/gateways", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Add creates a new gateway.
func (s *GatewaysService) Add(ctx context.Context, body GatewayAddRequest) (*GatewayEntry, error) {
	var out GatewayEntry
	if err := s.c.t.request(ctx, "POST", "/v2.2/gateways", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches a single gateway by id.
func (s *GatewaysService) Get(ctx context.Context, id int) (*GatewayEntry, error) {
	var out GatewayEntry
	if err := s.c.t.request(ctx, "GET", "/v2.2/gateways/"+strconv.Itoa(id), nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update partial-updates a gateway.
func (s *GatewaysService) Update(ctx context.Context, id int, body GatewayUpdateRequest) (*GatewayEntry, error) {
	var out GatewayEntry
	if err := s.c.t.request(ctx, "PUT", "/v2.2/gateways/"+strconv.Itoa(id), nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove deletes a gateway. Returns nil on 204 No Content.
func (s *GatewaysService) Remove(ctx context.Context, id int) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/gateways/"+strconv.Itoa(id), nil, nil, nil, true)
}

// Numbers returns every number routed through `id`.
func (s *GatewaysService) Numbers(ctx context.Context, id int) (*GatewayNumbersData, error) {
	var out GatewayNumbersData
	if err := s.c.t.request(ctx, "GET", "/v2.2/gateways/"+strconv.Itoa(id)+"/numbers", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

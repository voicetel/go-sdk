package voicetel

import (
	"context"
	"net/url"
	"strings"
)

// NumbersService is the entry point for every operation on a telephone number
// owned by the account.
type NumbersService struct{ c *Client }

// ---------------------------------------------------------------- requests ---

// NumberAddRequest is the body for POST /v2.2/numbers.
type NumberAddRequest struct {
	Number string `json:"number"`
	Route  *int   `json:"route,omitempty"` // gateway route ID; defaults to 4 (DID)
}

// NumberRouteRequest is the body for PUT /v2.2/numbers/{number}/route.
type NumberRouteRequest struct {
	Route int `json:"route"`
}

// NumberCnamRequest is the body for PUT /v2.2/numbers/{number}/cnam.
type NumberCnamRequest struct {
	Enabled bool `json:"enabled"`
}

// NumberLidbRequest is the body for PUT /v2.2/numbers/{number}/lidb.
type NumberLidbRequest struct {
	CNAM                   string  `json:"cnam"` // outbound caller name; max 15 alphanumeric chars
	CustomerOrderReference *string `json:"customerOrderReference,omitempty"`
}

// NumberFaxRequest is the body for PUT /v2.2/numbers/{number}/fax.
type NumberFaxRequest struct {
	Email string `json:"email"`
}

// NumberForwardRequest is the body for PUT /v2.2/numbers/{number}/forward.
type NumberForwardRequest struct {
	Destination string `json:"destination"` // 10-digit destination number
}

// NumberTranslationRequest is the body for PUT /v2.2/numbers/{number}/translation.
type NumberTranslationRequest struct {
	Translation string `json:"translation"` // digits and # only
}

// NumberSmsRequest is the body for PUT /v2.2/numbers/{number}/sms.
type NumberSmsRequest struct {
	Type     string `json:"type"`     // "email", "webhook", or "sip"
	Resource string `json:"resource"` // email/webhook URL/IP per Type
}

// NumberMessagingPatchRequest is the body for PATCH /v2.2/numbers/{number}/messaging.
//
// At least one of RouteIn or RouteOut must be set. RouteIn must be a
// numbers_sms row id owned by the authenticated account, or 0 to detach.
// RouteOut must be 0 (outbound off) or the route the number is enabled on;
// to move a number to another campaign or messaging path use AssignCampaign.
type NumberMessagingPatchRequest struct {
	RouteIn  *int `json:"routeIn,omitempty"`  // numbers_sms row id; 0 to detach
	RouteOut *int `json:"routeOut,omitempty"` // outbound carrier id
}

// NumberCampaignAssignRequest is the body for PUT /v2.2/numbers/{number}/messaging-campaign.
type NumberCampaignAssignRequest struct {
	CampaignID string `json:"campaignId"` // 7-character TCR campaign id, alphanumeric uppercase
}

// NumberMoveRequest is the body for PATCH /v2.2/numbers/{number}.
type NumberMoveRequest struct {
	AccountID int `json:"accountId"` // destination account id
	Route     int `json:"route"`
}

// PortOutPinUpdateRequest is the body for PATCH /v2.2/numbers/{number}/port-out-pin.
type PortOutPinUpdateRequest struct {
	PIN string `json:"pin"` // 4-digit numeric
}

// ------------------------------------------------------- entities & responses ---

// NumberDetail is the per-number routing/feature state returned by
// GET /v2.2/numbers and GET /v2.2/numbers/{number}.
type NumberDetail struct {
	Number     string  `json:"number"`
	Translated string  `json:"translated"`
	Route      int     `json:"route"`
	Gateway    *string `json:"gateway"`
	CNAM       bool    `json:"cnam"`
	Forward    bool    `json:"forward"`
	ForwardTo  *string `json:"forwardTo"`
	Carrier    int     `json:"carrier"`
	SMSEnabled bool    `json:"smsEnabled"`
	FaxEnabled bool    `json:"faxEnabled"`
}

// CampaignBinding is the campaign currently bound to a number, with CSP status.
type CampaignBinding struct {
	ID            string `json:"id"`
	Network       string `json:"network"` // "A" or "B"
	Status        string `json:"status"`  // ACTIVE, EXPIRED, SUSPENDED, ...
	UpstreamCnpID string `json:"upstreamCnpId"`
}

// MessagingEnablementJob is the state of an asynchronous (path C) enablement,
// returned by AssignCampaign (status "pending") and while in progress as
// NumberMessagingState.Pending.
type MessagingEnablementJob struct {
	CampaignID string `json:"campaignId"`
	State      string `json:"state"`     // queued, hosting, assigning, routing, done, or failed
	Message    string `json:"message"`   // latest progress note, or the reason for failure
	CreatedAt  string `json:"createdAt"` // date-time
	UpdatedAt  string `json:"updatedAt"` // date-time
}

// NumberMessagingState is the messaging-routing state for one number.
type NumberMessagingState struct {
	Number    string                  `json:"number"`
	OnAccount *bool                   `json:"onAccount,omitempty"`
	Enabled   bool                    `json:"enabled"`
	Carrier   int                     `json:"carrier"`
	RouteIn   int                     `json:"routeIn"`
	Resource  string                  `json:"resource"`
	Network   *string                 `json:"network"` // "A", "B", "C", or null
	Campaign  *CampaignBinding        `json:"campaign"`
	Pending   *MessagingEnablementJob `json:"pending,omitempty"` // async path-C enablement in progress
}

// NumberAddData is the response data for POST /v2.2/numbers.
type NumberAddData struct {
	Number string `json:"number"`
	Route  int    `json:"route"`
}

// NumberCnamData is the response data for PUT /v2.2/numbers/{number}/cnam.
type NumberCnamData struct {
	Number string `json:"number"`
	CNAM   bool   `json:"cnam"`
}

// NumberFaxData is the response data for GET/PUT /v2.2/numbers/{number}/fax.
type NumberFaxData struct {
	Number string `json:"number"`
	Email  string `json:"email"`
}

// NumberForwardData is the response data for PUT /v2.2/numbers/{number}/forward.
type NumberForwardData struct {
	Number    string  `json:"number"`
	ForwardTo *string `json:"forwardTo"` // 10-digit TN, or null when disabled
}

// NumberLidbData is the response data for PUT /v2.2/numbers/{number}/lidb.
type NumberLidbData struct {
	Number                 string `json:"number"`
	CNAM                   string `json:"cnam"`                   // sanitised caller name (max 15)
	CustomerOrderReference string `json:"customerOrderReference"` // echoed or auto-generated
	CarrierStatus          string `json:"carrierStatus"`          // "Success" or failure detail
}

// NumberMessagingPatchData is the response data for
// PATCH /v2.2/numbers/{number}/messaging.
type NumberMessagingPatchData struct {
	Number  string   `json:"number"`
	Updated []string `json:"updated"` // subset of {"routeIn", "routeOut"}
}

// NumberMoveData is the response data for PATCH /v2.2/numbers/{number}.
type NumberMoveData struct {
	Number    string `json:"number"`
	AccountID int    `json:"accountId"`
	Route     int    `json:"route"`
}

// NumberRouteData is the response data for PUT /v2.2/numbers/{number}/route.
type NumberRouteData struct {
	Number string `json:"number"`
	Route  int    `json:"route"`
}

// NumberSmsData is the response data for GET/PUT /v2.2/numbers/{number}/sms.
type NumberSmsData struct {
	Number   string `json:"number"`
	Type     string `json:"type"` // "email", "webhook", "sip", or "unknown"
	Resource string `json:"resource"`
}

// NumberTranslationData is the response data for PUT /v2.2/numbers/{number}/translation.
type NumberTranslationData struct {
	Number      string `json:"number"`
	Translation string `json:"translation"`
}

// NumberMessagingCampaignAssignData is the response data for
// PUT /v2.2/numbers/{number}/messaging-campaign.
//
// The campaign must be available to the account and ACTIVE; the number is
// enabled on the messaging path the campaign lives on — path B synchronously
// (200, Status "enabled"), path C asynchronously (202, Status "pending":
// poll GetMessaging until Pending.State is done or failed). Idempotent: a
// number already bound to the campaign reports Status "already".
type NumberMessagingCampaignAssignData struct {
	Number                 string                  `json:"number"`
	CampaignID             string                  `json:"campaignId"`
	Status                 string                  `json:"status"`                 // "enabled", "already", or "pending"
	Carrier                int                     `json:"carrier"`                // 19 = path B, 1 = path C
	Network                *string                 `json:"network"`                // "B", "C", or null
	UpstreamCnpID          *string                 `json:"upstreamCnpId"`          // CAMP003 = path B, BANDW = path C
	PreviousNetwork        *string                 `json:"previousNetwork"`        // "A", "B", or null
	PreviousNetworkCleared bool                    `json:"previousNetworkCleared"` // true if a prior binding was disabled
	AlreadyAssigned        bool                    `json:"alreadyAssigned"`        // true when already bound to this campaign
	OrderID                any                     `json:"orderId"`                // network order id (path B); null for path C, no-ops
	Warning                *string                 `json:"warning"`                // non-fatal condition detected, or null
	Message                *string                 `json:"message"`                // note for a pending enablement, or null
	Job                    *MessagingEnablementJob `json:"job"`                    // async path-C enablement, or null
}

// NumberMessagingCampaignUnassignData is the response data for
// DELETE /v2.2/numbers/{number}/messaging-campaign.
type NumberMessagingCampaignUnassignData struct {
	Number        string  `json:"number"`
	CampaignID    string  `json:"campaignId"`
	Network       *string `json:"network"`
	UpstreamCnpID *string `json:"upstreamCnpId"`
	Unassigned    bool    `json:"unassigned"` // always true on 200
}

// CampaignUnassignFailure is one row in NumbersMessagingCampaignUnassignData.Failed.
type CampaignUnassignFailure struct {
	Number string `json:"number"`
	Reason string `json:"reason"`
}

// NumbersMessagingCampaignUnassignData is the response data for
// DELETE /v2.2/numbers/messaging-campaign (bulk unassign).
type NumbersMessagingCampaignUnassignData struct {
	CampaignID        string                    `json:"campaignId"`
	Network           *string                   `json:"network"`
	UpstreamCnpID     *string                   `json:"upstreamCnpId"`
	UnassignedNumbers []string                  `json:"unassignedNumbers"`
	Failed            []CampaignUnassignFailure `json:"failed,omitempty"`
}

// NumbersListData is the response data for GET /v2.2/numbers.
type NumbersListData struct {
	Numbers []NumberDetail `json:"numbers"`
}

// NumbersMessagingListData is the response data for GET /v2.2/numbers/messaging.
type NumbersMessagingListData struct {
	Numbers []NumberMessagingState `json:"numbers"`
}

// PortOutPinUpdateData is the response data for
// PATCH /v2.2/numbers/{number}/port-out-pin.
type PortOutPinUpdateData struct {
	Number     string `json:"number"`     // ten-digit TN whose PIN is reported
	PortOutPin string `json:"portOutPin"` // PIN now on file (just set, or the locked existing value)
	Changed    bool   `json:"changed"`    // true if this call set the PIN; false if one was already locked
}

// ------------------------------------------------------------------ methods ---

// List returns every TN on the account.
func (s *NumbersService) List(ctx context.Context) (*NumbersListData, error) {
	var out NumbersListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Add attaches a TN to the account.
func (s *NumbersService) Add(ctx context.Context, body NumberAddRequest) (*NumberAddData, error) {
	var out NumberAddData
	if err := s.c.t.request(ctx, "POST", "/v2.2/numbers", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches one TN.
func (s *NumbersService) Get(ctx context.Context, number string) (*NumberDetail, error) {
	var out NumberDetail
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/"+url.PathEscape(number), nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove detaches a TN. Returns nil on 204 No Content.
func (s *NumbersService) Remove(ctx context.Context, number string) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/numbers/"+url.PathEscape(number), nil, nil, nil, true)
}

// Move transfers a TN to another account on the same authenticated org.
func (s *NumbersService) Move(ctx context.Context, number string, body NumberMoveRequest) (*NumberMoveData, error) {
	var out NumberMoveData
	if err := s.c.t.request(ctx, "PATCH", "/v2.2/numbers/"+url.PathEscape(number), nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Release returns a TN to the network. Returns nil on 204 No Content.
func (s *NumbersService) Release(ctx context.Context, number string) error {
	return s.c.t.request(ctx, "POST", "/v2.2/numbers/"+url.PathEscape(number)+"/release", nil, nil, nil, true)
}

// SetRoute updates a TN's outbound route.
func (s *NumbersService) SetRoute(ctx context.Context, number string, body NumberRouteRequest) (*NumberRouteData, error) {
	var out NumberRouteData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/route", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetTranslation updates a TN's DNIS translation.
func (s *NumbersService) SetTranslation(ctx context.Context, number string, body NumberTranslationRequest) (*NumberTranslationData, error) {
	var out NumberTranslationData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/translation", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetCNAM toggles inbound CNAM lookup for a TN.
func (s *NumbersService) SetCNAM(ctx context.Context, number string, body NumberCnamRequest) (*NumberCnamData, error) {
	var out NumberCnamData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/cnam", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetLidb updates a TN's outbound caller name (LIDB).
func (s *NumbersService) SetLidb(ctx context.Context, number string, body NumberLidbRequest) (*NumberLidbData, error) {
	var out NumberLidbData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/lidb", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetFax reads fax-to-email routing.
func (s *NumbersService) GetFax(ctx context.Context, number string) (*NumberFaxData, error) {
	var out NumberFaxData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/"+url.PathEscape(number)+"/fax", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetFax enables fax-to-email routing.
func (s *NumbersService) SetFax(ctx context.Context, number string, body NumberFaxRequest) (*NumberFaxData, error) {
	var out NumberFaxData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/fax", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveFax disables fax-to-email. Returns nil on 204 No Content.
func (s *NumbersService) RemoveFax(ctx context.Context, number string) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/numbers/"+url.PathEscape(number)+"/fax", nil, nil, nil, true)
}

// SetForward enables call forwarding.
func (s *NumbersService) SetForward(ctx context.Context, number string, body NumberForwardRequest) (*NumberForwardData, error) {
	var out NumberForwardData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/forward", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveForward disables call forwarding. Returns nil on 204 No Content.
func (s *NumbersService) RemoveForward(ctx context.Context, number string) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/numbers/"+url.PathEscape(number)+"/forward", nil, nil, nil, true)
}

// GetSMS reads SMS routing.
func (s *NumbersService) GetSMS(ctx context.Context, number string) (*NumberSmsData, error) {
	var out NumberSmsData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/"+url.PathEscape(number)+"/sms", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetSMS configures SMS routing.
func (s *NumbersService) SetSMS(ctx context.Context, number string, body NumberSmsRequest) (*NumberSmsData, error) {
	var out NumberSmsData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/sms", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveSMS clears SMS routing. Returns nil on 204 No Content.
func (s *NumbersService) RemoveSMS(ctx context.Context, number string) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/numbers/"+url.PathEscape(number)+"/sms", nil, nil, nil, true)
}

// GetMessaging returns the messaging state for one TN.
func (s *NumbersService) GetMessaging(ctx context.Context, number string) (*NumberMessagingState, error) {
	var out NumberMessagingState
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/"+url.PathEscape(number)+"/messaging", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchMessaging updates inbound/outbound routing for one TN.
func (s *NumbersService) PatchMessaging(ctx context.Context, number string, body NumberMessagingPatchRequest) (*NumberMessagingPatchData, error) {
	var out NumberMessagingPatchData
	if err := s.c.t.request(ctx, "PATCH", "/v2.2/numbers/"+url.PathEscape(number)+"/messaging", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// AssignCampaign binds a 10DLC campaign to a TN. Path-B campaigns are bound
// synchronously (Status "enabled"); path-C campaigns are provisioned
// asynchronously (202, Status "pending" — poll GetMessaging until the job is
// done or failed). Campaigns still on path A are being migrated and are
// refused (409).
func (s *NumbersService) AssignCampaign(ctx context.Context, number string, body NumberCampaignAssignRequest) (*NumberMessagingCampaignAssignData, error) {
	var out NumberMessagingCampaignAssignData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/numbers/"+url.PathEscape(number)+"/messaging-campaign", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnassignCampaign removes the binding to campaignID from a TN.
func (s *NumbersService) UnassignCampaign(ctx context.Context, number, campaignID string) (*NumberMessagingCampaignUnassignData, error) {
	q := url.Values{}
	q.Set("campaignId", campaignID)
	var out NumberMessagingCampaignUnassignData
	if err := s.c.t.request(ctx, "DELETE", "/v2.2/numbers/"+url.PathEscape(number)+"/messaging-campaign", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// BulkUnassignCampaign removes the binding to campaignID from many TNs at
// once (max 1000). The network is resolved via CSP sharing.
func (s *NumbersService) BulkUnassignCampaign(ctx context.Context, numbers []string, campaignID string) (*NumbersMessagingCampaignUnassignData, error) {
	q := url.Values{}
	q.Set("numbers", strings.Join(numbers, ","))
	q.Set("campaignId", campaignID)
	var out NumbersMessagingCampaignUnassignData
	if err := s.c.t.request(ctx, "DELETE", "/v2.2/numbers/messaging-campaign", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetPortOutPin sets the port-out PIN a losing carrier must present to
// release a TN. A PIN is locked for 30 days once set, so the call is
// idempotent: if the TN already has a PIN it is returned unchanged with
// Changed=false; a newly set PIN returns Changed=true.
func (s *NumbersService) SetPortOutPin(ctx context.Context, number string, body PortOutPinUpdateRequest) (*PortOutPinUpdateData, error) {
	var out PortOutPinUpdateData
	if err := s.c.t.request(ctx, "PATCH", "/v2.2/numbers/"+url.PathEscape(number)+"/port-out-pin", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

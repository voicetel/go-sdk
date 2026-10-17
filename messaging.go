package voicetel

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// MessagingService handles SMS/MMS sending and 10DLC brand/campaign registration.
type MessagingService struct{ c *Client }

// MessageSendRequest is the body for POST /v2.2/messages.
//
// MediaURLs being non-nil switches the message type to MMS and unlocks Subject.
type MessageSendRequest struct {
	FromNumber string   `json:"fromNumber"`          // 10-digit TN on the authenticated account
	ToNumber   string   `json:"toNumber"`            // 10-digit destination TN
	Text       string   `json:"text"`                // UTF-8 message body
	Subject    *string  `json:"subject,omitempty"`   // MMS only
	MediaURLs  []string `json:"mediaUrls,omitempty"` // presence makes this an MMS
}

// MessagingBrandCreateRequest is the body for POST /v2.2/messaging/brands.
type MessagingBrandCreateRequest struct {
	MessagingBrandID          string  `json:"messagingBrandId"` // starts with B, alphanumeric
	MessagingBrandName        string  `json:"messagingBrandName"`
	MessagingBrandDescription *string `json:"messagingBrandDescription,omitempty"`
}

// MessagingCampaignCreateRequest is the body for POST /v2.2/messaging/campaigns.
//
// CampaignClassName (from the campaign registry) and CampaignStartDate (today)
// are auto-populated if omitted.
type MessagingCampaignCreateRequest struct {
	MessagingBrandID    string  `json:"messagingBrandId"`
	ExternalCampaignID  string  `json:"externalCampaignId"`
	CampaignDescription string  `json:"campaignDescription"`
	CampaignClassName   *string `json:"campaignClassName,omitempty"`
	CampaignStartDate   *string `json:"campaignStartDate,omitempty"` // ISO 8601
}

// MessageRecordValue is the per-record value inside a MessageRecord. Shape
// depends on the requested message type:
//   - sms/mms: SourceNumber, DestinationNumber, Direction, Rate, Message
//   - dlr:     SourceNumber, DestinationNumber
type MessageRecordValue struct {
	SourceNumber      string `json:"sourceNumber,omitempty"`
	DestinationNumber string `json:"destinationNumber,omitempty"`
	Direction         string `json:"direction,omitempty"` // "in" or "out" (sms/mms only)
	Rate              string `json:"rate,omitempty"`      // billed rate per message (string for precision)
	Number            string `json:"number,omitempty"`    // far-end number (sms/mms only)
	Message           string `json:"message,omitempty"`   // message body (sms/mms only)
}

// MessageRecord is one row in MessageHistoryData.Messages.
type MessageRecord struct {
	ID    string             `json:"id"`
	Key   []any              `json:"key"` // mixed types; see MessageHistoryData docs
	Value MessageRecordValue `json:"value"`
}

// MessageHistoryData is the response data for GET /v2.2/messages.
type MessageHistoryData struct {
	Number   string          `json:"number"`
	Type     string          `json:"type"` // "sms", "mms", or "dlr"
	FromTs   int             `json:"fromTs"`
	ToTs     int             `json:"toTs"`
	Messages []MessageRecord `json:"messages"`
}

// MessageSendData is the response data for POST /v2.2/messages.
type MessageSendData struct {
	ID         string   `json:"id"`   // provider transaction id
	Type       string   `json:"type"` // "sms" or "mms"
	FromNumber string   `json:"fromNumber"`
	ToNumber   string   `json:"toNumber"`
	Parts      int      `json:"parts"` // billed SMS segments; 1 for MMS
	Subject    string   `json:"subject,omitempty"`
	MediaURLs  []string `json:"mediaUrls,omitempty"`
}

// BrandRegistrationResult is the status payload for brand registration.
type BrandRegistrationResult struct {
	StatusCode string `json:"statusCode"` // HTTP status code as string; "200" on success
	Status     string `json:"status"`     // "Success" on success
}

// MessagingBrandCreateData is the response data for POST /v2.2/messaging/brands.
type MessagingBrandCreateData struct {
	Result BrandRegistrationResult `json:"result"`
}

// CampaignRegistrationResult is the status payload for campaign registration.
type CampaignRegistrationResult struct {
	StatusCode string `json:"statusCode"`
	Status     string `json:"status"`
}

// MessagingCampaignCreateData is the response data for POST /v2.2/messaging/campaigns.
type MessagingCampaignCreateData struct {
	Result CampaignRegistrationResult `json:"result"`
}

// CampaignDcaCarrier is one carrier's latest DCA decision in CampaignDca.
type CampaignDcaCarrier struct {
	Carrier   string `json:"carrier"`   // carrier name, e.g. "T-Mobile"
	Status    string `json:"status"`    // ACTIVE, REJECTED, SUSPENDED, UNSUSPENDED or TERMINATED
	Detail    string `json:"detail"`    // the reviewer's note, as given
	UpdatedAt string `json:"updatedAt"` // date-time
}

// CampaignDca is the carrier (DCA) review progress for a campaign.
// nil for CAMPAIGN_INVALID_ID.
type CampaignDca struct {
	ElectionComplete bool                 `json:"electionComplete"` // every carrier has a DCA assigned
	ElectedCarriers  []string             `json:"electedCarriers"`  // carriers with a DCA so far; null when unavailable
	Carriers         []CampaignDcaCarrier `json:"carriers"`         // each carrier decision received, latest per carrier
}

// CampaignStatusItem is a single campaign and its currently-bound numbers.
type CampaignStatusItem struct {
	ID      string       `json:"id"`
	Status  string       `json:"status"` // CAMPAIGN_DCA_COMPLETE, CAMPAIGN_PENDING_DCA, CAMPAIGN_DCA_REJECTED, CAMPAIGN_INVALID_ID, CAMPAIGN_<registry status>, ...
	Numbers []string     `json:"numbers"`
	Dca     *CampaignDca `json:"dca,omitempty"` // carrier (DCA) review progress; null for CAMPAIGN_INVALID_ID
}

// MessagingCampaignStatusData is the response data for GET /v2.2/messaging/campaigns.
type MessagingCampaignStatusData struct {
	Campaigns []CampaignStatusItem `json:"campaigns"`
}

// HistoryOptions are the query filters for History. Number is required by
// the API; the rest are optional.
type HistoryOptions struct {
	Number string // 10-digit TN on the account whose history to fetch (required)
	From   int    // Unix timestamp range start (newer); 0 = server default (now)
	To     int    // Unix timestamp range end (older); 0 = server default (24h ago)
	Type   string // "sms" (default), "mms", or "dlr"
}

// History fetches message history for one number, newest first.
func (s *MessagingService) History(ctx context.Context, opts HistoryOptions) (*MessageHistoryData, error) {
	q := url.Values{}
	q.Set("number", opts.Number)
	if opts.From != 0 {
		q.Set("from", strconv.Itoa(opts.From))
	}
	if opts.To != 0 {
		q.Set("to", strconv.Itoa(opts.To))
	}
	if opts.Type != "" {
		q.Set("type", opts.Type)
	}
	var out MessageHistoryData
	if err := s.c.t.request(ctx, "GET", "/v2.2/messages", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Send sends an SMS or MMS.
func (s *MessagingService) Send(ctx context.Context, body MessageSendRequest) (*MessageSendData, error) {
	var out MessageSendData
	if err := s.c.t.request(ctx, "POST", "/v2.2/messages", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateBrand registers a 10DLC brand with the campaign registry.
func (s *MessagingService) CreateBrand(ctx context.Context, body MessagingBrandCreateRequest) (*MessagingBrandCreateData, error) {
	var out MessagingBrandCreateData
	if err := s.c.t.request(ctx, "POST", "/v2.2/messaging/brands", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// CampaignStatus returns the current status of the given 10DLC campaigns.
// ids holds the alphanumeric campaign ids (max 100); at least one is required.
func (s *MessagingService) CampaignStatus(ctx context.Context, ids []string) (*MessagingCampaignStatusData, error) {
	q := url.Values{}
	q.Set("ids", strings.Join(ids, ","))
	var out MessagingCampaignStatusData
	if err := s.c.t.request(ctx, "GET", "/v2.2/messaging/campaigns", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCampaign registers a 10DLC campaign with the carrier.
func (s *MessagingService) CreateCampaign(ctx context.Context, body MessagingCampaignCreateRequest) (*MessagingCampaignCreateData, error) {
	var out MessagingCampaignCreateData
	if err := s.c.t.request(ctx, "POST", "/v2.2/messaging/campaigns", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// NumbersState returns the messaging state for many numbers at once.
// At least one 10-digit TN is required (max 1000).
func (s *MessagingService) NumbersState(ctx context.Context, numbers []string) (*NumbersMessagingListData, error) {
	q := url.Values{}
	q.Set("numbers", strings.Join(numbers, ","))
	var out NumbersMessagingListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/messaging", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Health returns the account-wide messaging health overview — the same view as
// the portal Messaging tab: summary counts (numbers, messaging on, healthy,
// needing attention, in progress), campaigns with carrier-review progress,
// numbers needing attention (each with a plain-language title, detail and the
// actions that fix it), 14 days of delivery counts, and the alert-email setting.
// Updated hourly.
//
// The API spec defines no response schema for this endpoint, so the decoded JSON
// object is returned as-is rather than mapped onto a typed struct.
func (s *MessagingService) Health(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.t.request(ctx, "GET", "/v2.2/messaging/health", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

// HealthAlerts returns the alert-email setting: whether new messaging problems
// are emailed to the account.
//
// The spec defines no response schema, so the decoded JSON object is returned as-is.
func (s *MessagingService) HealthAlerts(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.t.request(ctx, "GET", "/v2.2/messaging/health/alerts", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

// SetHealthAlerts turns alert emails on or off. When enabled, the account is
// emailed as new messaging problems are found (at most one digest about every
// 20 hours).
//
// The spec defines no response schema, so the decoded JSON object is returned as-is.
func (s *MessagingService) SetHealthAlerts(ctx context.Context, enabled bool) (map[string]any, error) {
	var out map[string]any
	body := map[string]any{"enabled": enabled}
	if err := s.c.t.request(ctx, "PUT", "/v2.2/messaging/health/alerts", nil, body, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

// HealthCheck performs a live health check of one number: in service, messaging
// enabled, campaign approved, assigned to the campaign by the messaging provider,
// delivery failures in the last 24 hours, and enablement in progress. Limited to
// 30 checks per hour per account.
//
// The spec defines no response schema, so the decoded JSON object is returned as-is.
func (s *MessagingService) HealthCheck(ctx context.Context, number string) (map[string]any, error) {
	var out map[string]any
	body := map[string]any{"number": number}
	if err := s.c.t.request(ctx, "POST", "/v2.2/messaging/health/check", nil, body, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

// HealthSamples returns up to 3 recent outbound messages from the number that
// carriers blocked as spam (time, masked recipient, text, reason), to help review
// content.
//
// The spec defines no response schema, so the decoded JSON object is returned as-is.
func (s *MessagingService) HealthSamples(ctx context.Context, number string) (map[string]any, error) {
	q := url.Values{}
	q.Set("number", number)
	var out map[string]any
	if err := s.c.t.request(ctx, "GET", "/v2.2/messaging/health/samples", q, nil, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

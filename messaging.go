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
// CampaignClassName and CampaignStartDate are auto-populated if omitted.
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
	Number            int    `json:"number,omitempty"`    // far-end number (sms/mms only)
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

// CampaignStatusItem is a single campaign and its currently-bound numbers.
type CampaignStatusItem struct {
	ID      string   `json:"id"`
	Status  string   `json:"status"` // CSP status: ACTIVE, CAMPAIGN_DCA_COMPLETE, etc.
	Numbers []string `json:"numbers"`
}

// MessagingCampaignStatusData is the response data for GET /v2.2/messaging/campaigns.
type MessagingCampaignStatusData struct {
	Campaigns []CampaignStatusItem `json:"campaigns"`
}

// HistoryOptions are the optional query filters for History.
type HistoryOptions struct {
	Number string // 10-digit TN whose history to fetch
	Start  int    // Unix timestamp range start
	End    int    // Unix timestamp range end (older than Start)
	Type   string // "sms", "mms", or "dlr"
}

// History fetches message history. Pass HistoryOptions{} for the default set
// or populate any combination of fields.
func (s *MessagingService) History(ctx context.Context, opts HistoryOptions) (*MessageHistoryData, error) {
	q := url.Values{}
	if opts.Number != "" {
		q.Set("number", opts.Number)
	}
	if opts.Start != 0 {
		q.Set("start", strconv.Itoa(opts.Start))
	}
	if opts.End != 0 {
		q.Set("end", strconv.Itoa(opts.End))
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

// CampaignStatus returns the current 10DLC campaign statuses.
func (s *MessagingService) CampaignStatus(ctx context.Context) (*MessagingCampaignStatusData, error) {
	var out MessagingCampaignStatusData
	if err := s.c.t.request(ctx, "GET", "/v2.2/messaging/campaigns", nil, nil, &out, true); err != nil {
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
// Pass an empty slice for "all numbers on the account".
func (s *MessagingService) NumbersState(ctx context.Context, numbers []string) (*NumbersMessagingListData, error) {
	q := url.Values{}
	if len(numbers) > 0 {
		q.Set("numbers", strings.Join(numbers, ","))
	}
	var out NumbersMessagingListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/messaging", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

package voicetel

import (
	"context"
	"net/url"
	"strconv"
)

// AccountService groups every operation under the Account tag.
//
// Note: cdr, recurring_charges, payments, registration, and Client.Login share
// a 6 req/hour/IP rate limit. Bursting will trigger 429s.
type AccountService struct{ c *Client }

// AccountRates carries the per-service rates exposed on an account.
// Read-only for non-administrators.
type AccountRates struct {
	CNAM    float64 `json:"cnam,omitempty"`
	IntlMax float64 `json:"intlMax,omitempty"`
	Nibble  float64 `json:"nibble,omitempty"`
	LRN     float64 `json:"lrn,omitempty"`
	Fax     float64 `json:"fax,omitempty"`
	TfAdj   float64 `json:"tfAdj,omitempty"`
	DID     float64 `json:"did,omitempty"`
	MMS     float64 `json:"mms,omitempty"`
	SMS     float64 `json:"sms,omitempty"`
}

// AccountServices holds per-service feature flags. true = enabled on this account.
type AccountServices struct {
	E911        bool `json:"e911,omitempty"`
	CNAM        bool `json:"cnam,omitempty"`
	BypassMedia bool `json:"bypassMedia,omitempty"`
	Intl        bool `json:"intl,omitempty"`
	RCID        bool `json:"rcid,omitempty"`
	MMS         bool `json:"mms,omitempty"`
	Dialer      bool `json:"dialer,omitempty"`
	SMS         bool `json:"sms,omitempty"`
}

// AccountData is the profile returned by GET /v2.2/account.
type AccountData struct {
	Username        string           `json:"username,omitempty"`
	Name            string           `json:"name,omitempty"`
	Email           string           `json:"email,omitempty"`
	Enabled         bool             `json:"enabled,omitempty"`
	Created         string           `json:"created,omitempty"`
	Cash            float64          `json:"cash,omitempty"`
	CallerID        string           `json:"callerId,omitempty"`
	Timezone        string           `json:"timezone,omitempty"`
	AuthType        int              `json:"authType,omitempty"`
	CCS             int              `json:"ccs,omitempty"`
	Notify          bool             `json:"notify,omitempty"`
	NotifyThreshold int              `json:"notifyThreshold,omitempty"`
	Rates           *AccountRates    `json:"rates,omitempty"`
	Services        *AccountServices `json:"services,omitempty"`
}

// CreditEntry is a single credit row in AccountCreditsData.
type CreditEntry struct {
	Date   string  `json:"date"`
	Paid   bool    `json:"paid"`
	Amount float64 `json:"amount"`
}

// PaymentEntry is a single payment row in AccountPaymentsData.
//
// Status is one of "Completed", "Pending", "Reversed", "Refunded", "Failed",
// "Denied", "Canceled_Reversal".
type PaymentEntry struct {
	TransactionID string  `json:"transactionId,omitempty"`
	Date          string  `json:"date"`
	PayerEmail    string  `json:"payerEmail,omitempty"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
}

// CdrEntryValue is the per-call billing summary inside a CDR row.
//
// All numeric-looking fields (Dur, Ba, Nr) are intentionally strings to preserve
// exact precision on the wire — parse with math/big or strconv if you need math.
type CdrEntryValue struct {
	Dur string `json:"dur,omitempty"` // billed call duration in seconds
	Dst string `json:"dst,omitempty"` // destination 10-digit TN
	Ba  string `json:"ba,omitempty"`  // billed amount, USD
	Nr  string `json:"nr,omitempty"`  // nibble rate, USD/min
	Cn  string `json:"cn,omitempty"`  // URL-encoded display name (CNAM at call time)
	IP  string `json:"ip,omitempty"`  // IPv4 of the leg
	Cid string `json:"cid,omitempty"` // caller ID 10-digit TN
}

// CdrEntry is one row in AccountCdrData.Cdr.
type CdrEntry struct {
	ID    string        `json:"id"`
	Key   []string      `json:"key"` // [accountUsername, startEpochUnixSeconds]
	Value CdrEntryValue `json:"value"`
}

// AccountCdrData is the response data for GET /v2.2/account/cdr.
type AccountCdrData struct {
	Cdr   []CdrEntry `json:"cdr"`
	Start int        `json:"start"` // echo of the `start` query param
	End   int        `json:"end"`   // echo of the `end` query param
}

// AccountCreditsData is the response data for GET /v2.2/account/credits.
type AccountCreditsData struct {
	Credits []CreditEntry `json:"credits"`
}

// AccountPaymentsData is the response data for GET /v2.2/account/payments.
type AccountPaymentsData struct {
	Payments []PaymentEntry `json:"payments"`
}

// MrcCharge is a single monthly-recurring charge row inside AccountMrcData.
type MrcCharge struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
}

// AccountMrcData is the response data for GET /v2.2/account/recurring-charges.
type AccountMrcData struct {
	Charges []MrcCharge `json:"charges"`
	Total   float64     `json:"total"`
}

// AccountRegistrationData is the response data for GET /v2.2/account/registration.
type AccountRegistrationData struct {
	Agent   string `json:"agent,omitempty"`
	URI     string `json:"uri,omitempty"`
	Expires int    `json:"expires,omitempty"`
}

// AccountAddRequest is the body for POST /v2.2/account (admin-only sub-account creation).
type AccountAddRequest struct {
	Username      int    `json:"username"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	MasterAccount *int   `json:"masterAccount,omitempty"`
}

// AccountAddData is the response data for POST /v2.2/account.
type AccountAddData struct {
	Username      string `json:"username,omitempty"`
	Name          string `json:"name,omitempty"`
	Email         string `json:"email,omitempty"`
	MasterAccount string `json:"masterAccount,omitempty"`
	Password      string `json:"password,omitempty"` // auto-generated initial password
}

// AccountPutRequest is the body for PUT /v2.2/account.
//
// Pointer fields are optional; nil means "leave unchanged".
type AccountPutRequest struct {
	Notify          *bool   `json:"notify,omitempty"`
	NotifyThreshold *int    `json:"notifyThreshold,omitempty"`
	Timezone        *string `json:"timezone,omitempty"`
	CallerID        *string `json:"callerId,omitempty"`
	E911            *bool   `json:"e911,omitempty"` // admin only
	Intl            *bool   `json:"intl,omitempty"` // admin only
	SMS             *bool   `json:"sms,omitempty"`  // admin only
	MMS             *bool   `json:"mms,omitempty"`  // admin only
	CCS             *int    `json:"ccs,omitempty"`  // admin only
}

// AccountPutData is the response data for PUT /v2.2/account.
type AccountPutData struct {
	Updated []string `json:"updated"`
}

// AccountSignupRequest is the body for POST /v2.2/accounts (public sign-up).
type AccountSignupRequest struct {
	Name  string  `json:"name"`
	Email string  `json:"email"`
	Promo *string `json:"promo,omitempty"`
}

// AccountSignupData is the response data for POST /v2.2/accounts.
type AccountSignupData struct {
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

// AccountRecoverRequest is the body for POST /v2.2/account/recovery (no auth required).
type AccountRecoverRequest struct {
	Email string `json:"email"`
}

// AccountRecoverData is the response data for POST /v2.2/account/recovery.
type AccountRecoverData struct {
	Message string `json:"message,omitempty"`
}

// AccountApiKeyData is the response data for POST /v2.2/account/api-key.
type AccountApiKeyData struct {
	APIKey string `json:"apikey"`
}

// ---------------------------------------------------------------- methods ---

// Get returns the authenticated account's profile.
func (s *AccountService) Get(ctx context.Context) (*AccountData, error) {
	var out AccountData
	err := s.c.t.request(ctx, "GET", "/v2.2/account", nil, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update partial-updates account settings. Only fields you set on body are sent.
func (s *AccountService) Update(ctx context.Context, body AccountPutRequest) (*AccountPutData, error) {
	var out AccountPutData
	err := s.c.t.request(ctx, "PUT", "/v2.2/account", nil, body, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Add creates a sub-account. Admin-only.
func (s *AccountService) Add(ctx context.Context, body AccountAddRequest) (*AccountAddData, error) {
	var out AccountAddData
	err := s.c.t.request(ctx, "POST", "/v2.2/account", nil, body, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Signup is the public sign-up flow: POST /v2.2/accounts.
func (s *AccountService) Signup(ctx context.Context, body AccountSignupRequest) (*AccountSignupData, error) {
	var out AccountSignupData
	err := s.c.t.request(ctx, "POST", "/v2.2/accounts", nil, body, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CDR fetches call detail records in the [Start, End] Unix-seconds range.
// Rate-limited: 6 req/hour/IP shared with mrc/payments/registration/api-key.
func (s *AccountService) CDR(ctx context.Context, start, end int) (*AccountCdrData, error) {
	q := url.Values{}
	if start != 0 {
		q.Set("start", strconv.Itoa(start))
	}
	if end != 0 {
		q.Set("end", strconv.Itoa(end))
	}
	var out AccountCdrData
	err := s.c.t.request(ctx, "GET", "/v2.2/account/cdr", q, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Credits returns the full credit history, newest first.
func (s *AccountService) Credits(ctx context.Context) (*AccountCreditsData, error) {
	var out AccountCreditsData
	err := s.c.t.request(ctx, "GET", "/v2.2/account/credits", nil, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RecurringCharges returns active monthly-recurring charges. Rate-limited.
func (s *AccountService) RecurringCharges(ctx context.Context) (*AccountMrcData, error) {
	var out AccountMrcData
	err := s.c.t.request(ctx, "GET", "/v2.2/account/recurring-charges", nil, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Payments returns the full payment history, newest first. Rate-limited.
func (s *AccountService) Payments(ctx context.Context) (*AccountPaymentsData, error) {
	var out AccountPaymentsData
	err := s.c.t.request(ctx, "GET", "/v2.2/account/payments", nil, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Registration returns the current SIP registration. Rate-limited.
func (s *AccountService) Registration(ctx context.Context) (*AccountRegistrationData, error) {
	var out AccountRegistrationData
	err := s.c.t.request(ctx, "GET", "/v2.2/account/registration", nil, nil, &out, true)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Recover starts the password recovery flow (no auth required).
func (s *AccountService) Recover(ctx context.Context, body AccountRecoverRequest) (*AccountRecoverData, error) {
	var out AccountRecoverData
	err := s.c.t.request(ctx, "POST", "/v2.2/account/recovery", nil, body, &out, false)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

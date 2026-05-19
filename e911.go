package voicetel

import "context"

// E911Service manages e911 records and address validation.
//
// Note the asymmetric `dn` formats: requests take a 10-digit TN; responses
// return the 11-digit E.164 US form (country code 1 prepended).
type E911Service struct{ c *Client }

// E911AddressRequest is the body for POST /v2.2/e911/validations.
type E911AddressRequest struct {
	Address1 string  `json:"address1"`
	Address2 *string `json:"address2,omitempty"`
	City     string  `json:"city"`
	State    string  `json:"state"` // two-letter US state code
	Zip      string  `json:"zip"`
}

// E911CreateRequest is the body for POST /v2.2/e911 (validate + provision in one call).
type E911CreateRequest struct {
	DN         string  `json:"dn"` // 10-digit TN owned by the authenticated account
	Callername string  `json:"callername"`
	Address1   string  `json:"address1"`
	Address2   *string `json:"address2,omitempty"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	Zip        string  `json:"zip"`
}

// E911ProvisionByIDRequest is the body for PUT /v2.2/e911/{dn}.
type E911ProvisionByIDRequest struct {
	Callername string `json:"callername"`
	AddressID  int    `json:"addressid"` // from POST /v2.2/e911/validations
}

// E911Entry is an e911 record bound to a TN.
type E911Entry struct {
	DN         string `json:"dn"` // 11-digit E.164 US form (leading 1)
	Callername string `json:"callername"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        string `json:"zip"`
}

// E911ValidatedAddress is the result from POST /v2.2/e911/validations.
type E911ValidatedAddress struct {
	AddressID int    `json:"addressid"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2,omitempty"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zip       string `json:"zip"`
}

// E911AllData is the response data for GET /v2.2/e911.
type E911AllData struct {
	Records []E911Entry `json:"records"`
}

// E911RecordData is the response data for GET /v2.2/e911/{dn}, POST /v2.2/e911, PUT /v2.2/e911/{dn}.
type E911RecordData struct {
	Record E911Entry `json:"record"`
}

// E911ValidateData is the response data for POST /v2.2/e911/validations.
type E911ValidateData struct {
	Address E911ValidatedAddress `json:"address"`
}

// List returns every e911 record on the account.
func (s *E911Service) List(ctx context.Context) (*E911AllData, error) {
	var out E911AllData
	if err := s.c.t.request(ctx, "GET", "/v2.2/e911", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create validates and provisions an e911 record in one call.
func (s *E911Service) Create(ctx context.Context, body E911CreateRequest) (*E911RecordData, error) {
	var out E911RecordData
	if err := s.c.t.request(ctx, "POST", "/v2.2/e911", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Validate validates an address, returning an AddressID for use with Provision.
func (s *E911Service) Validate(ctx context.Context, body E911AddressRequest) (*E911ValidateData, error) {
	var out E911ValidateData
	if err := s.c.t.request(ctx, "POST", "/v2.2/e911/validations", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches the e911 record for `dn`.
func (s *E911Service) Get(ctx context.Context, dn string) (*E911RecordData, error) {
	var out E911RecordData
	if err := s.c.t.request(ctx, "GET", "/v2.2/e911/"+dn, nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Provision uses a previously-validated AddressID to provision e911 for `dn`.
func (s *E911Service) Provision(ctx context.Context, dn string, body E911ProvisionByIDRequest) (*E911RecordData, error) {
	var out E911RecordData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/e911/"+dn, nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove deletes the e911 record for `dn`. Returns nil on 204 No Content.
func (s *E911Service) Remove(ctx context.Context, dn string) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/e911/"+dn, nil, nil, nil, true)
}

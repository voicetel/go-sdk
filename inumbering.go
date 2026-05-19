package voicetel

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// INumberingService covers inventory searches, orders, and port-ins.
type INumberingService struct{ c *Client }

// OrderCreateRequest is the body for POST /v2.2/orders.
//
// Each entry in Numbers may be a plain string TN (use OrderNumber{Value: ...})
// or a {number, route} object (use OrderNumber{Spec: &OrderNumberSpec{...}}).
//
// Total entries: 1..100.
type OrderCreateRequest struct {
	Numbers []OrderNumber `json:"numbers"`
}

// OrderNumber is a single entry in OrderCreateRequest.Numbers. Exactly one of
// the two fields should be set; MarshalJSON enforces this.
type OrderNumber struct {
	Value string           // plain TN (10-digit string)
	Spec  *OrderNumberSpec // TN with optional gateway route override
}

// OrderNumberSpec is the {number, route} object variant.
type OrderNumberSpec struct {
	Number string `json:"number"`
	Route  *int   `json:"route,omitempty"`
}

// MarshalJSON serializes OrderNumber as either a JSON string or object.
func (n OrderNumber) MarshalJSON() ([]byte, error) {
	if n.Spec != nil {
		return json.Marshal(n.Spec)
	}
	return json.Marshal(n.Value)
}

// PortFeatureLidb is the LIDB feature for a port-in TN.
type PortFeatureLidb struct {
	Name string `json:"name"` // outbound caller name; max 15 chars
}

// PortFeatureRouting is the routing feature for a port-in TN.
type PortFeatureRouting struct {
	GatewayID int `json:"gatewayId"`
}

// PortFeatureSms is the SMS feature for a port-in TN.
type PortFeatureSms struct {
	CampaignID *string `json:"campaignId,omitempty"`
}

// PortFeature is per-TN feature configuration applied after the port completes.
type PortFeature struct {
	Number  string              `json:"number"`
	Routing *PortFeatureRouting `json:"routing,omitempty"`
	Lidb    *PortFeatureLidb    `json:"lidb,omitempty"`
	SMS     *PortFeatureSms     `json:"sms,omitempty"`
}

// PortSubmitRequest is the body for POST /v2.2/ports.
//
// StreetPrefix/StreetSuffix are one of "N","NE","E","SE","S","SW","W","NW".
type PortSubmitRequest struct {
	DID             []string      `json:"did"`             // 10-digit TNs (toll-free not supported)
	Name            string        `json:"name"`            // exactly as on losing carrier bill
	NameType        string        `json:"nameType"`        // "business" or "residential"
	LCBtn           string        `json:"lcBtn"`           // billing TN on losing carrier bill
	LCAccountNumber string        `json:"lcAccountNumber"` // account number on bill
	StreetNumber    string        `json:"streetNumber"`
	Street          string        `json:"street"`
	StreetType      string        `json:"streetType"` // USPS abbreviation: ST, AVE, BLVD, ...
	City            string        `json:"city"`
	State           string        `json:"state"` // two-letter
	Zip             string        `json:"zip"`
	Country         string        `json:"country"`
	AuthPerson      string        `json:"authPerson"` // full name authorised to sign LOA
	StreetPrefix    *string       `json:"streetPrefix,omitempty"`
	StreetSuffix    *string       `json:"streetSuffix,omitempty"`
	Floor           *string       `json:"floor,omitempty"`
	Room            *string       `json:"room,omitempty"`
	Building        *string       `json:"building,omitempty"`
	UnitValue       *string       `json:"unitValue,omitempty"`      // unit designator like "APT 3" or "STE 200"
	DesiredDueDate  *string       `json:"desiredDueDate,omitempty"` // ISO 8601; blank = standard SLA
	PIN             *string       `json:"pin,omitempty"`            // port-out PIN from losing carrier
	Features        []PortFeature `json:"features,omitempty"`
}

// InventoryItem is one TN available for assignment.
type InventoryItem struct {
	Number     string `json:"number"`
	RateCenter string `json:"rateCenter"`
	City       string `json:"city"`
	Province   string `json:"province"` // two-letter state/province
	LATA       string `json:"lata"`
}

// InventoryCoverageItem is one aggregated availability bucket. Which fields are
// populated depends on the countBy dimension on the query.
type InventoryCoverageItem struct {
	Count    int    `json:"count"`
	NPA      string `json:"npa,omitempty"`
	NXX      string `json:"nxx,omitempty"`
	Block    string `json:"block,omitempty"`
	City     string `json:"city,omitempty"`
	RcAbbre  string `json:"rcAbbre,omitempty"`
	LATA     string `json:"lata,omitempty"`
	LocState string `json:"locState,omitempty"`
}

// PortSummary is one row in the port-status list.
type PortSummary struct {
	Status     string `json:"status"`
	ID         string `json:"id,omitempty"`
	PID        string `json:"pid,omitempty"`
	FOC        string `json:"foc,omitempty"` // Firm Order Commitment (YYYYMMDD)
	CreatedAt  string `json:"createdAt,omitempty"`
	Message    string `json:"message,omitempty"`
	SupportURL string `json:"supportUrl,omitempty"`
}

// PortDetail is the full record for a single port-in.
type PortDetail struct {
	Status    string   `json:"status"`
	ID        string   `json:"id,omitempty"`
	PID       string   `json:"pid,omitempty"`
	Name      string   `json:"name,omitempty"`
	Email     string   `json:"email,omitempty"`
	FOC       string   `json:"foc,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
	Numbers   []string `json:"numbers,omitempty"`
	Message   string   `json:"message,omitempty"`
}

// InventorySearchData is the response data for GET /v2.2/inventory.
type InventorySearchData struct {
	Numbers []InventoryItem `json:"numbers"`
}

// InventoryCoverageData is the response data for GET /v2.2/inventory/coverage.
type InventoryCoverageData struct {
	Coverage []InventoryCoverageItem `json:"coverage"`
}

// OrderFailedEntry is one row in OrderCreateData.Failed.
type OrderFailedEntry struct {
	Number string `json:"number"`
	Reason string `json:"reason"`
}

// OrderCreateData is the response data for POST /v2.2/orders.
type OrderCreateData struct {
	OrderID        string             `json:"orderId"`
	AmountCharged  float64            `json:"amountCharged"`
	NumbersOrdered []string           `json:"numbersOrdered"`
	Failed         []OrderFailedEntry `json:"failed,omitempty"`
}

// PortListData is the response data for GET /v2.2/ports.
type PortListData struct {
	Ports []PortSummary `json:"ports"`
}

// PortDetailData is the response data for GET /v2.2/ports/{id}.
type PortDetailData struct {
	Port PortDetail `json:"port"`
}

// PortSubmitData is the response data for POST /v2.2/ports.
type PortSubmitData struct {
	PID     string `json:"pid"`    // 5-character port order ID
	Ticket  int    `json:"ticket"` // support ticket ID
	Message string `json:"message"`
	LoaURL  string `json:"loaUrl"`  // LOA download URL
	PortURL string `json:"portUrl"` // port status URL
}

// PortAvailabilityData is the response data for GET /v2.2/ports/availability/{number}.
type PortAvailabilityData struct {
	Number        string  `json:"number"`
	Portable      bool    `json:"portable"`
	LosingCarrier *string `json:"losingCarrier"` // SPID/OCN; nullable
	Reason        *string `json:"reason"`        // nullable when portable
}

// InventoryQuery are the query filters for SearchInventory.
type InventoryQuery struct {
	NPA        int
	NXX        int
	State      string
	RateCenter string
	Contains   string
	EndsWith   string
	Limit      int
}

// CoverageQuery are the query filters for Coverage.
type CoverageQuery struct {
	State      string
	RateCenter string
}

// ------------------------------------------------------------------ methods ---

// SearchInventory searches available TNs by NPA/NXX/state/rate-center/etc.
func (s *INumberingService) SearchInventory(ctx context.Context, q InventoryQuery) (*InventorySearchData, error) {
	v := url.Values{}
	if q.NPA != 0 {
		v.Set("npa", strconv.Itoa(q.NPA))
	}
	if q.NXX != 0 {
		v.Set("nxx", strconv.Itoa(q.NXX))
	}
	if q.State != "" {
		v.Set("state", q.State)
	}
	if q.RateCenter != "" {
		v.Set("ratecenter", q.RateCenter)
	}
	if q.Contains != "" {
		v.Set("contains", q.Contains)
	}
	if q.EndsWith != "" {
		v.Set("endswith", q.EndsWith)
	}
	if q.Limit != 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	var out InventorySearchData
	if err := s.c.t.request(ctx, "GET", "/v2.2/inventory", v, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Coverage returns aggregated availability buckets.
func (s *INumberingService) Coverage(ctx context.Context, q CoverageQuery) (*InventoryCoverageData, error) {
	v := url.Values{}
	if q.State != "" {
		v.Set("state", q.State)
	}
	if q.RateCenter != "" {
		v.Set("ratecenter", q.RateCenter)
	}
	var out InventoryCoverageData
	if err := s.c.t.request(ctx, "GET", "/v2.2/inventory/coverage", v, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Order purchases new TNs.
func (s *INumberingService) Order(ctx context.Context, body OrderCreateRequest) (*OrderCreateData, error) {
	var out OrderCreateData
	if err := s.c.t.request(ctx, "POST", "/v2.2/orders", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Ports lists every port-in record on the account.
func (s *INumberingService) Ports(ctx context.Context) (*PortListData, error) {
	var out PortListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/ports", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Port fetches detail for one port-in by id.
func (s *INumberingService) Port(ctx context.Context, id int) (*PortDetailData, error) {
	var out PortDetailData
	if err := s.c.t.request(ctx, "GET", "/v2.2/ports/"+strconv.Itoa(id), nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// SubmitPort submits a port-in order.
func (s *INumberingService) SubmitPort(ctx context.Context, body PortSubmitRequest) (*PortSubmitData, error) {
	var out PortSubmitData
	if err := s.c.t.request(ctx, "POST", "/v2.2/ports", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// PortAvailability checks whether a given TN can be ported in.
func (s *INumberingService) PortAvailability(ctx context.Context, number string) (*PortAvailabilityData, error) {
	var out PortAvailabilityData
	if err := s.c.t.request(ctx, "GET", "/v2.2/ports/availability/"+number, nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

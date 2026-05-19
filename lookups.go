package voicetel

import "context"

// LookupsService provides CNAM and LRN dips.
//
// Each call costs money; rate them per call rather than fanning out blindly.
type LookupsService struct{ c *Client }

// CnamData is the response data for GET /v2.2/cnam/{number}.
type CnamData struct {
	CNAM   string `json:"cnam,omitempty"`
	Number string `json:"number"`
}

// LrnData is the LRN dip result.
//
// Returned both as the top-level data on GET /v2.2/cnam/{number} (when no ANI
// is supplied) and nested inside LrnLookupData when the /lrn/{n}/{ani} form
// is used.
type LrnData struct {
	LRN          string `json:"lrn,omitempty"`
	State        string `json:"state,omitempty"`
	City         string `json:"city,omitempty"`
	RC           string `json:"rc,omitempty"` // rate center
	LATA         string `json:"lata,omitempty"`
	OCN          string `json:"ocn,omitempty"`
	LEC          string `json:"lec,omitempty"`
	LECType      string `json:"lecType,omitempty"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
	Local        string `json:"local,omitempty"` // Y/N — local to the ANI's rate center
}

// LrnLookupData is the response data for GET /v2.2/lrn/{number}/{ani}.
type LrnLookupData struct {
	ANI         string  `json:"ani"`
	Destination string  `json:"destination"`
	LRN         LrnData `json:"lrn"`
}

// CNAM performs a CNAM dip on `number` (10-digit TN).
func (s *LookupsService) CNAM(ctx context.Context, number string) (*CnamData, error) {
	var out CnamData
	if err := s.c.t.request(ctx, "GET", "/v2.2/cnam/"+number, nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// LRN performs an LRN dip. `ani` is the presented ANI (10-digit TN) used
// only for billing/auth — it is not echoed back.
func (s *LookupsService) LRN(ctx context.Context, number, ani string) (*LrnLookupData, error) {
	var out LrnLookupData
	if err := s.c.t.request(ctx, "GET", "/v2.2/lrn/"+number+"/"+ani, nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

package voicetel

// CidrEntry is a single row in the IP allowlist.
//
// Mask must be /8, /16, /24, or /32 and must describe a routable public address.
type CidrEntry struct {
	CIDR string `json:"cidr"`
}

// envelope wraps every successful API response: {"status": "success", "data": <T>}.
// The transport's decode() peels this off before deserializing into the response type,
// so callers never see it.

// String is a convenience helper for creating *string values inline.
// Useful when populating optional request fields:
//
//	c.Numbers.SetForward(ctx, "2015551234", voicetel.NumberForwardRequest{
//	    Destination: 2125551234,
//	})
//
// (most fields take plain values; the helper is here for fields like SetTimezone
// where the value is *string).
func String(s string) *string { return &s }

// Int is the *int counterpart to String.
func Int(i int) *int { return &i }

// Bool is the *bool counterpart to String.
func Bool(b bool) *bool { return &b }

// Float64 is the *float64 counterpart to String.
func Float64(f float64) *float64 { return &f }

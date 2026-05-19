package voicetel

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// Client is the entry point for the VoiceTel API. Construct one with NewClient
// and reach the API through its resource fields — for example
// client.Numbers.List(ctx).
//
// Client is safe for concurrent use by multiple goroutines as long as the
// configured *http.Client is.
type Client struct {
	t *transport

	Account        *AccountService
	ACL            *ACLService
	Authentication *AuthenticationService
	E911           *E911Service
	Gateways       *GatewaysService
	INumbering     *INumberingService
	Lookups        *LookupsService
	Messaging      *MessagingService
	Numbers        *NumbersService
	Support        *SupportService
}

// Option mutates a Client during construction. Apply with NewClient(opts...).
type Option func(*clientConfig)

type clientConfig struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	userAgent  string
	maxRetries int
}

// WithBaseURL overrides the API endpoint (defaults to DefaultBaseURL).
// Useful for pointing at a sandbox or for tests.
func WithBaseURL(u string) Option { return func(c *clientConfig) { c.baseURL = u } }

// WithAPIKey installs a bearer token. If omitted, you must call Client.Login.
func WithAPIKey(k string) Option { return func(c *clientConfig) { c.apiKey = k } }

// WithHTTPClient swaps in a custom *http.Client (for example, one with a
// connection pool or middleware). The supplied client's Timeout takes effect.
func WithHTTPClient(h *http.Client) Option { return func(c *clientConfig) { c.httpClient = h } }

// WithUserAgent overrides the User-Agent header sent on every request.
func WithUserAgent(ua string) Option { return func(c *clientConfig) { c.userAgent = ua } }

// WithMaxRetries sets how many times the transport will retry 429/5xx responses
// before returning an error. Defaults to 2; total attempts is N+1.
func WithMaxRetries(n int) Option { return func(c *clientConfig) { c.maxRetries = n } }

// WithTimeout is a shortcut for WithHTTPClient(&http.Client{Timeout: d}). If you
// also pass WithHTTPClient, the explicit *http.Client wins.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		if c.httpClient == nil {
			c.httpClient = &http.Client{Timeout: d}
		}
	}
}

// NewClient constructs a Client. Pass any combination of options:
//
//	c := voicetel.NewClient(
//	    voicetel.WithAPIKey(os.Getenv("VOICETEL_API_KEY")),
//	    voicetel.WithTimeout(30 * time.Second),
//	)
func NewClient(opts ...Option) *Client {
	cfg := clientConfig{
		baseURL:    DefaultBaseURL,
		userAgent:  DefaultUserAgent,
		maxRetries: 2,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	t := &transport{
		baseURL:    strings.TrimRight(cfg.baseURL, "/"),
		apiKey:     cfg.apiKey,
		httpClient: cfg.httpClient,
		userAgent:  cfg.userAgent,
		maxRetries: cfg.maxRetries,
	}
	c := &Client{t: t}
	c.Account = &AccountService{c: c}
	c.ACL = &ACLService{c: c}
	c.Authentication = &AuthenticationService{c: c}
	c.E911 = &E911Service{c: c}
	c.Gateways = &GatewaysService{c: c}
	c.INumbering = &INumberingService{c: c}
	c.Lookups = &LookupsService{c: c}
	c.Messaging = &MessagingService{c: c}
	c.Numbers = &NumbersService{c: c}
	c.Support = &SupportService{c: c}
	return c
}

// BaseURL returns the API endpoint this client is configured against.
func (c *Client) BaseURL() string { return c.t.baseURL }

// APIKey returns the currently installed bearer token. Returns "" before Login.
func (c *Client) APIKey() string { return c.t.apiKey }

// Login exchanges username + password for a 32-hex API key and installs it on
// this client. The exchange counts against the 6 req/hour/IP rate limit shared
// by every account/* endpoint (cdr, mrc, payments, registration, api-key).
//
// The endpoint accepts username as either an int or a string of digits. Use
// the typed forms (int) where you can.
func (c *Client) Login(ctx context.Context, username int, password string) (string, error) {
	var data AccountApiKeyData
	body := map[string]any{"username": username, "password": password}
	if err := c.t.request(ctx, "POST", "/v2.2/account/api-key", nil, body, &data, false); err != nil {
		return "", err
	}
	if data.APIKey == "" {
		return "", &APIError{
			Kind:    KindAuthentication,
			Message: "api-key response did not contain data.apikey",
			Body:    data,
		}
	}
	c.t.setBearer(data.APIKey)
	return data.APIKey, nil
}

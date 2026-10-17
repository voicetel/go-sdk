package voicetel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestServer returns a Client wired to an httptest.Server whose handler is
// invoked for every request, plus a teardown the caller must defer.
func newTestServer(t *testing.T, handler http.HandlerFunc, opts ...Option) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	all := append([]Option{WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(0)}, opts...)
	return NewClient(all...), srv
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.BaseURL() != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.BaseURL(), DefaultBaseURL)
	}
	if c.APIKey() != "" {
		t.Errorf("APIKey unexpectedly set: %q", c.APIKey())
	}
	if c.t.userAgent != DefaultUserAgent {
		t.Errorf("user agent = %q, want %q", c.t.userAgent, DefaultUserAgent)
	}
	if c.t.maxRetries != 2 {
		t.Errorf("maxRetries = %d, want 2", c.t.maxRetries)
	}
}

func TestWithOptionsOverride(t *testing.T) {
	c := NewClient(
		WithBaseURL("https://example.test/"),
		WithAPIKey("abc"),
		WithUserAgent("custom/1.0"),
		WithMaxRetries(5),
		WithTimeout(2*time.Second),
	)
	if c.BaseURL() != "https://example.test" {
		t.Errorf("trailing slash should be stripped, got %q", c.BaseURL())
	}
	if c.APIKey() != "abc" {
		t.Error("WithAPIKey did not take effect")
	}
	if c.t.userAgent != "custom/1.0" {
		t.Error("WithUserAgent did not take effect")
	}
	if c.t.maxRetries != 5 {
		t.Error("WithMaxRetries did not take effect")
	}
}

func TestWithHTTPClientPreservedAcrossTimeout(t *testing.T) {
	custom := &http.Client{Timeout: 999 * time.Millisecond}
	c := NewClient(WithHTTPClient(custom), WithTimeout(50*time.Millisecond))
	if c.t.httpClient != custom {
		t.Error("WithTimeout should not override an explicit WithHTTPClient")
	}
}

func TestRequestSendsBearerAndUserAgent(t *testing.T) {
	var captured *http.Request
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	}, WithAPIKey("k"+strings.Repeat("0", 31)), WithUserAgent("ua/1"))

	if _, err := c.Account.Get(context.Background()); err != nil {
		t.Fatalf("Account.Get: %v", err)
	}
	if got := captured.Header.Get("Authorization"); got != "Bearer k"+strings.Repeat("0", 31) {
		t.Errorf("Authorization = %q", got)
	}
	if captured.Header.Get("User-Agent") != "ua/1" {
		t.Errorf("user-agent = %q", captured.Header.Get("User-Agent"))
	}
	if captured.Header.Get("Accept") != "application/json" {
		t.Errorf("accept = %q", captured.Header.Get("Accept"))
	}
}

func TestRequestOmitsAuthWhenNotRequired(t *testing.T) {
	var captured *http.Request
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		_, _ = w.Write([]byte(`{"status":"success","data":{"message":"sent"}}`))
	}, WithAPIKey(""))

	_, err := c.Account.Recover(context.Background(), AccountRecoverRequest{Email: "x@y.com"})
	if err != nil {
		t.Fatalf("Account.Recover: %v", err)
	}
	if h := captured.Header.Get("Authorization"); h != "" {
		t.Errorf("Authorization unexpectedly set: %q", h)
	}
}

func TestRequestErrorsBeforeSendWhenAuthMissing(t *testing.T) {
	c := NewClient(WithBaseURL("http://0.0.0.0:1"))
	_, err := c.Account.Get(context.Background())
	if !IsAuthentication(err) {
		t.Fatalf("expected KindAuthentication, got %v", err)
	}
}

func TestErrorMappingByStatus(t *testing.T) {
	for status, want := range map[int]ErrorKind{
		400: KindBadRequest,
		401: KindAuthentication,
		403: KindPermissionDenied,
		404: KindNotFound,
		409: KindConflict,
		429: KindRateLimit,
		500: KindServer,
		503: KindServer,
		418: KindUnknown,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"status":"error","error":{"code":` +
					strconv.Itoa(status) + `,"message":"boom"}}`))
			})
			_, err := c.Account.Get(context.Background())
			ae, ok := err.(*APIError)
			if !ok {
				t.Fatalf("err is not *APIError: %v", err)
			}
			if ae.Kind != want {
				t.Errorf("Kind = %v, want %v", ae.Kind, want)
			}
			if ae.StatusCode != status {
				t.Errorf("StatusCode = %d, want %d", ae.StatusCode, status)
			}
			if ae.Code != status {
				t.Errorf("Code = %d, want %d", ae.Code, status)
			}
			if !strings.Contains(ae.Error(), "boom") {
				t.Errorf("Error() = %q (missing 'boom')", ae.Error())
			}
		})
	}
}

// TestErrorEnvelopeSpecShape pins decoding of the spec's errorResponse
// envelope: {"status":"error","error":{"code":<int>,"message":<string>}}.
func TestErrorEnvelopeSpecShape(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"status":"error","error":{"code":400,"message":"Field Required: number"}}`))
	})
	_, err := c.Messaging.History(context.Background(), HistoryOptions{})
	ae, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err is not *APIError: %v", err)
	}
	if ae.Code != 400 {
		t.Errorf("Code = %d, want 400", ae.Code)
	}
	if ae.Message != "Field Required: number" {
		t.Errorf("Message = %q", ae.Message)
	}
	if ae.Kind != KindBadRequest {
		t.Errorf("expected KindBadRequest, got %v", ae.Kind)
	}
	if _, ok := ae.Body.(map[string]any); !ok {
		t.Errorf("Body should be the decoded envelope map, got %T", ae.Body)
	}
}

// TestAclConflictDecoded exercises the structured 409 body from POST/DELETE
// /v2.2/acl: the APIError must expose it via AclConflict.
func TestAclConflictDecoded(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"status":"error","error":{"code":409,"message":"CIDR overlaps with existing entry"},` +
			`"data":{"added":[{"cidr":"203.0.113.0/24"}],"removed":[],"failed":[{"cidr":"abc123","reason":"Invalid mask: must be /8, /16, /24, or /32"}]}}`))
	})
	_, err := c.ACL.Add(context.Background(), AclModifyRequest{ACL: []CidrEntry{{CIDR: "203.0.113.0/24"}}})
	if !IsConflict(err) {
		t.Fatalf("expected KindConflict, got %v", err)
	}
	ae := err.(*APIError)
	if ae.Message != "CIDR overlaps with existing entry" || ae.Code != 409 {
		t.Errorf("envelope not parsed: %+v", ae)
	}
	data, ok := ae.AclConflict()
	if !ok {
		t.Fatalf("AclConflict not decoded; Body=%#v", ae.Body)
	}
	if data.Added[0].CIDR != "203.0.113.0/24" || data.Failed[0].CIDR != "abc123" {
		t.Errorf("conflict data wrong: %+v", data)
	}
	if !strings.Contains(data.Failed[0].Reason, "Invalid mask") {
		t.Errorf("failed reason: %+v", data.Failed[0])
	}
}

// TestAuthPutConflictDecoded exercises the structured 409 body from
// PUT /v2.2/auth: partially-applied updates plus per-field rejections.
func TestAuthPutConflictDecoded(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"status":"error","error":{"code":409,"message":"One or more updates failed"},` +
			`"updated":[{"field":"authType","value":1}],` +
			`"errors":[{"field":"password","message":"Must be 6-10 alphanumeric characters with at least one letter and one number"}]}`))
	})
	_, err := c.Authentication.Update(context.Background(), AuthPutRequest{AuthType: Int(1), Password: String("weak")})
	if !IsConflict(err) {
		t.Fatalf("expected KindConflict, got %v", err)
	}
	ae := err.(*APIError)
	data, ok := ae.AuthPutConflict()
	if !ok {
		t.Fatalf("AuthPutConflict not decoded; Body=%#v", ae.Body)
	}
	if data.Updated[0].Field != "authType" || data.Updated[0].Value != 1 {
		t.Errorf("updated not decoded: %+v", data.Updated)
	}
	if data.Errors[0].Field != "password" || !strings.Contains(data.Errors[0].Message, "6-10") {
		t.Errorf("errors not decoded: %+v", data.Errors)
	}
}

// TestConflictHelpersRejectNonConflict guards the kind gate on the 409 helpers.
func TestConflictHelpersRejectNonConflict(t *testing.T) {
	notFound := &APIError{Kind: KindNotFound, StatusCode: 404, Body: map[string]any{}}
	if _, ok := notFound.AclConflict(); ok {
		t.Error("AclConflict on 404 should be !ok")
	}
	if _, ok := notFound.AuthPutConflict(); ok {
		t.Error("AuthPutConflict on 404 should be !ok")
	}
	wrongShape := &APIError{Kind: KindConflict, StatusCode: 409, Body: map[string]any{"status": "error"}}
	if _, ok := wrongShape.AclConflict(); ok {
		t.Error("AclConflict without data should be !ok")
	}
	if _, ok := wrongShape.AuthPutConflict(); ok {
		t.Error("AuthPutConflict without updated/errors should be !ok")
	}
	rawString := &APIError{Kind: KindConflict, StatusCode: 409, Body: "conflict"}
	if _, ok := rawString.AclConflict(); ok {
		t.Error("AclConflict on a string body should be !ok")
	}
}

func TestErrorWithNonJSONBody(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("plain text"))
	})
	_, err := c.Account.Get(context.Background())
	ae, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err is not *APIError: %v", err)
	}
	if body, _ := ae.Body.(string); body != "plain text" {
		t.Errorf("Body = %#v, want plain text", ae.Body)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var calls int32
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"username":"1"}}`))
	}, WithMaxRetries(2))

	me, err := c.Account.Get(context.Background())
	if err != nil {
		t.Fatalf("Account.Get: %v", err)
	}
	if me.Username != "1" {
		t.Errorf("retry didn't deliver final body, got %#v", me)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestRetryExhaustedReturnsRateLimit(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
	}, WithMaxRetries(1))

	_, err := c.Account.Get(context.Background())
	if !IsRateLimit(err) {
		t.Fatalf("expected RateLimit after exhausted retries, got %v", err)
	}
}

func TestRetryOn503(t *testing.T) {
	var calls int32
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	}, WithMaxRetries(1))

	if _, err := c.Account.Get(context.Background()); err != nil {
		t.Fatalf("Account.Get: %v", err)
	}
}

func TestRetryDoesNotConsumeContextDeadline(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// Server will sleep longer than the context deadline.
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(429)
	}, WithMaxRetries(2))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := c.Account.Get(ctx)
	if err == nil {
		t.Fatal("expected context-deadline error")
	}
}

func TestTransportErrorRetriesThenSurfaces(t *testing.T) {
	// Point at a closed port so the transport fails immediately.
	c := NewClient(WithBaseURL("http://127.0.0.1:1"), WithAPIKey("k"), WithMaxRetries(1),
		WithHTTPClient(&http.Client{Timeout: 50 * time.Millisecond}))
	_, err := c.Account.Get(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
	ae, ok := err.(*APIError)
	if !ok {
		t.Fatalf("not *APIError: %T %v", err, err)
	}
	if !strings.Contains(ae.Error(), "transport error") {
		t.Errorf("Error() = %q", ae.Error())
	}
	if !errors.Is(ae, ae.Unwrap()) {
		t.Error("Unwrap should expose underlying cause")
	}
}

func TestContextCancelledBeforeSend(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}, WithMaxRetries(0))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Account.Get(ctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestEmpty204ReturnsNil(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	})
	if err := c.Numbers.Remove(context.Background(), "2015550123"); err != nil {
		t.Fatalf("remove: %v", err)
	}
}

func TestUnwrapPassesThroughNonEnvelopeBody(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// Bare object, not wrapped — the SDK should still decode it.
		_, _ = w.Write([]byte(`{"cnam":"VOICETEL","number":"2015550123"}`))
	})
	r, err := c.Lookups.CNAM(context.Background(), "2015550123")
	if err != nil {
		t.Fatalf("CNAM: %v", err)
	}
	if r.CNAM != "VOICETEL" {
		t.Errorf("got %#v", r)
	}
}

func TestUnwrapHandlesMalformedJSON(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not json}`))
	})
	_, err := c.Account.Get(context.Background())
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestQueryParamsSerialized(t *testing.T) {
	var captured *http.Request
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		_, _ = w.Write([]byte(`{"status":"success","data":{"start":1,"end":2,"cdr":[]}}`))
	})
	_, err := c.Account.CDR(context.Background(), 1747345200, 1747258800)
	if err != nil {
		t.Fatalf("CDR: %v", err)
	}
	if got := captured.URL.Query().Get("start"); got != "1747345200" {
		t.Errorf("start = %q", got)
	}
	if got := captured.URL.Query().Get("end"); got != "1747258800" {
		t.Errorf("end = %q", got)
	}
}

func TestLoginInstallsBearerThenAuthenticates(t *testing.T) {
	calls := map[string]int{}
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/v2.2/account/api-key":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"username":"1000000001"`) {
				t.Errorf("login body should carry username as a JSON string: %s", body)
			}
			if r.Header.Get("Authorization") != "" {
				t.Errorf("login should not send Authorization")
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"apikey":"32hex"}}`))
		case "/v2.2/account":
			if r.Header.Get("Authorization") != "Bearer 32hex" {
				t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"username":"1000000001"}}`))
		}
	}, WithAPIKey(""))

	key, err := c.Login(context.Background(), "1000000001", "pw")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if key != "32hex" {
		t.Errorf("key = %q", key)
	}
	if c.APIKey() != "32hex" {
		t.Errorf("client.APIKey() = %q", c.APIKey())
	}
	if _, err := c.Account.Get(context.Background()); err != nil {
		t.Fatalf("post-login Get: %v", err)
	}
	if calls["/v2.2/account/api-key"] != 1 || calls["/v2.2/account"] != 1 {
		t.Errorf("call counts: %v", calls)
	}
}

func TestLoginWithoutAPIKeyInResponseFails(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	}, WithAPIKey(""))
	_, err := c.Login(context.Background(), "1", "p")
	if !IsAuthentication(err) {
		t.Fatalf("expected KindAuthentication, got %v", err)
	}
}

func TestErrorHelpers(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		fn   func(error) bool
		want bool
	}{
		{"rate-limit positive", &APIError{Kind: KindRateLimit}, IsRateLimit, true},
		{"rate-limit non-api", errors.New("io: short read"), IsRateLimit, false},
		{"not-found", &APIError{Kind: KindNotFound}, IsNotFound, true},
		{"auth", &APIError{Kind: KindAuthentication}, IsAuthentication, true},
		{"conflict", &APIError{Kind: KindConflict}, IsConflict, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(tc.err); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBackoffDelayHonorsRetryAfter(t *testing.T) {
	r := &http.Response{Header: make(http.Header)}
	r.Header.Set("Retry-After", "0")
	if d := backoffDelay(0, r); d != 0 {
		t.Errorf("delay with Retry-After:0 = %v", d)
	}
	r.Header.Set("Retry-After", "garbage")
	if d := backoffDelay(0, r); d == 0 {
		t.Error("delay should fall back to exponential when Retry-After is unparseable")
	}
	// A server-sent Retry-After beyond the ceiling is clamped to it.
	r.Header.Set("Retry-After", "120")
	if d := backoffDelay(0, r); d != 8*time.Second {
		t.Errorf("Retry-After 120 honored as %v, want capped at 8s", d)
	}
	// Exponential cap.
	if d := backoffDelay(20, nil); d != 8*time.Second {
		t.Errorf("cap = %v, want 8s", d)
	}
}

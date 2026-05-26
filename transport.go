package voicetel

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// transport is the low-level HTTP client used by every resource service.
// It is unexported; callers configure it indirectly via Client and the
// functional options passed to NewClient.
type transport struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	userAgent  string
	maxRetries int
}

// Mutators used both by Client.Login (to install the freshly-exchanged key)
// and by tests that want a deterministic transport.
func (t *transport) setBearer(apiKey string) { t.apiKey = apiKey }

// retryableStatuses is the set of response codes we'll back off and retry.
var retryableStatuses = map[int]struct{}{
	429: {}, 500: {}, 502: {}, 503: {}, 504: {},
}

// request performs an HTTP call. body is JSON-marshaled and sent as the request body
// if non-nil; out (when non-nil) receives the decoded JSON response.
//
// requireAuth=false skips the Authorization header — used only by the api-key
// exchange and the public password-recovery endpoint.
func (t *transport) request(
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
	out any,
	requireAuth bool,
) error {
	if requireAuth && t.apiKey == "" {
		return &APIError{
			Kind:    KindAuthentication,
			Message: "no api key set; call client.Login or pass WithAPIKey to NewClient",
		}
	}

	target := t.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return &APIError{Message: "marshal request body", cause: err}
		}
	}

	var idempotencyKey string
	if method == "POST" || method == "PUT" || method == "PATCH" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		idempotencyKey = hex.EncodeToString(b)
	}

	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(bodyBytes))
		if err != nil {
			return &APIError{Message: "build request", cause: err}
		}
		req.Header.Set("User-Agent", t.userAgent)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if requireAuth {
			req.Header.Set("Authorization", "Bearer "+t.apiKey)
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := t.httpClient.Do(req)
		if err != nil {
			// Transport-level failure (DNS, TCP, TLS, context).
			lastErr = err
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return &APIError{Message: "context cancelled before response", cause: err}
			}
			if attempt >= t.maxRetries {
				return &APIError{
					Message: fmt.Sprintf("transport error after %d attempt(s): %v", attempt+1, err),
					cause:   err,
				}
			}
			if waitErr := sleep(ctx, backoffDelay(attempt, nil)); waitErr != nil {
				return &APIError{Message: "context cancelled during backoff", cause: waitErr}
			}
			continue
		}

		if _, retry := retryableStatuses[resp.StatusCode]; retry && attempt < t.maxRetries {
			delay := backoffDelay(attempt, resp)
			_ = resp.Body.Close()
			if waitErr := sleep(ctx, delay); waitErr != nil {
				return &APIError{Message: "context cancelled during backoff", cause: waitErr}
			}
			continue
		}

		return decode(resp, out)
	}

	// Defensive — the loop above always returns.
	return &APIError{Message: "retry loop exhausted", cause: lastErr}
}

// decode parses the response: drains 2xx into out (when supplied),
// or converts the body into an APIError.
func decode(resp *http.Response, out any) error {
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "read response body", cause: err}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || len(raw) == 0 {
			return nil
		}
		// Strip the envelope before decoding into out so callers receive the inner data shape.
		inner, err := unwrap(raw)
		if err != nil {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "decode response envelope",
				Body:       string(raw),
				cause:      err,
			}
		}
		if err := json.Unmarshal(inner, out); err != nil {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "decode response body",
				Body:       string(inner),
				cause:      err,
			}
		}
		return nil
	}

	// Error path — try to extract a structured code/message.
	var body any = string(raw)
	var code, message string
	var generic map[string]any
	if json.Unmarshal(raw, &generic) == nil {
		body = generic
		if v, ok := generic["code"].(string); ok {
			code = v
		} else if v, ok := generic["error"].(string); ok {
			code = v
		}
		if v, ok := generic["message"].(string); ok {
			message = v
		} else if v, ok := generic["error"].(string); ok {
			message = v
		}
	}
	if message == "" {
		message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return errorFromStatus(resp.StatusCode, code, message, body)
}

// unwrap strips the {status, data} envelope when present. Returns the inner
// data payload as raw JSON; if no envelope, returns the input unchanged.
func unwrap(raw []byte) ([]byte, error) {
	// Cheap path: only attempt unwrapping if it looks like a JSON object.
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return raw, nil
	}
	var env struct {
		Status *string         `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Status != nil && len(env.Data) > 0 {
		return env.Data, nil
	}
	return raw, nil
}

// backoffDelay returns the delay before retry attempt+1. Honors a Retry-After
// header if the server sent one as integer seconds.
func backoffDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if h := resp.Header.Get("Retry-After"); h != "" {
			if secs, err := strconv.Atoi(h); err == nil && secs >= 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	// Exponential, capped at 8s.
	base := 500 * time.Millisecond
	d := base << attempt
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	return d
}

// sleep is a context-aware time.Sleep — returns nil after d elapses, or the
// context's error if it cancels first.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

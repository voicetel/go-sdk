package voicetel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHelperConstructors(t *testing.T) {
	if s := String("hi"); s == nil || *s != "hi" {
		t.Errorf("String() = %v", s)
	}
	if i := Int(42); i == nil || *i != 42 {
		t.Errorf("Int() = %v", i)
	}
	if b := Bool(true); b == nil || *b != true {
		t.Errorf("Bool() = %v", b)
	}
	if f := Float64(3.14); f == nil || *f != 3.14 {
		t.Errorf("Float64() = %v", f)
	}
}

// TestUnmarshalErrorOnTypeMismatch exercises the decode() path where the JSON
// envelope is valid but the inner data type doesn't match the target struct.
// (This catches actual server-side spec drift in practice.)
func TestUnmarshalErrorOnTypeMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// AccountData is a struct, but the server returns a string in `data`.
		_, _ = w.Write([]byte(`{"status":"success","data":"surprise!"}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(0))
	_, err := c.Account.Get(context.Background())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode response body") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestContextCancelDuringBackoff covers the path where ctx is cancelled while
// the transport is sleeping between retries.
func TestContextCancelDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Always 429 — forces the transport into the retry/sleep loop.
		w.Header().Set("Retry-After", "10") // long enough for the test to cancel first
		w.WriteHeader(429)
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(3))

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after the first request fires.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := c.Account.Get(ctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

// TestSleepReturnsImmediatelyForZeroDuration is a small unit-level check on the
// helper used by retry backoff.
func TestSleepReturnsImmediatelyForZeroDuration(t *testing.T) {
	start := time.Now()
	if err := sleep(context.Background(), 0); err != nil {
		t.Fatalf("sleep(0): %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Errorf("zero sleep took %v", elapsed)
	}
}

// TestTransportErrorThenSuccess exercises the path where the first attempt
// errors at the transport layer and the second succeeds.
func TestTransportErrorThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			// Force a transport-level disconnect by hijacking and closing without writing.
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"success","data":{"username":"1"}}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(2))
	me, err := c.Account.Get(context.Background())
	if err != nil {
		t.Fatalf("Account.Get after transient error: %v", err)
	}
	if me.Username != "1" {
		t.Errorf("got %#v", me)
	}
}

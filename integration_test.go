//go:build integration

// Live read-only checks against api.voicetel.com.
//
// These tests are gated behind the `integration` build tag and the
// VOICETEL_USERNAME / VOICETEL_PASSWORD env vars. Run them with:
//
//	export VOICETEL_USERNAME=...
//	export VOICETEL_PASSWORD=...
//	go test -tags=integration ./... -v -run Integration
//
// Strict rules in this file:
//   - No state mutations. No POST/PUT/PATCH/DELETE.
//   - Login + account/registration + account/cdr + account/recurring-charges + account/payments
//     all share a 6/hr/IP rate limit. Run sparingly.

package voicetel

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func integrationClient(t *testing.T) *Client {
	t.Helper()
	user := os.Getenv("VOICETEL_USERNAME")
	pass := os.Getenv("VOICETEL_PASSWORD")
	if user == "" || pass == "" {
		t.Skip("VOICETEL_USERNAME and VOICETEL_PASSWORD must be set for integration tests")
	}
	base := os.Getenv("VOICETEL_BASE_URL")
	if base == "" {
		base = DefaultBaseURL
	}

	uid, err := strconv.Atoi(user)
	if err != nil {
		t.Fatalf("VOICETEL_USERNAME must be numeric: %v", err)
	}

	c := NewClient(WithBaseURL(base), WithTimeout(30*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := c.Login(ctx, uid, pass); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return c
}

func TestIntegrationAccountGet(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	me, err := c.Account.Get(ctx)
	if err != nil {
		t.Fatalf("Account.Get: %v", err)
	}
	if me.Username == "" {
		t.Error("Account.Get returned empty Username")
	}
}

func TestIntegrationReadOnlyLists(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for name, fn := range map[string]func() error{
		"Numbers.List":  func() error { _, err := c.Numbers.List(ctx); return err },
		"Gateways.List": func() error { _, err := c.Gateways.List(ctx); return err },
		"ACL.List":      func() error { _, err := c.ACL.List(ctx); return err },
		"E911.List":     func() error { _, err := c.E911.List(ctx); return err },
		"Support.List":  func() error { _, err := c.Support.List(ctx); return err },
		"INumbering.Coverage": func() error {
			_, err := c.INumbering.Coverage(ctx, CoverageQuery{})
			return err
		},
		"INumbering.Ports":         func() error { _, err := c.INumbering.Ports(ctx); return err },
		"Authentication.Get":       func() error { _, err := c.Authentication.Get(ctx); return err },
		"Messaging.CampaignStatus": func() error { _, err := c.Messaging.CampaignStatus(ctx); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := fn(); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		})
	}
}

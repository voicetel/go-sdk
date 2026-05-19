package voicetel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEveryMethodPropagatesAPIError points every resource method at a server
// that always replies 500 and verifies the SDK returns an APIError instead of
// silently dropping the failure. This covers the `return nil, err` branch in
// every resource method, which the happy-path tests don't reach.
func TestEveryMethodPropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"code":"X","message":"server is angry"}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(0))
	ctx := context.Background()

	assertErr := func(t *testing.T, name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: expected error, got nil", name)
			return
		}
		if _, ok := err.(*APIError); !ok {
			t.Errorf("%s: expected *APIError, got %T %v", name, err, err)
		}
	}

	// Account
	_, err := c.Account.Get(ctx)
	assertErr(t, "Account.Get", err)
	_, err = c.Account.Update(ctx, AccountPutRequest{Timezone: String("UTC")})
	assertErr(t, "Account.Update", err)
	_, err = c.Account.Add(ctx, AccountAddRequest{Username: 1, Name: "x", Email: "x@y.com"})
	assertErr(t, "Account.Add", err)
	_, err = c.Account.Signup(ctx, AccountSignupRequest{Name: "x", Email: "x@y.com"})
	assertErr(t, "Account.Signup", err)
	_, err = c.Account.CDR(ctx, 1, 2)
	assertErr(t, "Account.CDR", err)
	_, err = c.Account.Credits(ctx)
	assertErr(t, "Account.Credits", err)
	_, err = c.Account.RecurringCharges(ctx)
	assertErr(t, "Account.RecurringCharges", err)
	_, err = c.Account.Payments(ctx)
	assertErr(t, "Account.Payments", err)
	_, err = c.Account.Registration(ctx)
	assertErr(t, "Account.Registration", err)
	_, err = c.Account.Recover(ctx, AccountRecoverRequest{Email: "x@y.com"})
	assertErr(t, "Account.Recover", err)

	// ACL
	_, err = c.ACL.List(ctx)
	assertErr(t, "ACL.List", err)
	_, err = c.ACL.Add(ctx, AclModifyRequest{ACL: []CidrEntry{{CIDR: "203.0.113.0/24"}}})
	assertErr(t, "ACL.Add", err)
	_, err = c.ACL.Remove(ctx, AclModifyRequest{ACL: []CidrEntry{{CIDR: "203.0.113.0/24"}}})
	assertErr(t, "ACL.Remove", err)

	// Authentication
	_, err = c.Authentication.Get(ctx)
	assertErr(t, "Authentication.Get", err)
	_, err = c.Authentication.Update(ctx, AuthPutRequest{AuthType: Int(1)})
	assertErr(t, "Authentication.Update", err)

	// E911
	_, err = c.E911.List(ctx)
	assertErr(t, "E911.List", err)
	_, err = c.E911.Create(ctx, E911CreateRequest{DN: "2015551234", Callername: "x", Address1: "1", City: "C", State: "NJ", Zip: "07601"})
	assertErr(t, "E911.Create", err)
	_, err = c.E911.Validate(ctx, E911AddressRequest{Address1: "1", City: "C", State: "NJ", Zip: "07601"})
	assertErr(t, "E911.Validate", err)
	_, err = c.E911.Get(ctx, "2015551234")
	assertErr(t, "E911.Get", err)
	_, err = c.E911.Provision(ctx, "2015551234", E911ProvisionByIDRequest{Callername: "x", AddressID: 1})
	assertErr(t, "E911.Provision", err)
	if err := c.E911.Remove(ctx, "2015551234"); err == nil {
		t.Error("E911.Remove expected error")
	}

	// Gateways
	_, err = c.Gateways.List(ctx)
	assertErr(t, "Gateways.List", err)
	_, err = c.Gateways.Add(ctx, GatewayAddRequest{Gateway: "1.2.3.4"})
	assertErr(t, "Gateways.Add", err)
	_, err = c.Gateways.Get(ctx, 1)
	assertErr(t, "Gateways.Get", err)
	_, err = c.Gateways.Update(ctx, 1, GatewayUpdateRequest{Prefix: String("9")})
	assertErr(t, "Gateways.Update", err)
	if err := c.Gateways.Remove(ctx, 1); err == nil {
		t.Error("Gateways.Remove expected error")
	}
	_, err = c.Gateways.Numbers(ctx, 1)
	assertErr(t, "Gateways.Numbers", err)

	// Lookups
	_, err = c.Lookups.CNAM(ctx, "2015551234")
	assertErr(t, "Lookups.CNAM", err)
	_, err = c.Lookups.LRN(ctx, "2015551234", "2012548000")
	assertErr(t, "Lookups.LRN", err)

	// Messaging
	_, err = c.Messaging.History(ctx, HistoryOptions{})
	assertErr(t, "Messaging.History", err)
	_, err = c.Messaging.Send(ctx, MessageSendRequest{FromNumber: "2012548000", ToNumber: "2015551234", Text: "hi"})
	assertErr(t, "Messaging.Send", err)
	_, err = c.Messaging.CreateBrand(ctx, MessagingBrandCreateRequest{MessagingBrandID: "BABC", MessagingBrandName: "X"})
	assertErr(t, "Messaging.CreateBrand", err)
	_, err = c.Messaging.CampaignStatus(ctx)
	assertErr(t, "Messaging.CampaignStatus", err)
	_, err = c.Messaging.CreateCampaign(ctx, MessagingCampaignCreateRequest{MessagingBrandID: "B", ExternalCampaignID: "C", CampaignDescription: "d"})
	assertErr(t, "Messaging.CreateCampaign", err)
	_, err = c.Messaging.NumbersState(ctx, nil)
	assertErr(t, "Messaging.NumbersState", err)

	// Numbers
	_, err = c.Numbers.List(ctx)
	assertErr(t, "Numbers.List", err)
	_, err = c.Numbers.Add(ctx, NumberAddRequest{Number: "2015551234"})
	assertErr(t, "Numbers.Add", err)
	_, err = c.Numbers.Get(ctx, "2015551234")
	assertErr(t, "Numbers.Get", err)
	if err := c.Numbers.Remove(ctx, "2015551234"); err == nil {
		t.Error("Numbers.Remove expected error")
	}
	_, err = c.Numbers.Move(ctx, "2015551234", NumberMoveRequest{AccountID: 1, Route: 4})
	assertErr(t, "Numbers.Move", err)
	if err := c.Numbers.Release(ctx, "2015551234"); err == nil {
		t.Error("Numbers.Release expected error")
	}
	_, err = c.Numbers.SetRoute(ctx, "2015551234", NumberRouteRequest{Route: 7})
	assertErr(t, "Numbers.SetRoute", err)
	_, err = c.Numbers.SetTranslation(ctx, "2015551234", NumberTranslationRequest{Translation: "2015551235"})
	assertErr(t, "Numbers.SetTranslation", err)
	_, err = c.Numbers.SetCNAM(ctx, "2015551234", NumberCnamRequest{Enabled: true})
	assertErr(t, "Numbers.SetCNAM", err)
	_, err = c.Numbers.SetLidb(ctx, "2015551234", NumberLidbRequest{CNAM: "ACME"})
	assertErr(t, "Numbers.SetLidb", err)
	_, err = c.Numbers.GetFax(ctx, "2015551234")
	assertErr(t, "Numbers.GetFax", err)
	_, err = c.Numbers.SetFax(ctx, "2015551234", NumberFaxRequest{Email: "f@x.com"})
	assertErr(t, "Numbers.SetFax", err)
	if err := c.Numbers.RemoveFax(ctx, "2015551234"); err == nil {
		t.Error("Numbers.RemoveFax expected error")
	}
	_, err = c.Numbers.SetForward(ctx, "2015551234", NumberForwardRequest{Destination: 2125551234})
	assertErr(t, "Numbers.SetForward", err)
	if err := c.Numbers.RemoveForward(ctx, "2015551234"); err == nil {
		t.Error("Numbers.RemoveForward expected error")
	}
	_, err = c.Numbers.GetSMS(ctx, "2015551234")
	assertErr(t, "Numbers.GetSMS", err)
	_, err = c.Numbers.SetSMS(ctx, "2015551234", NumberSmsRequest{Type: "email", Resource: "x@y.com"})
	assertErr(t, "Numbers.SetSMS", err)
	if err := c.Numbers.RemoveSMS(ctx, "2015551234"); err == nil {
		t.Error("Numbers.RemoveSMS expected error")
	}
	_, err = c.Numbers.GetMessaging(ctx, "2015551234")
	assertErr(t, "Numbers.GetMessaging", err)
	_, err = c.Numbers.PatchMessaging(ctx, "2015551234", NumberMessagingPatchRequest{RouteIn: Int(1)})
	assertErr(t, "Numbers.PatchMessaging", err)
	_, err = c.Numbers.AssignCampaign(ctx, "2015551234", NumberCampaignAssignRequest{CampaignID: "C1"})
	assertErr(t, "Numbers.AssignCampaign", err)
	_, err = c.Numbers.UnassignCampaign(ctx, "2015551234")
	assertErr(t, "Numbers.UnassignCampaign", err)
	_, err = c.Numbers.BulkUnassignCampaign(ctx, []string{"2015551234"})
	assertErr(t, "Numbers.BulkUnassignCampaign", err)
	_, err = c.Numbers.SetPortOutPin(ctx, "2015551234", PortOutPinUpdateRequest{PIN: "1234"})
	assertErr(t, "Numbers.SetPortOutPin", err)

	// Support
	_, err = c.Support.List(ctx)
	assertErr(t, "Support.List", err)
	_, err = c.Support.Create(ctx, TicketCreateRequest{Subject: "s", Message: "m"})
	assertErr(t, "Support.Create", err)
	_, err = c.Support.Get(ctx, 1)
	assertErr(t, "Support.Get", err)
	_, err = c.Support.Update(ctx, 1, TicketUpdateRequest{Status: "closed"})
	assertErr(t, "Support.Update", err)
	if err := c.Support.Delete(ctx, 1); err == nil {
		t.Error("Support.Delete expected error")
	}
	_, err = c.Support.Messages(ctx, 1)
	assertErr(t, "Support.Messages", err)
	_, err = c.Support.Reply(ctx, 1, TicketReplyRequest{Message: "ok"})
	assertErr(t, "Support.Reply", err)

	// iNumbering
	_, err = c.INumbering.SearchInventory(ctx, InventoryQuery{State: "NJ"})
	assertErr(t, "INumbering.SearchInventory", err)
	_, err = c.INumbering.Coverage(ctx, CoverageQuery{State: "NJ"})
	assertErr(t, "INumbering.Coverage", err)
	_, err = c.INumbering.Order(ctx, OrderCreateRequest{Numbers: []OrderNumber{{Value: "2015551234"}}})
	assertErr(t, "INumbering.Order", err)
	_, err = c.INumbering.Ports(ctx)
	assertErr(t, "INumbering.Ports", err)
	_, err = c.INumbering.Port(ctx, 42)
	assertErr(t, "INumbering.Port", err)
	_, err = c.INumbering.SubmitPort(ctx, PortSubmitRequest{
		DID: []string{"2015551234"}, Name: "X", NameType: "business",
		LCBtn: "1", LCAccountNumber: "1", StreetNumber: "1", Street: "M",
		StreetType: "ST", City: "C", State: "IL", Zip: "60601", Country: "US", AuthPerson: "J",
	})
	assertErr(t, "INumbering.SubmitPort", err)
	_, err = c.INumbering.PortAvailability(ctx, "2017301000")
	assertErr(t, "INumbering.PortAvailability", err)
}

// Login covers the success and missing-key cases in transport_test.go;
// here we exercise the error path where the api-key endpoint itself fails.
func TestLoginPropagatesEndpointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Login(context.Background(), 1, "p")
	if !IsRateLimit(err) {
		t.Fatalf("expected KindRateLimit, got %v", err)
	}
}

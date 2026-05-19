package voicetel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mux returns a Client wired to an httptest.Server that dispatches by
// method+path. Each handler receives the request and returns (status, body).
func mux(t *testing.T, routes map[string]func(*http.Request) (int, string)) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		fn, ok := routes[key]
		if !ok {
			t.Errorf("unexpected request: %s", key)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status, body := fn(r)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(WithBaseURL(srv.URL), WithAPIKey("k"), WithMaxRetries(0)), srv
}

func ok(data string) func(*http.Request) (int, string) {
	return func(_ *http.Request) (int, string) {
		return 200, `{"status":"success","data":` + data + `}`
	}
}

func created(data string) func(*http.Request) (int, string) {
	return func(_ *http.Request) (int, string) {
		return 201, `{"status":"success","data":` + data + `}`
	}
}

func noContent(_ *http.Request) (int, string) { return 204, "" }

// ----------------------------------------------------------------- Account ---

func TestAccountFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/account": ok(`{"username":"1","name":"Acme","cash":12.5,"rates":{"sms":0.01},"services":{"sms":true}}`),
		"PUT /v2.2/account": func(r *http.Request) (int, string) {
			var got map[string]any
			_ = json.NewDecoder(r.Body).Decode(&got)
			if got["timezone"] != "UTC" {
				t.Errorf("body missing timezone: %v", got)
			}
			return 200, `{"status":"success","data":{"updated":["timezone"]}}`
		},
		"POST /v2.2/account":                  created(`{"username":"2","password":"p"}`),
		"POST /v2.2/accounts":                 created(`{"username":"3","password":"q"}`),
		"GET /v2.2/account/cdr":               ok(`{"start":1,"end":2,"cdr":[{"id":"r1","key":["1","2"],"value":{"dur":"40","nr":"0.025"}}]}`),
		"GET /v2.2/account/credits":           ok(`{"credits":[{"amount":25,"date":"2024-01-01","paid":false}]}`),
		"GET /v2.2/account/recurring-charges": ok(`{"charges":[{"amount":0.5,"description":"DID"}],"total":0.5}`),
		"GET /v2.2/account/payments":          ok(`{"payments":[{"amount":25,"date":"2024-01-01","status":"Completed"}]}`),
		"GET /v2.2/account/registration":      ok(`{"agent":"Zoiper","uri":"sip:x"}`),
		"POST /v2.2/account/recovery":         ok(`{"message":"sent"}`),
	})

	if me, err := c.Account.Get(ctx); err != nil || me.Cash != 12.5 || me.Services.SMS != true {
		t.Fatalf("Get: %v, %#v", err, me)
	}
	if r, err := c.Account.Update(ctx, AccountPutRequest{Timezone: String("UTC")}); err != nil || r.Updated[0] != "timezone" {
		t.Fatalf("Update: %v", err)
	}
	if r, err := c.Account.Add(ctx, AccountAddRequest{Username: 2, Name: "S", Email: "s@x.com"}); err != nil || r.Password != "p" {
		t.Fatalf("Add: %v", err)
	}
	if r, err := c.Account.Signup(ctx, AccountSignupRequest{Name: "S", Email: "s@x.com"}); err != nil || r.Password != "q" {
		t.Fatalf("Signup: %v", err)
	}
	if r, err := c.Account.CDR(ctx, 1, 2); err != nil || len(r.Cdr) != 1 || r.Cdr[0].Value.Dur != "40" {
		t.Fatalf("CDR: %v", err)
	}
	if r, err := c.Account.Credits(ctx); err != nil || r.Credits[0].Amount != 25 {
		t.Fatalf("Credits: %v", err)
	}
	if r, err := c.Account.RecurringCharges(ctx); err != nil || r.Total != 0.5 {
		t.Fatalf("RecurringCharges: %v", err)
	}
	if r, err := c.Account.Payments(ctx); err != nil || r.Payments[0].Status != "Completed" {
		t.Fatalf("Payments: %v", err)
	}
	if r, err := c.Account.Registration(ctx); err != nil || r.Agent != "Zoiper" {
		t.Fatalf("Registration: %v", err)
	}
	if r, err := c.Account.Recover(ctx, AccountRecoverRequest{Email: "x@y.com"}); err != nil || r.Message != "sent" {
		t.Fatalf("Recover: %v", err)
	}
}

// ----------------------------------------------------------------------- ACL ---

func TestACLFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/acl":    ok(`{"acl":[{"cidr":"203.0.113.0/24"}]}`),
		"POST /v2.2/acl":   ok(`{"added":[{"cidr":"203.0.113.0/24"}]}`),
		"DELETE /v2.2/acl": ok(`{"removed":[{"cidr":"203.0.113.0/24"}]}`),
	})
	body := AclModifyRequest{ACL: []CidrEntry{{CIDR: "203.0.113.0/24"}}}
	if r, err := c.ACL.List(ctx); err != nil || r.ACL[0].CIDR != "203.0.113.0/24" {
		t.Fatalf("List: %v", err)
	}
	if r, err := c.ACL.Add(ctx, body); err != nil || r.Added[0].CIDR != "203.0.113.0/24" {
		t.Fatalf("Add: %v", err)
	}
	if r, err := c.ACL.Remove(ctx, body); err != nil || r.Removed[0].CIDR != "203.0.113.0/24" {
		t.Fatalf("Remove: %v", err)
	}
}

// --------------------------------------------------------------- Authentication ---

func TestAuthenticationFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/auth": ok(`{"authType":1,"authTypeDescription":"IP Auth","acl":[{"cidr":"203.0.113.0/24"}]}`),
		"PUT /v2.2/auth": ok(`{"updated":[{"field":"authType","value":2}]}`),
	})
	if r, err := c.Authentication.Get(ctx); err != nil || r.AuthType != 1 {
		t.Fatalf("Get: %v", err)
	}
	if r, err := c.Authentication.Update(ctx, AuthPutRequest{AuthType: Int(2)}); err != nil ||
		r.Updated[0].Field != "authType" || r.Updated[0].Value != 2 {
		t.Fatalf("Update: %v %#v", err, r)
	}
}

// ------------------------------------------------------------------------ E911 ---

func TestE911FullSurface(t *testing.T) {
	ctx := context.Background()
	record := `{"dn":"12015551234","callername":"ACME","address1":"1 Main","city":"Closter","state":"NJ","zip":"07624"}`
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/e911":               ok(`{"records":[` + record + `]}`),
		"POST /v2.2/e911":              created(`{"record":` + record + `}`),
		"POST /v2.2/e911/validations":  ok(`{"address":{"addressid":1,"address1":"1 Main","city":"Closter","state":"NJ","zip":"07624"}}`),
		"GET /v2.2/e911/2015551234":    ok(`{"record":` + record + `}`),
		"PUT /v2.2/e911/2015551234":    ok(`{"record":` + record + `}`),
		"DELETE /v2.2/e911/2015551234": noContent,
	})

	if r, err := c.E911.List(ctx); err != nil || r.Records[0].DN != "12015551234" {
		t.Fatalf("List: %v", err)
	}
	if r, err := c.E911.Create(ctx, E911CreateRequest{DN: "2015551234", Callername: "ACME", Address1: "1 Main", City: "Closter", State: "NJ", Zip: "07624"}); err != nil || r.Record.Callername != "ACME" {
		t.Fatalf("Create: %v", err)
	}
	if r, err := c.E911.Validate(ctx, E911AddressRequest{Address1: "1 Main", City: "Closter", State: "NJ", Zip: "07624"}); err != nil || r.Address.AddressID != 1 {
		t.Fatalf("Validate: %v", err)
	}
	if r, err := c.E911.Get(ctx, "2015551234"); err != nil || r.Record.DN != "12015551234" {
		t.Fatalf("Get: %v", err)
	}
	if r, err := c.E911.Provision(ctx, "2015551234", E911ProvisionByIDRequest{Callername: "ACME", AddressID: 1}); err != nil || r.Record.DN != "12015551234" {
		t.Fatalf("Provision: %v", err)
	}
	if err := c.E911.Remove(ctx, "2015551234"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

// --------------------------------------------------------------------- Gateways ---

func TestGatewaysFullSurface(t *testing.T) {
	ctx := context.Background()
	gw := `{"id":1000,"gateway":"1.2.3.4:5060","prefix":"9","limit":23,"system":false}`
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/gateways":              ok(`{"gateways":[` + gw + `]}`),
		"POST /v2.2/gateways":             created(gw),
		"GET /v2.2/gateways/1000":         ok(gw),
		"PUT /v2.2/gateways/1000":         ok(gw),
		"DELETE /v2.2/gateways/1000":      noContent,
		"GET /v2.2/gateways/1000/numbers": ok(`{"numbers":[{"number":"2015551234","translated":"2015551234","forward":false,"forwardTo":null,"cnam":false,"carrier":0,"smsEnabled":false,"faxEnabled":false}]}`),
	})

	if r, err := c.Gateways.List(ctx); err != nil || r.Gateways[0].ID != 1000 {
		t.Fatalf("List: %v", err)
	}
	if r, err := c.Gateways.Add(ctx, GatewayAddRequest{Gateway: "1.2.3.4:5060"}); err != nil || r.ID != 1000 {
		t.Fatalf("Add: %v", err)
	}
	if r, err := c.Gateways.Get(ctx, 1000); err != nil || r.Prefix != "9" {
		t.Fatalf("Get: %v", err)
	}
	if r, err := c.Gateways.Update(ctx, 1000, GatewayUpdateRequest{Prefix: String("9")}); err != nil || r.ID != 1000 {
		t.Fatalf("Update: %v", err)
	}
	if err := c.Gateways.Remove(ctx, 1000); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r, err := c.Gateways.Numbers(ctx, 1000); err != nil || r.Numbers[0].Number != "2015551234" {
		t.Fatalf("Numbers: %v", err)
	}
}

// ---------------------------------------------------------------------- Lookups ---

func TestLookupsFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/cnam/2012548000":           ok(`{"cnam":"VOICETEL","number":"2012548000"}`),
		"GET /v2.2/lrn/2015551234/2012548000": ok(`{"ani":"2012548000","destination":"2015551234","lrn":{"lrn":"12125550000","state":"NY"}}`),
	})
	if r, err := c.Lookups.CNAM(ctx, "2012548000"); err != nil || r.CNAM != "VOICETEL" {
		t.Fatalf("CNAM: %v", err)
	}
	if r, err := c.Lookups.LRN(ctx, "2015551234", "2012548000"); err != nil || r.LRN.LRN != "12125550000" {
		t.Fatalf("LRN: %v", err)
	}
}

// -------------------------------------------------------------------- Messaging ---

func TestMessagingFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/messages":             ok(`{"number":"2012548000","type":"sms","fromTs":1,"toTs":2,"messages":[]}`),
		"POST /v2.2/messages":            created(`{"id":"x","type":"sms","fromNumber":"2012548000","toNumber":"2015551234","parts":1}`),
		"POST /v2.2/messaging/brands":    created(`{"result":{"statusCode":"200","status":"Success"}}`),
		"GET /v2.2/messaging/campaigns":  ok(`{"campaigns":[{"id":"C1","status":"ACTIVE","numbers":["2015551234"]}]}`),
		"POST /v2.2/messaging/campaigns": created(`{"result":{"statusCode":"200","status":"Success"}}`),
		"GET /v2.2/numbers/messaging":    ok(`{"numbers":[]}`),
	})

	if r, err := c.Messaging.History(ctx, HistoryOptions{Number: "2012548000", Start: 1, End: 2, Type: "sms"}); err != nil || r.Type != "sms" {
		t.Fatalf("History: %v", err)
	}
	if r, err := c.Messaging.Send(ctx, MessageSendRequest{FromNumber: "2012548000", ToNumber: "2015551234", Text: "hi"}); err != nil || r.ID != "x" {
		t.Fatalf("Send: %v", err)
	}
	if r, err := c.Messaging.CreateBrand(ctx, MessagingBrandCreateRequest{MessagingBrandID: "BABC", MessagingBrandName: "X"}); err != nil || r.Result.Status != "Success" {
		t.Fatalf("CreateBrand: %v", err)
	}
	if r, err := c.Messaging.CampaignStatus(ctx); err != nil || r.Campaigns[0].ID != "C1" {
		t.Fatalf("CampaignStatus: %v", err)
	}
	if r, err := c.Messaging.CreateCampaign(ctx, MessagingCampaignCreateRequest{MessagingBrandID: "B", ExternalCampaignID: "C", CampaignDescription: "d"}); err != nil || r.Result.Status != "Success" {
		t.Fatalf("CreateCampaign: %v", err)
	}
	if r, err := c.Messaging.NumbersState(ctx, []string{"2015551234"}); err != nil || r.Numbers == nil {
		t.Fatalf("NumbersState: %v", err)
	}
	if r, err := c.Messaging.NumbersState(ctx, nil); err != nil || r.Numbers == nil {
		t.Fatalf("NumbersState(nil): %v", err)
	}
}

// ---------------------------------------------------------------------- Numbers ---

func TestNumbersFullSurface(t *testing.T) {
	ctx := context.Background()
	nd := `{"number":"2015551234","translated":"2015551234","route":4,"gateway":"1.2.3.4","cnam":true,"forward":false,"forwardTo":null,"carrier":0,"smsEnabled":true,"faxEnabled":false}`
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/numbers":                                  ok(`{"numbers":[` + nd + `]}`),
		"POST /v2.2/numbers":                                 created(`{"number":"2015551234","route":4}`),
		"GET /v2.2/numbers/2015551234":                       ok(nd),
		"DELETE /v2.2/numbers/2015551234":                    noContent,
		"PATCH /v2.2/numbers/2015551234":                     ok(`{"number":"2015551234","accountId":99,"route":4}`),
		"POST /v2.2/numbers/2015551234/release":              noContent,
		"PUT /v2.2/numbers/2015551234/route":                 ok(`{"number":"2015551234","route":7}`),
		"PUT /v2.2/numbers/2015551234/translation":           ok(`{"number":"2015551234","translation":"2015551235"}`),
		"PUT /v2.2/numbers/2015551234/cnam":                  ok(`{"number":"2015551234","cnam":true}`),
		"PUT /v2.2/numbers/2015551234/lidb":                  ok(`{"number":"2015551234","cnam":"ACME","customerOrderReference":"r1","carrierStatus":"Success"}`),
		"GET /v2.2/numbers/2015551234/fax":                   ok(`{"number":"2015551234","email":"f@x.com"}`),
		"PUT /v2.2/numbers/2015551234/fax":                   ok(`{"number":"2015551234","email":"f@x.com"}`),
		"DELETE /v2.2/numbers/2015551234/fax":                noContent,
		"PUT /v2.2/numbers/2015551234/forward":               ok(`{"number":"2015551234","forwardTo":"2125551234"}`),
		"DELETE /v2.2/numbers/2015551234/forward":            noContent,
		"GET /v2.2/numbers/2015551234/sms":                   ok(`{"number":"2015551234","type":"email","resource":"x@y.com"}`),
		"PUT /v2.2/numbers/2015551234/sms":                   ok(`{"number":"2015551234","type":"email","resource":"x@y.com"}`),
		"DELETE /v2.2/numbers/2015551234/sms":                noContent,
		"GET /v2.2/numbers/2015551234/messaging":             ok(`{"number":"2015551234","enabled":true,"carrier":16,"routeIn":0,"resource":"x","network":"A","campaign":null}`),
		"PATCH /v2.2/numbers/2015551234/messaging":           ok(`{"number":"2015551234","updated":["routeIn"]}`),
		"PUT /v2.2/numbers/2015551234/messaging-campaign":    ok(`{"number":"2015551234","campaignId":"C1","carrier":17,"network":"A","upstreamCnpId":"SFL9UTQ","previousNetwork":null,"previousNetworkCleared":false}`),
		"DELETE /v2.2/numbers/2015551234/messaging-campaign": ok(`{"number":"2015551234","campaignId":"C1","network":"A","upstreamCnpId":"SFL9UTQ","unassigned":true}`),
		"DELETE /v2.2/numbers/messaging-campaign":            ok(`{"campaignId":"C1","network":"A","upstreamCnpId":"SFL9UTQ","unassignedNumbers":["2015551234"],"failed":[]}`),
		"PATCH /v2.2/numbers/2015551234/port-out-pin":        ok(`{"number":"2015551234","portOutPin":"1234"}`),
	})

	if r, err := c.Numbers.List(ctx); err != nil || r.Numbers[0].Route != 4 {
		t.Fatalf("List: %v", err)
	}
	if r, err := c.Numbers.Add(ctx, NumberAddRequest{Number: "2015551234"}); err != nil || r.Route != 4 {
		t.Fatalf("Add: %v", err)
	}
	if r, err := c.Numbers.Get(ctx, "2015551234"); err != nil || !r.CNAM {
		t.Fatalf("Get: %v", err)
	}
	if err := c.Numbers.Remove(ctx, "2015551234"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r, err := c.Numbers.Move(ctx, "2015551234", NumberMoveRequest{AccountID: 99, Route: 4}); err != nil || r.AccountID != 99 {
		t.Fatalf("Move: %v", err)
	}
	if err := c.Numbers.Release(ctx, "2015551234"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if r, err := c.Numbers.SetRoute(ctx, "2015551234", NumberRouteRequest{Route: 7}); err != nil || r.Route != 7 {
		t.Fatalf("SetRoute: %v", err)
	}
	if r, err := c.Numbers.SetTranslation(ctx, "2015551234", NumberTranslationRequest{Translation: "2015551235"}); err != nil || r.Translation != "2015551235" {
		t.Fatalf("SetTranslation: %v", err)
	}
	if r, err := c.Numbers.SetCNAM(ctx, "2015551234", NumberCnamRequest{Enabled: true}); err != nil || !r.CNAM {
		t.Fatalf("SetCNAM: %v", err)
	}
	if r, err := c.Numbers.SetLidb(ctx, "2015551234", NumberLidbRequest{CNAM: "ACME"}); err != nil || r.CarrierStatus != "Success" {
		t.Fatalf("SetLidb: %v", err)
	}
	if r, err := c.Numbers.GetFax(ctx, "2015551234"); err != nil || r.Email != "f@x.com" {
		t.Fatalf("GetFax: %v", err)
	}
	if r, err := c.Numbers.SetFax(ctx, "2015551234", NumberFaxRequest{Email: "f@x.com"}); err != nil || r.Email != "f@x.com" {
		t.Fatalf("SetFax: %v", err)
	}
	if err := c.Numbers.RemoveFax(ctx, "2015551234"); err != nil {
		t.Fatalf("RemoveFax: %v", err)
	}
	if r, err := c.Numbers.SetForward(ctx, "2015551234", NumberForwardRequest{Destination: "2125551234"}); err != nil || r.ForwardTo == nil || *r.ForwardTo != "2125551234" {
		t.Fatalf("SetForward: %v", err)
	}
	if err := c.Numbers.RemoveForward(ctx, "2015551234"); err != nil {
		t.Fatalf("RemoveForward: %v", err)
	}
	if r, err := c.Numbers.GetSMS(ctx, "2015551234"); err != nil || r.Type != "email" {
		t.Fatalf("GetSMS: %v", err)
	}
	if r, err := c.Numbers.SetSMS(ctx, "2015551234", NumberSmsRequest{Type: "email", Resource: "x@y.com"}); err != nil || r.Resource != "x@y.com" {
		t.Fatalf("SetSMS: %v", err)
	}
	if err := c.Numbers.RemoveSMS(ctx, "2015551234"); err != nil {
		t.Fatalf("RemoveSMS: %v", err)
	}
	if r, err := c.Numbers.GetMessaging(ctx, "2015551234"); err != nil || r.Network == nil || *r.Network != "A" {
		t.Fatalf("GetMessaging: %v", err)
	}
	if r, err := c.Numbers.PatchMessaging(ctx, "2015551234", NumberMessagingPatchRequest{RouteIn: Int(1)}); err != nil || r.Updated[0] != "routeIn" {
		t.Fatalf("PatchMessaging: %v", err)
	}
	if r, err := c.Numbers.AssignCampaign(ctx, "2015551234", NumberCampaignAssignRequest{CampaignID: "C1"}); err != nil || r.Carrier != 17 {
		t.Fatalf("AssignCampaign: %v", err)
	}
	if r, err := c.Numbers.UnassignCampaign(ctx, "2015551234"); err != nil || !r.Unassigned {
		t.Fatalf("UnassignCampaign: %v", err)
	}
	if r, err := c.Numbers.BulkUnassignCampaign(ctx, []string{"2015551234"}); err != nil || r.UnassignedNumbers[0] != "2015551234" {
		t.Fatalf("BulkUnassignCampaign: %v", err)
	}
	if r, err := c.Numbers.SetPortOutPin(ctx, "2015551234", PortOutPinUpdateRequest{PIN: "1234"}); err != nil || r.PortOutPin != "1234" {
		t.Fatalf("SetPortOutPin: %v", err)
	}
}

// ---------------------------------------------------------------------- Support ---

func TestSupportFullSurface(t *testing.T) {
	ctx := context.Background()
	ticket := `{"id":1,"status":"active","subject":"S","number":1015}`
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/support/tickets":            ok(`{"tickets":[` + ticket + `]}`),
		"POST /v2.2/support/tickets":           created(`{"ticket":` + ticket + `}`),
		"GET /v2.2/support/tickets/1":          ok(`{"ticket":` + ticket + `}`),
		"PUT /v2.2/support/tickets/1":          ok(`{"id":1,"status":"success"}`),
		"DELETE /v2.2/support/tickets/1":       noContent,
		"GET /v2.2/support/tickets/1/messages": ok(`{"messages":[]}`),
		"POST /v2.2/support/tickets/1/replies": created(`{"message":"Reply added"}`),
	})

	if r, err := c.Support.List(ctx); err != nil || r.Tickets[0].TicketNumber != 1015 {
		t.Fatalf("List: %v %#v", err, r)
	}
	if r, err := c.Support.Create(ctx, TicketCreateRequest{Subject: "s", Message: "m"}); err != nil || r.Ticket.Subject != "S" {
		t.Fatalf("Create: %v", err)
	}
	if r, err := c.Support.Get(ctx, 1); err != nil || r.Ticket.ID != 1 {
		t.Fatalf("Get: %v", err)
	}
	if r, err := c.Support.Update(ctx, 1, TicketUpdateRequest{Status: "closed"}); err != nil || r.Status != "success" {
		t.Fatalf("Update: %v", err)
	}
	if err := c.Support.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if r, err := c.Support.Messages(ctx, 1); err != nil || r.Messages == nil {
		t.Fatalf("Messages: %v", err)
	}
	if r, err := c.Support.Reply(ctx, 1, TicketReplyRequest{Message: "ok"}); err != nil || r.Message != "Reply added" {
		t.Fatalf("Reply: %v", err)
	}
}

// ------------------------------------------------------------------- iNumbering ---

func TestINumberingFullSurface(t *testing.T) {
	ctx := context.Background()
	c, _ := mux(t, map[string]func(*http.Request) (int, string){
		"GET /v2.2/inventory":                     ok(`{"numbers":[{"number":"2019085750","rateCenter":"CLOSTER","city":"Closter","province":"NJ","lata":"224"}]}`),
		"GET /v2.2/inventory/coverage":            ok(`{"coverage":[{"count":100,"npa":"201"}]}`),
		"POST /v2.2/orders":                       created(`{"orderId":"1","amountCharged":0.5,"numbersOrdered":["2015551234"],"failed":[]}`),
		"GET /v2.2/ports":                         ok(`{"ports":[{"id":"abc","status":"Complete"}]}`),
		"GET /v2.2/ports/42":                      ok(`{"port":{"id":"abc","status":"Complete"}}`),
		"POST /v2.2/ports":                        created(`{"pid":"a3a2a","ticket":2114,"message":"ok","loaUrl":"https://x/loa","portUrl":"https://x/port"}`),
		"GET /v2.2/ports/availability/2017301000": ok(`{"number":"2017301000","portable":true,"losingCarrier":"Sinch Voice-NSR-10X-Port/1","localRoutingNumber":"6463071993","rateCenterTier":"0","reason":null}`),
	})

	if r, err := c.INumbering.SearchInventory(ctx, InventoryQuery{State: "NJ", Limit: 10, NPA: 201, NXX: 555, RateCenter: "CLOSTER", Contains: "55", EndsWith: "00"}); err != nil || r.Numbers[0].Number != "2019085750" {
		t.Fatalf("SearchInventory: %v", err)
	}
	if r, err := c.INumbering.Coverage(ctx, CoverageQuery{State: "NJ", RateCenter: "CLOSTER"}); err != nil || r.Coverage[0].Count != 100 {
		t.Fatalf("Coverage: %v", err)
	}
	body := OrderCreateRequest{Numbers: []OrderNumber{{Value: "2015551234"}, {Spec: &OrderNumberSpec{Number: "2015551235", Route: Int(4)}}}}
	if r, err := c.INumbering.Order(ctx, body); err != nil || r.OrderID != "1" {
		t.Fatalf("Order: %v", err)
	}
	if r, err := c.INumbering.Ports(ctx); err != nil || r.Ports[0].ID != "abc" {
		t.Fatalf("Ports: %v", err)
	}
	if r, err := c.INumbering.Port(ctx, 42); err != nil || r.Port.ID != "abc" {
		t.Fatalf("Port: %v", err)
	}
	if r, err := c.INumbering.SubmitPort(ctx, PortSubmitRequest{
		DID: []string{"2015551234"}, Name: "Acme", NameType: "business",
		LCBtn: "2015551000", LCAccountNumber: "acct", StreetNumber: "550",
		Street: "Main", StreetType: "ST", City: "Chicago", State: "IL", Zip: "60601", Country: "US", AuthPerson: "J",
	}); err != nil || r.PID != "a3a2a" {
		t.Fatalf("SubmitPort: %v", err)
	}
	if r, err := c.INumbering.PortAvailability(ctx, "2017301000"); err != nil || !r.Portable {
		t.Fatalf("PortAvailability: %v", err)
	} else if r.LocalRoutingNumber == nil || *r.LocalRoutingNumber != "6463071993" || r.RateCenterTier == nil || *r.RateCenterTier != "0" {
		// v2.2.10 added LocalRoutingNumber and RateCenterTier
		t.Errorf("PortAvailability v2.2.10 fields not decoded: %+v", r)
	}
}

// --------------------------------------------------- OrderNumber custom marshal ---

func TestOrderNumberMarshalJSON(t *testing.T) {
	plain, err := json.Marshal(OrderNumber{Value: "2015551234"})
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != `"2015551234"` {
		t.Errorf("plain TN: %s", plain)
	}
	obj, err := json.Marshal(OrderNumber{Spec: &OrderNumberSpec{Number: "2015551235", Route: Int(4)}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(obj), `"number":"2015551235"`) || !strings.Contains(string(obj), `"route":4`) {
		t.Errorf("spec TN: %s", obj)
	}
}

// ---------------------------------------------------------- error helpers + repr ---

func TestAPIErrorRepr(t *testing.T) {
	e := &APIError{Kind: KindNotFound, StatusCode: 404, Code: "NF", Message: "missing"}
	if !strings.Contains(e.Error(), "404") || !strings.Contains(e.Error(), "missing") {
		t.Errorf("Error() = %q", e.Error())
	}
	e2 := &APIError{StatusCode: 0, cause: errEOF}
	if !strings.Contains(e2.Error(), "voicetel:") {
		t.Errorf("cause-only repr = %q", e2.Error())
	}
}

// errEOF is a sentinel for the cause-only error test.
var errEOF = &APIError{Message: "eof"}

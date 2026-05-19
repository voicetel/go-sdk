# 📞 VoiceTel Go SDK

The official Go client for the [VoiceTel REST API](https://voicetel.com/docs/api/v2.2/) — provision numbers, place orders, validate e911, send messages, and manage your account, all with strongly-typed, context-aware Go.

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![Go](https://img.shields.io/badge/go-1.22%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Coverage](https://img.shields.io/badge/coverage-97%25-brightgreen)

## 📚 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [Quickstart](#-quickstart)
- [Authentication](#-authentication)
- [Resource Reference](#-resource-reference)
- [Error Handling](#-error-handling)
- [Cancellation and Timeouts](#-cancellation-and-timeouts)
- [Rate Limits](#-rate-limits)
- [Development](#-development)
- [API Documentation](#-api-documentation)
- [Contributors](#-contributors)
- [Sponsors](#-sponsors)
- [License](#-license)

## ✨ Features

### 🛡️ Strongly Typed End-to-End
- **Native Go structs** for every one of the 73 API operations — JSON encoded with `encoding/json`, no reflection magic.
- **Pointer types for optional request fields** — distinguish "not set" from "zero" cleanly when PATCH-ing.
- **Pointer types for nullable response fields** — `ForwardTo *string` lets you tell apart "no forward configured" from an empty destination.
- **Context-aware throughout.** Every method takes `context.Context` as the first argument; cancel and timeouts propagate down to the HTTP layer.

### 🔁 Production-Grade Transport
- Built on `net/http` — no third-party dependencies.
- **Automatic retry** with exponential backoff on 429 / 5xx — honors `Retry-After` headers, capped at 8s.
- **Configurable timeout** per client (defaults to 30s).
- **Bearer auth** managed for you; the password→key exchange is one method call (`client.Login`).
- **Structured `*APIError`** with typed `Kind` so you can `switch err.Kind { case voicetel.KindRateLimit: ... }` without parsing HTTP status codes.

### 📞 Complete API Coverage
- **Numbers** — list, get, add, remove, route, translate, CNAM, LIDB, fax, forward, SMS, messaging campaigns, port-out PIN, account moves.
- **Account** — profile, sub-accounts, CDRs, credits, payments, MRC, registration, password recovery.
- **e911** — record provisioning, address validation, lookup, removal.
- **Gateways** — list, create, update, delete, view bound numbers.
- **Messaging** — SMS & MMS sending, message history, 10DLC brand and campaign registration, per-number messaging state.
- **Lookups** — CNAM and LRN dips.
- **iNumbering** — inventory search, coverage queries, number orders, port-in submissions, port-out availability.
- **Support** — ticket create / read / update / delete, threaded messages, replies.
- **ACL** — IP allowlist management with structured 409 conflict bodies.
- **Authentication** — switch between Digest, IP-only, or hybrid modes; rotate passwords.

### 🧪 Battle-Tested
- **97% statement coverage** with `go test ./... -cover`.
- **`httptest`-based unit tests** that exercise every method and every error path.
- **Race-detector clean** (`go test -race`).
- **`go vet` and `gofmt` clean.**

### 📦 Clean Distribution
- Zero codegen footprint — every byte hand-written.
- Single module (`github.com/voicetel/go-sdk`); install with `go get`.
- No external dependencies beyond the standard library.

## 🚀 Installation

```bash
go get github.com/voicetel/go-sdk
```

Requires Go 1.22 or later.

## 🏁 Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    voicetel "github.com/voicetel/go-sdk"
)

func main() {
    c := voicetel.NewClient()

    ctx := context.Background()

    // Exchange username + password for an API key (one-time per session)
    if _, err := c.Login(ctx, 1000000001, "hunter2"); err != nil {
        log.Fatal(err)
    }

    // Typed responses — your IDE knows what `me` is.
    me, err := c.Account.Get(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Balance: $%.2f  |  Caller ID: %s\n", me.Cash, me.CallerID)

    // List your numbers
    numbers, err := c.Numbers.List(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, n := range numbers.Numbers {
        fmt.Printf("%s  route=%d  cnam=%v  sms=%v\n", n.Number, n.Route, n.CNAM, n.SMSEnabled)
    }
}
```

Or, if you already have an API key:

```go
c := voicetel.NewClient(voicetel.WithAPIKey("32hex..."))
coverage, _ := c.INumbering.Coverage(ctx, voicetel.CoverageQuery{State: "NJ"})
for _, bucket := range coverage.Coverage {
    fmt.Printf("%s-%s: %d TNs available\n", bucket.NPA, bucket.NXX, bucket.Count)
}
```

## 🔑 Authentication

Every endpoint requires `Authorization: Bearer <apikey>` **except** `POST /v2.2/account/api-key`, which exchanges username + password for a fresh key. `Client.Login()` handles the exchange and installs the returned key on the transport.

Re-fetch the API key after any password change — the old one is invalidated.

> Don't have credentials yet? Get them at **[voicetel.com/docs/api/v2.2/credentials](https://voicetel.com/docs/api/v2.2/credentials/)**.

```go
c := voicetel.NewClient()
key, err := c.Login(ctx, 1000000001, "hunter2")
// `key` is the new 32-hex bearer; the client already has it installed.
```

## 🗺️ Resource Reference

| Resource | Field on Client | Example |
|---|---|---|
| Account | `c.Account` | `c.Account.CDR(ctx, t1, t2)` |
| ACL | `c.ACL` | `c.ACL.Add(ctx, voicetel.AclModifyRequest{...})` |
| Authentication | `c.Authentication` | `c.Authentication.Update(ctx, voicetel.AuthPutRequest{AuthType: voicetel.Int(1)})` |
| e911 | `c.E911` | `c.E911.Validate(ctx, voicetel.E911AddressRequest{...})` |
| Gateways | `c.Gateways` | `c.Gateways.List(ctx)` |
| iNumbering | `c.INumbering` | `c.INumbering.SearchInventory(ctx, voicetel.InventoryQuery{NPA: 201})` |
| Lookups | `c.Lookups` | `c.Lookups.LRN(ctx, "2015551234", "2012548000")` |
| Messaging | `c.Messaging` | `c.Messaging.Send(ctx, voicetel.MessageSendRequest{...})` |
| Numbers | `c.Numbers` | `c.Numbers.AssignCampaign(ctx, "2015551234", ...)` |
| Support | `c.Support` | `c.Support.Create(ctx, voicetel.TicketCreateRequest{...})` |

Optional request fields are pointer types. Use the helper constructors when you don't want to take addresses inline:

```go
c.Account.Update(ctx, voicetel.AccountPutRequest{
    Timezone:        voicetel.String("America/Chicago"),
    NotifyThreshold: voicetel.Int(5),
    Notify:          voicetel.Bool(true),
})
```

## 🚨 Error Handling

All HTTP errors return a `*voicetel.APIError`. Inspect `Kind` or use the helpers:

| Kind | HTTP status |
|---|---|
| `KindBadRequest` | 400 |
| `KindAuthentication` | 401 |
| `KindPermissionDenied` | 403 |
| `KindNotFound` | 404 |
| `KindConflict` | 409 |
| `KindRateLimit` | 429 |
| `KindServer` | 5xx |
| `KindUnknown` | other / transport |

```go
n, err := c.Numbers.Get(ctx, "9999999999")
switch {
case voicetel.IsNotFound(err):
    fmt.Println("That number isn't on your account.")
case voicetel.IsRateLimit(err):
    fmt.Println("Slow down — backoff and retry.")
case err != nil:
    log.Fatal(err)
default:
    fmt.Printf("Got: %#v\n", n)
}
```

Or type-assert to inspect details:

```go
var ae *voicetel.APIError
if errors.As(err, &ae) {
    fmt.Printf("status=%d code=%q body=%v\n", ae.StatusCode, ae.Code, ae.Body)
}
```

## ⏱️ Cancellation and Timeouts

Every method takes `context.Context`. Use it for cancellation and per-call deadlines:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
me, err := c.Account.Get(ctx)
```

For a global per-request timeout, configure the HTTP client:

```go
c := voicetel.NewClient(voicetel.WithTimeout(30 * time.Second))
// or, with a fully custom client:
c := voicetel.NewClient(voicetel.WithHTTPClient(&http.Client{
    Timeout:   45 * time.Second,
    Transport: myInstrumentedTransport,
}))
```

## ⏱️ Rate Limits

These endpoints are limited to **6 requests per hour per IP**:

- `account/info`
- `account/mrc` (`Account.RecurringCharges`)
- `account/cdr` (`Account.CDR`)
- `account/api-key` (`Client.Login`)

The SDK automatically retries 429 responses with `Retry-After` honored, up to `WithMaxRetries(n)` (default 2). To bump it:

```go
c := voicetel.NewClient(
    voicetel.WithAPIKey(key),
    voicetel.WithMaxRetries(4),
    voicetel.WithTimeout(60 * time.Second),
)
```

## 🛠️ Development

```bash
git clone https://github.com/voicetel/go-sdk
cd go-sdk

# Run unit tests
go test ./...

# With race detector
go test ./... -race

# With coverage
go test ./... -cover

# Static checks
go vet ./...
gofmt -l .
```

## 📖 API Documentation

- **Reference docs:** [voicetel.com/docs/api/v2.2/](https://voicetel.com/docs/api/v2.2/)
- **Interactive playground:** [voicetel.com/docs/api/v2.2/playground/](https://voicetel.com/docs/api/v2.2/playground/) — try the API in your browser without writing any code
- **API credentials:** [voicetel.com/docs/api/v2.2/credentials/](https://voicetel.com/docs/api/v2.2/credentials/)

## 🙌 Contributors

- [Michael Mavroudis](https://github.com/mavroudis) — Lead Developer

Contributions welcome. Open an issue describing the change, or send a pull request against `main`.

## 💖 Sponsors

| Sponsor | Contribution |
|---------|--------------|
| [VoiceTel Communications](https://www.voicetel.com) | Primary development and production hosting |

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

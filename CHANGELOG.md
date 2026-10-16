# Changelog

## 2.2.15 — 2026-10-15

Sync with VoiceTel API spec v2.2.15.

### Module path

- The module path is now `github.com/voicetel/go-sdk/v2`. Go only resolves
  `v2.x.y` tags for a module whose path ends in `/v2`, so earlier tags could not
  be pinned and `go get` fell back to a pseudo-version of `main`. Update imports
  to `voicetel "github.com/voicetel/go-sdk/v2"` and run
  `go get github.com/voicetel/go-sdk/v2@v2.2.15`. The API surface is unchanged
  apart from the messaging-health additions below.

### Messaging

- New **messaging health** surface (the same view as the portal Messaging tab).
  The spec defines no response schema for these endpoints, so each method returns
  the decoded JSON object as `map[string]any` rather than a typed struct; only the
  request body/query is shaped.
  - `GET /v2.2/messaging/health` → `Messaging.Health(ctx)` — account-wide overview:
    summary counts, campaigns with carrier-review progress, numbers needing
    attention (with fix-it actions), 14 days of delivery counts, and the
    alert-email setting. Updated hourly.
  - `GET /v2.2/messaging/health/alerts` → `Messaging.HealthAlerts(ctx)` — whether
    new messaging problems are emailed to the account.
  - `PUT /v2.2/messaging/health/alerts` → `Messaging.SetHealthAlerts(ctx, enabled)`
    — turn alert emails on or off (at most one digest about every 20 hours).
  - `POST /v2.2/messaging/health/check` → `Messaging.HealthCheck(ctx, number)` —
    live check of one number (in service, messaging enabled, campaign approved,
    assigned by the provider, delivery failures in the last 24h, enablement in
    progress). Limited to 30 checks per hour per account.
  - `GET /v2.2/messaging/health/samples` → `Messaging.HealthSamples(ctx, number)` —
    up to 3 recent outbound messages from the number that carriers blocked as spam.

## 2.2.14 — 2026-10-14

Sync with VoiceTel API spec v2.2.14.

### Messaging

- `GET /v2.2/messaging/campaigns` — campaign status items now carry `dca`,
  the carrier (DCA) review progress: `electionComplete` (every carrier has a
  DCA assigned), `electedCarriers` (DCAs assigned so far, null when
  unavailable), and `carriers[]` — the latest per-carrier decision
  (`carrier`, `status` of ACTIVE / REJECTED / SUSPENDED / UNSUSPENDED /
  TERMINATED, `detail`, `updatedAt`). `dca` is null for
  `CAMPAIGN_INVALID_ID`.
- Campaign `status` values documented: `CAMPAIGN_DCA_COMPLETE` (ready:
  carrier review finished, numbers assigned),
  `CAMPAIGN_DCA_COMPLETE_NO_NUMBERS_ASSIGNED`, `CAMPAIGN_PENDING_DCA`
  (registry-active but carrier review unfinished — numbers cannot be enabled
  yet), `CAMPAIGN_DCA_REJECTED` (see `dca.carriers`),
  `CAMPAIGN_<registry status>`, `CAMPAIGN_INVALID_ID`.
- `POST /v2.2/messaging/campaigns` — clarified that `campaignClassName` is
  auto-populated from the campaign registry and `campaignStartDate` defaults
  to today.

### Tooling

- golangci-lint: revive's stuttering check re-enabled (0 findings).

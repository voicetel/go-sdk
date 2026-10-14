# Changelog

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

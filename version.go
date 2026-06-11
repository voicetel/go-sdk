// Package voicetel is the official Go SDK for the VoiceTel REST API.
//
// See https://voicetel.com/docs/api/v2.2/ for the full API reference,
// https://voicetel.com/docs/api/v2.2/playground/ for an interactive playground,
// and https://voicetel.com/docs/api/v2.2/credentials/ to obtain credentials.
package voicetel

// SDKVersion is this client library's semantic version.
const SDKVersion = "2.2.10"

// APIVersion is the VoiceTel REST API version this SDK targets.
const APIVersion = "v2.2.10"

// DefaultBaseURL is the production VoiceTel API endpoint.
const DefaultBaseURL = "https://api.voicetel.com"

// DefaultUserAgent is sent on every request unless WithUserAgent overrides it.
const DefaultUserAgent = "voicetel-go/" + SDKVersion + " (+https://github.com/voicetel/go-sdk)"

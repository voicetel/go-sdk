package voicetel

import (
	"context"
	"net/url"
	"strconv"
)

// FaxService sends faxes from the account's fax-capable numbers and reads
// their history and documents.
type FaxService struct{ c *Client }

// FaxJobStatus is the lifecycle state of a fax job.
type FaxJobStatus string

// Fax job states.
const (
	FaxQueued       FaxJobStatus = "queued"
	FaxSending      FaxJobStatus = "sending"
	FaxRetryPending FaxJobStatus = "retry_pending"
	FaxSent         FaxJobStatus = "sent"
	FaxFailed       FaxJobStatus = "failed"
)

// FaxNumberEntry is one fax-capable number on the account.
type FaxNumberEntry struct {
	Number        string `json:"number"`
	DestinationID int    `json:"destinationId"` // per-number destination row id; 0 = account email
	Email         string `json:"email"`
}

// FaxNumbersData is the response data for GET /v2.2/numbers/fax.
type FaxNumbersData struct {
	Numbers []FaxNumberEntry `json:"numbers"`
}

// FaxDocumentMeta is the metadata of one document attached to a fax.
type FaxDocumentMeta struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	ContentType string `json:"contentType"`
}

// FaxSendData is the response data for POST /v2.2/faxes.
type FaxSendData struct {
	ID         string            `json:"id"`
	FromNumber string            `json:"fromNumber"`
	ToNumber   string            `json:"toNumber"`
	Status     FaxJobStatus      `json:"status"` // "queued" on acceptance
	Documents  []FaxDocumentMeta `json:"documents"`
}

// FaxJob is one fax job: routing, lifecycle timestamps and documents.
type FaxJob struct {
	ID                string            `json:"id"`
	FromNumber        string            `json:"fromNumber"`
	ToNumber          string            `json:"toNumber"`
	ClientToken       *string           `json:"clientToken"` // the Idempotency-Key it was submitted with
	Status            FaxJobStatus      `json:"status"`
	SubmittedAt       string            `json:"submittedAt,omitempty"` // ISO-8601 UTC
	StartedAt         *string           `json:"startedAt,omitempty"`
	CompletedAt       *string           `json:"completedAt,omitempty"`
	Attempts          *int              `json:"attempts,omitempty"`
	FailureReason     *string           `json:"failureReason,omitempty"` // on failed jobs
	Documents         []FaxDocumentMeta `json:"documents"`
	DocumentAvailable bool              `json:"documentAvailable"`
	DocumentName      string            `json:"documentName"`
	DocumentSize      *int              `json:"documentSize,omitempty"`
}

// FaxHistoryData is the response data for GET /v2.2/faxes.
type FaxHistoryData struct {
	Number     string   `json:"number"`
	FromTs     int64    `json:"fromTs"`
	ToTs       int64    `json:"toTs"`
	Faxes      []FaxJob `json:"faxes"`
	NextCursor *string  `json:"nextCursor"`
}

// FaxHistoryOptions are the optional filters for FaxService.History.
type FaxHistoryOptions struct {
	FromTs int64  // newer edge of the range, unix seconds (default: now)
	ToTs   int64  // older edge of the range, unix seconds (default: 24h ago)
	Limit  int    // page size 1..100 (default 50)
	Cursor string // paging cursor from a previous NextCursor
	Status string // all | active | queued | sending | retry_pending | sent | failed
}

// FaxDocumentFile is one document to send with FaxService.Send.
type FaxDocumentFile struct {
	Name        string
	ContentType string // application/pdf, application/msword, or the docx type
	Content     []byte
}

// ListNumbers returns every fax-capable number on the account.
func (s *FaxService) ListNumbers(ctx context.Context) (*FaxNumbersData, error) {
	var out FaxNumbersData
	if err := s.c.t.request(ctx, "GET", "/v2.2/numbers/fax", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Send submits a fax for delivery (multipart, 201, status queued).
//
// idempotencyKey may be empty (a random key is generated); pass your own UUID
// to make retries safe across process restarts — the same documents with the
// same key return the original job instead of sending twice.
func (s *FaxService) Send(ctx context.Context, fromNumber, toNumber string, documents []FaxDocumentFile, idempotencyKey string) (*FaxSendData, error) {
	fields := map[string]string{"fromNumber": fromNumber, "toNumber": toNumber}
	files := make([]MultipartFile, 0, len(documents))
	for _, d := range documents {
		files = append(files, MultipartFile{Field: "documents", Name: d.Name, ContentType: d.ContentType, Content: d.Content})
	}
	var out FaxSendData
	if err := s.c.t.requestMultipart(ctx, "POST", "/v2.2/faxes", fields, files, &out, idempotencyKey); err != nil {
		return nil, err
	}
	return &out, nil
}

// History returns one page of the fax history for a fax number, newest first.
func (s *FaxService) History(ctx context.Context, number string, opts *FaxHistoryOptions) (*FaxHistoryData, error) {
	q := url.Values{}
	q.Set("number", number)
	if opts != nil {
		if opts.FromTs != 0 {
			q.Set("from", strconv.FormatInt(opts.FromTs, 10))
		}
		if opts.ToTs != 0 {
			q.Set("to", strconv.FormatInt(opts.ToTs, 10))
		}
		if opts.Limit != 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			q.Set("cursor", opts.Cursor)
		}
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
	}
	var out FaxHistoryData
	if err := s.c.t.request(ctx, "GET", "/v2.2/faxes", q, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one fax job. Poll until Status is FaxSent or FaxFailed.
func (s *FaxService) Get(ctx context.Context, id string) (*FaxJob, error) {
	var out FaxJob
	if err := s.c.t.request(ctx, "GET", "/v2.2/faxes/"+url.PathEscape(id), nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Document downloads the fax PDF as raw bytes (available when the job reports
// DocumentAvailable).
func (s *FaxService) Document(ctx context.Context, id string) ([]byte, error) {
	return s.c.t.requestRaw(ctx, "GET", "/v2.2/faxes/"+url.PathEscape(id)+"/document", nil, true)
}

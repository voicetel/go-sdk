package voicetel

import (
	"context"
	"strconv"
)

// SupportService manages support tickets (create, read, update, delete, reply).
type SupportService struct{ c *Client }

// TicketStatus is one of "active", "pending", "closed", "spam".
type TicketStatus = string

// TicketCreateRequest is the body for POST /v2.2/support/tickets.
type TicketCreateRequest struct {
	Subject string  `json:"subject"`
	Message string  `json:"message"`
	Email   *string `json:"email,omitempty"` // admin only: create on behalf of this customer
}

// TicketUpdateRequest is the body for PUT /v2.2/support/tickets/{id}.
type TicketUpdateRequest struct {
	Status TicketStatus `json:"status"`
}

// TicketReplyRequest is the body for POST /v2.2/support/tickets/{id}/replies.
type TicketReplyRequest struct {
	Message string `json:"message"`
}

// --- shared sub-types -----------------------------------------------------

// TicketSource describes how a ticket or thread originated.
type TicketSource struct {
	Via  string `json:"via,omitempty"`
	Type string `json:"type,omitempty"`
}

// TicketAction is the action descriptor on a thread.
type TicketAction struct {
	Text string `json:"text,omitempty"`
	Type string `json:"type,omitempty"`
}

// TicketActor is the createdBy/assignee/assignedTo/closedByUser shape.
type TicketActor struct {
	ID        int    `json:"id,omitempty"`
	Type      string `json:"type,omitempty"` // "customer" or "user"
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	PhotoURL  string `json:"photoUrl,omitempty"`
}

// CustomFieldValue is one custom-field row on a conversation.
type CustomFieldValue struct {
	ID    int    `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
	Text  string `json:"text,omitempty"`
}

// CustomerContactEntry is a {id, value, type} entry under embedded.emails / phones / socialProfiles.
type CustomerContactEntry struct {
	ID    int    `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}

// CustomerWebsiteEntry is a {id, value} entry under embedded.websites.
type CustomerWebsiteEntry struct {
	ID    int    `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}

// CustomerAddress is the embedded.address shape on a SupportCustomer.
type CustomerAddress struct {
	Street  string `json:"street,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`   // two-letter US state code
	Country string `json:"country,omitempty"` // ISO 3166-1 alpha-2
	Zip     string `json:"zip,omitempty"`
}

// CustomerEmbedded is the embedded shape on a SupportCustomer.
type CustomerEmbedded struct {
	Address        *CustomerAddress       `json:"address,omitempty"`
	Emails         []CustomerContactEntry `json:"emails,omitempty"`
	Phones         []CustomerContactEntry `json:"phones,omitempty"`
	SocialProfiles []CustomerContactEntry `json:"socialProfiles,omitempty"`
	Websites       []CustomerWebsiteEntry `json:"websites,omitempty"`
}

// SupportAttachment is one file attached to a support thread.
type SupportAttachment struct {
	ID       int    `json:"id,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FileURL  string `json:"fileUrl,omitempty"`
	Size     int    `json:"size,omitempty"` // bytes
}

// ThreadEmbedded is the embedded shape on a SupportThread.
type ThreadEmbedded struct {
	Attachments []SupportAttachment `json:"attachments,omitempty"`
}

// ConversationEmbedded is the embedded shape on a SupportConversation.
type ConversationEmbedded struct {
	Threads []SupportThread `json:"threads,omitempty"`
}

// SupportCustomer is the end-user profile attached to a support ticket.
type SupportCustomer struct {
	ID        int               `json:"id,omitempty"`
	FirstName string            `json:"firstName,omitempty"`
	LastName  string            `json:"lastName,omitempty"`
	Email     string            `json:"email,omitempty"`
	Company   string            `json:"company,omitempty"` // free-form, max 60
	JobTitle  string            `json:"jobTitle,omitempty"`
	PhotoType string            `json:"photoType,omitempty"`
	PhotoURL  string            `json:"photoUrl,omitempty"`
	Notes     string            `json:"notes,omitempty"`
	Type      string            `json:"type,omitempty"` // always "customer"
	CreatedAt string            `json:"createdAt,omitempty"`
	UpdatedAt string            `json:"updatedAt,omitempty"`
	Embedded  *CustomerEmbedded `json:"embedded,omitempty"`
}

// SupportThread is one message in a ticket conversation.
type SupportThread struct {
	ID            int              `json:"id,omitempty"`
	Status        TicketStatus     `json:"status"`
	State         string           `json:"state,omitempty"`
	Type          string           `json:"type,omitempty"` // "customer", "message", or "note"
	Body          string           `json:"body,omitempty"`
	Rating        int              `json:"rating,omitempty"`
	RatingComment string           `json:"ratingComment,omitempty"`
	OpenedAt      string           `json:"openedAt,omitempty"`
	CreatedAt     string           `json:"createdAt,omitempty"`
	Source        *TicketSource    `json:"source,omitempty"`
	Action        *TicketAction    `json:"action,omitempty"`
	CreatedBy     *TicketActor     `json:"createdBy,omitempty"`
	AssignedTo    *TicketActor     `json:"assignedTo,omitempty"`
	Customer      *SupportCustomer `json:"customer,omitempty"`
	To            []string         `json:"to,omitempty"`
	CC            []string         `json:"cc,omitempty"`
	BCC           []string         `json:"bcc,omitempty"`
	Embedded      *ThreadEmbedded  `json:"embedded,omitempty"`
}

// SupportConversation is a support ticket.
//
// Note: the wire field "number" is a ticket sequence number (1015, 2114, ...),
// NOT a phone number. We surface it as TicketNumber to avoid confusion with
// 10-digit TNs everywhere else in this API.
type SupportConversation struct {
	ID                   int                   `json:"id,omitempty"`
	TicketNumber         int                   `json:"number,omitempty"` // human-readable ticket sequence
	Status               TicketStatus          `json:"status"`
	State                string                `json:"state,omitempty"`
	Subject              string                `json:"subject,omitempty"`
	Preview              string                `json:"preview,omitempty"`
	Type                 string                `json:"type,omitempty"`
	MailboxID            int                   `json:"mailboxId,omitempty"`
	FolderID             int                   `json:"folderId,omitempty"`
	ThreadsCount         int                   `json:"threadsCount,omitempty"`
	ClosedBy             int                   `json:"closedBy,omitempty"`
	ClosedAt             string                `json:"closedAt,omitempty"`
	CreatedAt            string                `json:"createdAt,omitempty"`
	UpdatedAt            string                `json:"updatedAt,omitempty"`
	UserUpdatedAt        string                `json:"userUpdatedAt,omitempty"`
	CustomerWaitingSince map[string]any        `json:"customerWaitingSince,omitempty"`
	Source               *TicketSource         `json:"source,omitempty"`
	CreatedBy            *TicketActor          `json:"createdBy,omitempty"`
	Assignee             *TicketActor          `json:"assignee,omitempty"`
	ClosedByUser         *TicketActor          `json:"closedByUser,omitempty"`
	Customer             *SupportCustomer      `json:"customer,omitempty"`
	CC                   []string              `json:"cc,omitempty"`
	BCC                  []string              `json:"bcc,omitempty"`
	CustomFields         []CustomFieldValue    `json:"customFields,omitempty"`
	Embedded             *ConversationEmbedded `json:"embedded,omitempty"`
}

// TicketData is the response data for GET/POST /v2.2/support/tickets/{...}.
type TicketData struct {
	Ticket SupportConversation `json:"ticket"`
}

// TicketsListData is the response data for GET /v2.2/support/tickets.
type TicketsListData struct {
	Tickets []SupportConversation `json:"tickets"`
}

// TicketThreadsData is the response data for GET /v2.2/support/tickets/{id}/messages.
type TicketThreadsData struct {
	Messages []SupportThread `json:"messages"`
}

// TicketReplyData is the response data for POST /v2.2/support/tickets/{id}/replies.
type TicketReplyData struct {
	Message string `json:"message"` // always "Reply added"
}

// TicketUpdateData is the response data for PUT /v2.2/support/tickets/{id}.
type TicketUpdateData struct {
	ID     int    `json:"id,omitempty"`
	Status string `json:"status"` // outcome, e.g. "success"
}

// ------------------------------------------------------------------ methods ---

// List returns every ticket on the account.
func (s *SupportService) List(ctx context.Context) (*TicketsListData, error) {
	var out TicketsListData
	if err := s.c.t.request(ctx, "GET", "/v2.2/support/tickets", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create opens a new support ticket.
func (s *SupportService) Create(ctx context.Context, body TicketCreateRequest) (*TicketData, error) {
	var out TicketData
	if err := s.c.t.request(ctx, "POST", "/v2.2/support/tickets", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches one ticket by id.
func (s *SupportService) Get(ctx context.Context, id int) (*TicketData, error) {
	var out TicketData
	if err := s.c.t.request(ctx, "GET", "/v2.2/support/tickets/"+strconv.Itoa(id), nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a ticket's status.
func (s *SupportService) Update(ctx context.Context, id int, body TicketUpdateRequest) (*TicketUpdateData, error) {
	var out TicketUpdateData
	if err := s.c.t.request(ctx, "PUT", "/v2.2/support/tickets/"+strconv.Itoa(id), nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a ticket. Admin only. Returns nil on 204 No Content.
func (s *SupportService) Delete(ctx context.Context, id int) error {
	return s.c.t.request(ctx, "DELETE", "/v2.2/support/tickets/"+strconv.Itoa(id), nil, nil, nil, true)
}

// Messages returns every thread (message) on a ticket.
func (s *SupportService) Messages(ctx context.Context, id int) (*TicketThreadsData, error) {
	var out TicketThreadsData
	if err := s.c.t.request(ctx, "GET", "/v2.2/support/tickets/"+strconv.Itoa(id)+"/messages", nil, nil, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

// Reply adds a reply to a ticket.
func (s *SupportService) Reply(ctx context.Context, id int, body TicketReplyRequest) (*TicketReplyData, error) {
	var out TicketReplyData
	if err := s.c.t.request(ctx, "POST", "/v2.2/support/tickets/"+strconv.Itoa(id)+"/replies", nil, body, &out, true); err != nil {
		return nil, err
	}
	return &out, nil
}

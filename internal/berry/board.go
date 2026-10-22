package berry

import (
	"context"
	"net/http"
	"net/url"
)

// Board is the subset of the BerryTicket board (/api/state) the app uses:
// every task, the review inbox, people and projects.
type Board struct {
	User         BoardUser     `json:"user"`
	Tickets      []BoardTicket `json:"tickets"`
	Members      []Member      `json:"members"`
	Projects     []Project     `json:"projects"`
	Proposals    []Proposal    `json:"proposals"`
	Calendar     Calendar      `json:"calendar"`
	DesignReview DesignReview  `json:"design_review"`
	Revision     string        `json:"revision"`
}

type BoardUser struct {
	Role string `json:"role"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BoardTicket is a task row as the board API returns it.
type BoardTicket struct {
	ID          string  `json:"id"`
	Number      int     `json:"number"`
	ProjectID   string  `json:"project_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`   // todo | in_progress | blocked | done
	Priority    string  `json:"priority"` // low | normal | high | urgent
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
	SourceURL   string  `json:"source_url"`
	Area        string  `json:"area"` // "" | engineering | design | marketing
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// Proposal is a task the assistant found in a conversation and wants confirmed.
type Proposal struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"` // create | update
	ProjectID string          `json:"project_id"`
	TicketID  *string         `json:"ticket_id"`
	Payload   ProposalPayload `json:"payload"`
	Reason    string          `json:"reason"`
	CreatedAt string          `json:"created_at"`
}

type ProposalPayload struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	AssigneeID  *string  `json:"assignee_id"`
	DueDate     *string  `json:"due_date"`
	SourceURL   string   `json:"source_url"`
	Routing     *Routing `json:"_routing,omitempty"`
}

// Routing.Confirmed == false means the project still has to be chosen before approving.
type Routing struct {
	Confirmed *bool  `json:"confirmed"`
	Reason    string `json:"reason"`
}

type Calendar struct {
	Proposals []MeetingProposal `json:"proposals"`
}

type MeetingProposal struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	ProjectID string         `json:"project_id"`
	Payload   MeetingPayload `json:"payload"`
	Reason    string         `json:"reason"`
}

type MeetingPayload struct {
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Location  string   `json:"location"`
	SourceURL string   `json:"source_url"`
	Status    string   `json:"status"`
	Routing   *Routing `json:"_routing,omitempty"`
}

type DesignReview struct {
	Questions []DesignQuestion `json:"questions"`
}

type DesignQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Reason   string `json:"reason"`
	URL      string `json:"url"`
}

func (c *Client) Board(ctx context.Context) (Board, error) {
	var b Board
	err := c.do(ctx, http.MethodGet, "/api/state", nil, true, &b)
	return b, err
}

func (c *Client) BoardRevision(ctx context.Context) (string, error) {
	var r struct {
		Revision string `json:"revision"`
	}
	err := c.do(ctx, http.MethodGet, "/api/updates", nil, true, &r)
	return r.Revision, err
}

// UpdateTicket changes fields of any task; expected is the updated_at the change is based on.
func (c *Client) UpdateTicket(ctx context.Context, id, expected string, changes map[string]any) (BoardTicket, error) {
	body := map[string]any{"expected_updated_at": expected}
	for k, v := range changes {
		body[k] = v
	}
	var out BoardTicket
	err := c.do(ctx, http.MethodPatch, "/api/tickets/"+url.PathEscape(id), body, true, &out)
	return out, err
}

// ApproveProposal adds the proposed task; projectID is required only when routing is unconfirmed.
func (c *Client) ApproveProposal(ctx context.Context, id, projectID string) error {
	body := map[string]any{}
	if projectID != "" {
		body["project_id"] = projectID
	}
	return c.do(ctx, http.MethodPost, "/api/proposals/"+url.PathEscape(id)+"/approve", body, true, nil)
}

func (c *Client) DismissProposal(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/proposals/"+url.PathEscape(id)+"/dismiss", map[string]any{}, true, nil)
}

func (c *Client) ApproveMeeting(ctx context.Context, id, projectID string) error {
	body := map[string]any{}
	if projectID != "" {
		body["project_id"] = projectID
	}
	return c.do(ctx, http.MethodPost, "/api/meeting-proposals/"+url.PathEscape(id)+"/approve", body, true, nil)
}

func (c *Client) DismissMeeting(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/meeting-proposals/"+url.PathEscape(id)+"/dismiss", map[string]any{}, true, nil)
}

func (c *Client) DismissDesignQuestion(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/design-review/questions/"+url.PathEscape(id)+"/dismiss", map[string]any{}, true, nil)
}

// Tasks converts board rows to the Ticket shape the app shows, with the project attached.
func (b Board) Tasks() []Ticket {
	projects := map[string]*Project{}
	for i := range b.Projects {
		projects[b.Projects[i].ID] = &b.Projects[i]
	}
	out := make([]Ticket, 0, len(b.Tickets))
	for _, t := range b.Tickets {
		out = append(out, t.Ticket(projects[t.ProjectID]))
	}
	return out
}

func (t BoardTicket) Ticket(p *Project) Ticket {
	return Ticket{ID: t.ID, Number: t.Number, Title: t.Title, Description: t.Description, Status: t.Status,
		Priority: t.Priority, DueDate: t.DueDate, SourceURL: t.SourceURL, UpdatedAt: t.UpdatedAt, Project: p,
		AssigneeID: t.AssigneeID, Area: t.Area, CreatedAt: t.CreatedAt}
}

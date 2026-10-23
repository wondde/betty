package berry

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// Board is the subset of the BerryTicket board (/api/state) the app uses:
// every task, the review inbox, people and projects.
type Board struct {
	User            BoardUser               `json:"user"`
	Tickets         []BoardTicket           `json:"tickets"`
	Members         []Member                `json:"members"`
	Projects        []Project               `json:"projects"`
	Proposals       []Proposal              `json:"proposals"`
	Calendar        Calendar                `json:"calendar"`
	DesignReview    DesignReview            `json:"design_review"`
	GitHub          GitHub                  `json:"github"`
	Settings        map[string]bool         `json:"settings"`
	DiscordChannels []DiscordChannel        `json:"discord_channels"`
	ProjectChannels []ProjectChannel        `json:"project_channels"`
	Progress        map[string]TaskProgress `json:"progress"`     // by ticket id
	DesignLinks     map[string][]DesignLink `json:"design_links"` // by ticket id
	Revision        string                  `json:"revision"`
}

// TaskProgress is what the automations confirmed finished and still open for a task.
type TaskProgress struct {
	Finished  []ProgressStep `json:"finished"`
	Remaining []ProgressStep `json:"remaining"`
	UpdatedAt string         `json:"updated_at"`
}

type ProgressStep struct {
	Text  string `json:"text"`
	Quote string `json:"quote"`
	URL   string `json:"url"`
}

// DesignLink is a Figma screen or comment the design review tied to a task.
type DesignLink struct {
	URL    string `json:"url"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
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
	Priority    string   `json:"priority"`
	Area        string   `json:"area"`
	SourceURL   string   `json:"source_url"`
	Routing     *Routing `json:"_routing,omitempty"`
}

// Routing.Confirmed == false means the project still has to be chosen before approving.
type Routing struct {
	Confirmed *bool  `json:"confirmed"`
	Reason    string `json:"reason"`
}

type Calendar struct {
	Meetings  []Meeting         `json:"meetings"`
	Proposals []MeetingProposal `json:"proposals"`
}

// Meeting is a calendar entry (Asia/Tokyo dates and times).
type Meeting struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	Title        string   `json:"title"`
	Date         string   `json:"date"`       // YYYY-MM-DD
	StartTime    string   `json:"start_time"` // HH:MM or ""
	EndTime      string   `json:"end_time"`
	Participants []string `json:"participants"`
	Location     string   `json:"location"`
	Description  string   `json:"description"`
	SourceURL    string   `json:"source_url"`
	Status       string   `json:"status"` // scheduled | cancelled
	UpdatedAt    string   `json:"updated_at"`
}

// MeetingInput creates or edits a meeting; ExpectedUpdatedAt is required when editing.
type MeetingInput struct {
	ProjectID         string   `json:"project_id,omitempty"`
	Title             string   `json:"title"`
	Date              string   `json:"date"`
	StartTime         string   `json:"start_time"`
	EndTime           string   `json:"end_time"`
	Participants      []string `json:"participants"`
	Location          string   `json:"location"`
	Description       string   `json:"description"`
	Status            string   `json:"status,omitempty"`
	ExpectedUpdatedAt string   `json:"expected_updated_at,omitempty"`
}

// History is one page of what the assistant changed on its own (linked, progressed, completed).
type History struct {
	Items      []Change `json:"items"`
	Total      int      `json:"total"`
	NextCursor *string  `json:"next_cursor"`
}

type Change struct {
	ID          string       `json:"id"`
	Source      string       `json:"source"`
	TicketID    *string      `json:"ticket_id"`
	Kind        string       `json:"kind"` // linked | progressed | completed
	CreatedAt   string       `json:"created_at"`
	UndoneAt    string       `json:"undone_at"`
	UndoneBy    string       `json:"undone_by"`
	Ticket      ChangeTicket `json:"ticket"`
	Reason      string       `json:"reason"`
	Evidence    []Evidence   `json:"evidence"`
	BeforeState *string      `json:"before_status"`
	AfterState  *string      `json:"after_status"`
	Links       []ChangeLink `json:"links"`
	CanUndo     bool         `json:"can_undo"`
	UndoReason  string       `json:"undo_reason"`
}

type ChangeTicket struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	ProjectID string `json:"project_id"`
}

type Evidence struct {
	Label string `json:"label"`
	Quote string `json:"quote"`
	URL   string `json:"url"`
}

type ChangeLink struct {
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Number     int    `json:"number"`
	Repository string `json:"repository"`
	URL        string `json:"url"`
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
	b.colorProjects()
	return b, err
}

// Projects are created with a shared default colour (the server and discovery
// use different purples), so every project looks the same. A colour that only
// one project has was picked on purpose and is kept; shared or default colours
// get the next palette colour no other project uses. "Project undecided" is grey.
const undecidedProject = "berry-project-undecided"

var defaultProjectColors = map[string]bool{"#7564d9": true, "#6175ba": true}

var projectPalette = []string{"#0a84ff", "#30a46c", "#f76b15", "#d6409f", "#12a594", "#e5484d", "#8e4ec6", "#c99a06", "#3e63dd", "#978365"}

func (b *Board) colorProjects() {
	uses := map[string]int{}
	for _, p := range b.Projects {
		uses[strings.ToLower(p.Color)]++
	}
	keep := func(p Project) bool {
		c := strings.ToLower(p.Color)
		return c != "" && uses[c] == 1 && !defaultProjectColors[c]
	}
	taken := map[string]bool{}
	for _, p := range b.Projects {
		if p.ID != undecidedProject && keep(p) {
			taken[strings.ToLower(p.Color)] = true
		}
	}
	next := 0
	for i := range b.Projects {
		p := &b.Projects[i]
		if p.ID == undecidedProject {
			p.Color = "#8b8d98"
			continue
		}
		if keep(*p) {
			continue
		}
		for n := 0; n < len(projectPalette) && taken[projectPalette[next%len(projectPalette)]]; n++ {
			next++
		}
		p.Color = projectPalette[next%len(projectPalette)]
		taken[p.Color] = true
		next++
	}
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

func (c *Client) SaveMeeting(ctx context.Context, id string, in MeetingInput) error {
	if id == "" {
		return c.do(ctx, http.MethodPost, "/api/meetings", in, true, nil)
	}
	return c.do(ctx, http.MethodPatch, "/api/meetings/"+url.PathEscape(id), in, true, nil)
}

// History lists automatic changes, newest first; cursor continues from the previous page.
func (c *Client) History(ctx context.Context, cursor, status, ticketID string) (History, error) {
	q := url.Values{"limit": {"30"}}
	if ticketID != "" {
		q.Set("ticket_id", ticketID)
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if status != "" {
		q.Set("status", status)
	}
	var h History
	err := c.do(ctx, http.MethodGet, "/api/automation/history?"+q.Encode(), nil, true, &h)
	return h, err
}

// Capture hands a pasted conversation to the extractor; what it finds shows up
// in the review inbox (or is applied directly when automatic tasks are on).
func (c *Client) Capture(ctx context.Context, content, projectID string) error {
	return c.do(ctx, http.MethodPost, "/api/sources", map[string]any{"content": content, "project_id": projectID}, true, nil)
}

func (c *Client) Undo(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/automation/history/"+url.PathEscape(id)+"/undo", map[string]any{}, true, nil)
}

// ---------- GitHub (read-only sync on the server) ----------

type GitHub struct {
	Repos       []GitHubRepo       `json:"repos"`
	Items       []GitHubItem       `json:"items"`
	Suggestions []GitHubSuggestion `json:"suggestions"`
	Identities  []GitHubIdentity   `json:"identities"`
}

type GitHubRepo struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	ProjectID string `json:"project_id"`
	SyncedAt  string `json:"synced_at"`
	Error     string `json:"error"`
}

type GitHubItem struct {
	RepoID      string   `json:"repo_id"`
	Number      int      `json:"number"`
	Kind        string   `json:"kind"`  // issue | pr
	State       string   `json:"state"` // open | closed
	Stage       string   `json:"stage"` // open | closed | not_planned | draft | review_requested | merged
	Title       string   `json:"title"`
	StateReason string   `json:"state_reason"`
	Author      string   `json:"author"`
	Assignees   []string `json:"assignees"`
	Reviewers   []string `json:"reviewers"`
	UpdatedAt   string   `json:"updated_at"`
	URL         string   `json:"url"`
	ProjectID   string   `json:"project_id"`
	TicketIDs   []string `json:"ticket_ids"`
}

// GitHubSuggestion: the assistant thinks this task and GitHub item are the same work.
type GitHubSuggestion struct {
	TicketID string `json:"ticket_id"`
	RepoID   string `json:"repo_id"`
	Number   int    `json:"number"`
	Reason   string `json:"reason"`
}

type GitHubIdentity struct {
	Login    string `json:"login"`
	MemberID string `json:"member_id"`
}

type DiscordChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type ProjectChannel struct {
	ChannelID string `json:"channel_id"`
	ProjectID string `json:"project_id"`
}

// GitHubLink links (or with remove, unlinks) a task and a GitHub item; versions guard both sides.
func (c *Client) GitHubLink(ctx context.Context, ticketID, ticketVersion, repoID string, number int, itemVersion string, allowShared, remove bool) error {
	method := http.MethodPost
	if remove {
		method = http.MethodDelete
	}
	body := map[string]any{"repo_id": repoID, "number": number, "expected_updated_at": ticketVersion, "expected_github_updated_at": itemVersion, "allow_shared": allowShared}
	return c.do(ctx, method, "/api/tickets/"+url.PathEscape(ticketID)+"/github-links", body, true, nil)
}

// GitHubReject records that a suggested pair is not the same work, so it is not suggested again.
func (c *Client) GitHubReject(ctx context.Context, ticketID, ticketVersion, repoID string, number int) error {
	body := map[string]any{"repo_id": repoID, "number": number, "expected_updated_at": ticketVersion}
	return c.do(ctx, http.MethodPost, "/api/tickets/"+url.PathEscape(ticketID)+"/github-links/reject", body, true, nil)
}

func (c *Client) AddRepo(ctx context.Context, fullName, projectID string) error {
	return c.do(ctx, http.MethodPost, "/api/github/repos", map[string]any{"full_name": fullName, "project_id": projectID}, true, nil)
}

func (c *Client) RemoveRepo(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/github/repos/"+url.PathEscape(id), map[string]any{}, true, nil)
}

// ---------- team settings ----------

func (c *Client) SaveProject(ctx context.Context, id, name string, channelIDs []string) error {
	body := map[string]any{"name": name, "channel_ids": channelIDs}
	if id == "" {
		return c.do(ctx, http.MethodPost, "/api/projects", body, true, nil)
	}
	return c.do(ctx, http.MethodPatch, "/api/projects/"+url.PathEscape(id), body, true, nil)
}

type MemberInput struct {
	Name        string `json:"name"`
	DiscordID   string `json:"discord_id"`
	ChannelID   string `json:"channel_id"`
	GitHubLogin string `json:"github_login"`
}

func (c *Client) SaveMember(ctx context.Context, id string, in MemberInput) error {
	if id == "" {
		return c.do(ctx, http.MethodPost, "/api/members", in, true, nil)
	}
	return c.do(ctx, http.MethodPatch, "/api/members/"+url.PathEscape(id), in, true, nil)
}

// SetSetting flips one automation switch (see the board's /api/settings for the allowed keys).
func (c *Client) SetSetting(ctx context.Context, key string, on bool) error {
	return c.do(ctx, http.MethodPost, "/api/settings", map[string]any{key: on}, true, nil)
}

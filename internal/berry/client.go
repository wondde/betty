// Package berry is a small client for the BerryTicket widget API.
// A device token only reaches /api/widget/* and /api/devices/*, and only the
// signed-in member's own tasks can be changed.
package berry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Project struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Member struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DiscordID string `json:"discord_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"` // personal work channel
	Area      string `json:"area,omitempty"`       // engineering | design | marketing | ""
	IsAdmin   bool   `json:"is_admin,omitempty"`
}

type Ticket struct {
	ID          string   `json:"id"`
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // todo | in_progress | blocked | done
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"due_date"`
	SourceURL   string   `json:"source_url"`
	UpdatedAt   string   `json:"updated_at"`
	Project     *Project `json:"project"`
	AssigneeID  *string  `json:"assignee_id,omitempty"`
	Area        string   `json:"area,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

type State struct {
	Member   Member    `json:"member"`
	Tickets  []Ticket  `json:"tickets"`
	Members  []Member  `json:"members"`
	Projects []Project `json:"projects"`
	Revision string    `json:"revision"`
}

// CreateInput adds a task. AssigneeID nil = my own task.
type CreateInput struct {
	Title      string  `json:"title"`
	AssigneeID *string `json:"assignee_id,omitempty"`
	ProjectID  string  `json:"project_id,omitempty"`
	DueDate    string  `json:"due_date,omitempty"`
}

// APIError carries the server's (Korean) message and HTTP status.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrOffline      = errors.New("offline")
)

type Client struct {
	BaseURL string
	Token   func() string
	HTTP    *http.Client
}

func New(baseURL string, token func() string) *Client {
	return &Client{BaseURL: baseURL, Token: token, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, body any, auth bool, out any) error {
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.Token())
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized && auth {
		return ErrUnauthorized
	}
	if res.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		if e.Error == "" {
			e.Error = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		return &APIError{Status: res.StatusCode, Message: e.Error}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) State(ctx context.Context) (State, error) {
	var s State
	err := c.do(ctx, http.MethodGet, "/api/widget/state", nil, true, &s)
	return s, err
}

func (c *Client) Revision(ctx context.Context) (string, error) {
	var r struct {
		Revision string `json:"revision"`
	}
	err := c.do(ctx, http.MethodGet, "/api/widget/updates", nil, true, &r)
	return r.Revision, err
}

func (c *Client) SetStatus(ctx context.Context, t Ticket, status string) (Ticket, error) {
	var out Ticket
	err := c.do(ctx, http.MethodPatch, "/api/widget/tickets/"+url.PathEscape(t.ID),
		map[string]string{"status": status, "expected_updated_at": t.UpdatedAt}, true, &out)
	return out, err
}

func (c *Client) Create(ctx context.Context, in CreateInput) (Ticket, error) {
	var out Ticket
	err := c.do(ctx, http.MethodPost, "/api/widget/tickets", in, true, &out)
	return out, err
}

// Exchange trades the one-time sign-in code for a device token.
func (c *Client) Exchange(ctx context.Context, code, verifier string) (string, Member, error) {
	var out struct {
		Token  string `json:"token"`
		Member Member `json:"member"`
	}
	err := c.do(ctx, http.MethodPost, "/api/devices/exchange", map[string]string{"code": code, "verifier": verifier}, false, &out)
	return out.Token, out.Member, err
}

// SignOut revokes this device's token on the server.
func (c *Client) SignOut(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/api/devices/current", nil, true, nil)
}

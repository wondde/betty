package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/wondde/betty/internal/berry"
	"github.com/wondde/betty/internal/prefs"
	"github.com/wondde/betty/internal/signin"
	"github.com/wondde/betty/internal/vault"
)

const pollEvery = 15 * time.Second

// View is everything the windows need to draw. It is pushed with the "betty:view" event
// whenever it changes, so every window (menu bar, quick add) stays in sync.
type View struct {
	SignedIn  bool            `json:"signedIn"`
	SigningIn bool            `json:"signingIn"`
	Member    *berry.Member   `json:"member"`
	Tickets   []berry.Ticket  `json:"tickets"`
	Members   []berry.Member  `json:"members"`
	Projects  []berry.Project `json:"projects"`
	New       []string        `json:"new"`
	Order     []string        `json:"order"`
	HintDone  bool            `json:"hintDone"`
	Offline   bool            `json:"offline"`
	SyncedAt  string          `json:"syncedAt"`
	Server    string          `json:"server"`
}

// Betty is the one service the frontend talks to. Network, Keychain, notifications and
// polling all live here so the UI only renders View and calls these methods.
type Betty struct {
	prefs  *prefs.Store
	client *berry.Client
	notify *notifications.NotificationService

	mu        sync.Mutex
	revision  string
	offline   bool
	syncedAt  time.Time
	signingIn context.CancelFunc

	onCount func(open int) // menu bar label
}

func NewBetty(p *prefs.Store, ns *notifications.NotificationService) *Betty {
	b := &Betty{prefs: p, notify: ns}
	b.client = berry.New(p.Server(), vault.Token)
	return b
}

func (b *Betty) ServiceName() string { return "Betty" }

func (b *Betty) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	go b.poll(ctx)
	return nil
}

// ---------- view ----------

func (b *Betty) View() View {
	d := b.prefs.Get()
	b.mu.Lock()
	v := View{SignedIn: vault.Token() != "", SigningIn: b.signingIn != nil, Offline: b.offline, Server: b.prefs.Server(),
		Order: d.Order, HintDone: d.HintDone}
	if !b.syncedAt.IsZero() {
		v.SyncedAt = b.syncedAt.Format(time.RFC3339)
	}
	b.mu.Unlock()
	if s := d.Snapshot; s != nil && v.SignedIn {
		member := s.Member
		v.Member, v.Tickets, v.Members, v.Projects = &member, s.Tickets, s.Members, s.Projects
		seen := toSet(d.Seen)
		for _, t := range s.Tickets {
			if t.Status != "done" && !seen[t.ID] {
				v.New = append(v.New, t.ID)
			}
		}
	}
	if v.Tickets == nil {
		v.Tickets = []berry.Ticket{}
	}
	return v
}

func (b *Betty) emit() {
	v := b.View()
	if app := application.Get(); app != nil {
		app.Event.Emit("betty:view", v)
	}
	if b.onCount != nil {
		open := 0
		for _, t := range v.Tickets {
			if t.Status != "done" {
				open++
			}
		}
		b.onCount(open)
	}
}

// ---------- sync ----------

func (b *Betty) poll(ctx context.Context) {
	_ = b.Refresh()
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if vault.Token() == "" {
				continue
			}
			rev, err := b.client.Revision(ctx)
			b.mu.Lock()
			same := err == nil && rev == b.revision
			b.mu.Unlock()
			if err != nil {
				b.handle(err)
				continue
			}
			if !same {
				_ = b.Refresh()
			}
		}
	}
}

// Refresh reloads my tasks now.
func (b *Betty) Refresh() error {
	if vault.Token() == "" {
		b.emit()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	state, err := b.client.State(ctx)
	if err != nil {
		return b.handle(err)
	}
	b.mu.Lock()
	b.revision, b.offline, b.syncedAt = state.Revision, false, time.Now()
	b.mu.Unlock()

	var announce []berry.Ticket
	b.prefs.Update(func(d *prefs.Data) {
		first := d.Snapshot == nil && len(d.Seen) == 0
		var open []string
		for _, t := range state.Tickets {
			if t.Status != "done" {
				open = append(open, t.ID)
			}
		}
		if first {
			// Everything on the list at first sign-in is not "new".
			d.Seen = prefs.Remember(d.Seen, open...)
			d.Notified = prefs.Remember(d.Notified, open...)
		} else {
			notified := toSet(d.Notified)
			for _, t := range state.Tickets {
				if t.Status != "done" && !notified[t.ID] {
					announce = append(announce, t)
				}
			}
			for _, t := range announce {
				d.Notified = prefs.Remember(d.Notified, t.ID)
			}
		}
		snapshot := state
		d.Snapshot = &snapshot
	})
	b.announce(announce)
	b.emit()
	return nil
}

func (b *Betty) announce(tickets []berry.Ticket) {
	if len(tickets) == 0 || b.notify == nil {
		return
	}
	title := "새 업무가 배정됐어요"
	if len(tickets) > 1 {
		title = fmt.Sprintf("새 업무 %d건이 배정됐어요", len(tickets))
	}
	var lines []string
	for _, t := range tickets {
		lines = append(lines, fmt.Sprintf("BT-%d %s", t.Number, t.Title))
	}
	if ok, _ := b.notify.CheckNotificationAuthorization(); !ok {
		if ok, _ = b.notify.RequestNotificationAuthorization(); !ok {
			return
		}
	}
	err := b.notify.SendNotification(notifications.NotificationOptions{
		ID: "betty-new-" + tickets[0].ID, Title: title, Body: strings.Join(lines, "\n"),
		Data: map[string]interface{}{"open": "main"},
	})
	if err != nil {
		log.Printf("notification: %v", err)
	}
}

// handle turns transport errors into state: offline flag, or sign-out on a revoked token.
func (b *Betty) handle(err error) error {
	switch {
	case errors.Is(err, berry.ErrUnauthorized):
		_ = vault.ClearToken()
		b.prefs.Update(func(d *prefs.Data) { d.Snapshot = nil })
		b.emit()
		return errors.New("이 기기의 로그인이 해제됐어요. 다시 로그인해 주세요")
	case errors.Is(err, berry.ErrOffline):
		b.mu.Lock()
		b.offline = true
		b.mu.Unlock()
		b.emit()
		return errors.New("서버에 연결하지 못했어요")
	}
	return err
}

// ---------- tasks ----------

func (b *Betty) ticket(id string) (berry.Ticket, bool) {
	if s := b.prefs.Get().Snapshot; s != nil {
		for _, t := range s.Tickets {
			if t.ID == id {
				return t, true
			}
		}
	}
	return berry.Ticket{}, false
}

// SetStatus changes one of my tasks: todo | in_progress | blocked | done.
func (b *Betty) SetStatus(id, status string) (berry.Ticket, error) {
	t, ok := b.ticket(id)
	if !ok {
		return berry.Ticket{}, errors.New("업무를 찾지 못했어요. 새로고침해 주세요")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	updated, err := b.client.SetStatus(ctx, t, status)
	if err != nil {
		var api *berry.APIError
		if errors.As(err, &api) && api.Status == 409 {
			_ = b.Refresh()
		}
		return berry.Ticket{}, b.handle(err)
	}
	b.prefs.Update(func(d *prefs.Data) {
		d.Seen = prefs.Remember(d.Seen, id)
		if d.Snapshot == nil {
			return
		}
		for i := range d.Snapshot.Tickets {
			if d.Snapshot.Tickets[i].ID == id {
				d.Snapshot.Tickets[i] = updated
			}
		}
	})
	b.emit()
	return updated, nil
}

// Create adds a task (mine, or a teammate's when AssigneeID is set).
func (b *Betty) Create(in berry.CreateInput) (berry.Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	created, err := b.client.Create(ctx, in)
	if err != nil {
		return berry.Ticket{}, b.handle(err)
	}
	mine := in.AssigneeID == nil
	if s := b.prefs.Get().Snapshot; s != nil && in.AssigneeID != nil && *in.AssigneeID == s.Member.ID {
		mine = true
	}
	b.prefs.Update(func(d *prefs.Data) {
		d.Seen = prefs.Remember(d.Seen, created.ID) // my own task is never "new"
		d.Notified = prefs.Remember(d.Notified, created.ID)
		if mine && d.Snapshot != nil {
			d.Snapshot.Tickets = append([]berry.Ticket{created}, d.Snapshot.Tickets...)
		}
	})
	b.emit()
	return created, nil
}

func (b *Betty) Acknowledge(id string) {
	b.prefs.Update(func(d *prefs.Data) { d.Seen = prefs.Remember(d.Seen, id) })
	b.emit()
}

func (b *Betty) SaveOrder(ids []string) {
	b.prefs.Update(func(d *prefs.Data) { d.Order = ids })
}

func (b *Betty) DismissHint() {
	b.prefs.Update(func(d *prefs.Data) { d.HintDone = true })
	b.emit()
}

// ---------- account ----------

// SignIn opens the browser for Discord and waits for the loopback callback.
func (b *Betty) SignIn() error {
	b.mu.Lock()
	if b.signingIn != nil {
		b.signingIn()
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.signingIn = cancel
	b.mu.Unlock()
	b.emit()
	defer func() {
		b.mu.Lock()
		b.signingIn = nil
		b.mu.Unlock()
		cancel()
		b.emit()
	}()

	host, _ := os.Hostname()
	device := "Betty · " + strings.TrimSuffix(host, ".local")
	res, err := signin.Run(ctx, b.prefs.Server(), device, func(u string) error {
		return application.Get().Browser.OpenURL(u)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	xctx, xcancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer xcancel()
	token, _, err := b.client.Exchange(xctx, res.Code, res.Verifier)
	if err != nil {
		return err
	}
	if err := vault.SetToken(token); err != nil {
		return fmt.Errorf("키체인에 저장하지 못했어요: %w", err)
	}
	b.prefs.Update(func(d *prefs.Data) { d.Snapshot = nil; d.Seen = nil; d.Notified = nil })
	go func() { _ = b.Refresh() }()
	return nil
}

func (b *Betty) CancelSignIn() {
	b.mu.Lock()
	if b.signingIn != nil {
		b.signingIn()
	}
	b.mu.Unlock()
}

// SignOut revokes this device on the server (best effort) and forgets the token.
func (b *Betty) SignOut() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = b.client.SignOut(ctx)
	_ = vault.ClearToken()
	b.prefs.Update(func(d *prefs.Data) { d.Snapshot = nil })
	b.emit()
	return nil
}

// ---------- windows & links ----------

// OpenURL opens an http(s) link in the default browser.
func (b *Betty) OpenURL(link string) error {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("열 수 없는 주소예요")
	}
	return application.Get().Browser.OpenURL(link)
}

func (b *Betty) OpenBoard() error { return b.OpenURL(b.prefs.Server() + "/") }

func (b *Betty) HideWindow(name string) {
	if w, ok := application.Get().Window.GetByName(name); ok {
		w.Hide()
	}
}

func toSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, id := range list {
		m[id] = true
	}
	return m
}

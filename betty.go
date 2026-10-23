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

// version is shown in Settings and "Betty에 관하여"; keep in sync with build/config.yml.
const version = "0.7.0"

// View is everything the windows need to draw. It is pushed with the "betty:view" event
// whenever it changes, so every window (menu bar, quick add) stays in sync.
type View struct {
	SignedIn  bool            `json:"signedIn"`
	SigningIn bool            `json:"signingIn"`
	Member    *berry.Member   `json:"member"`
	Tickets   []berry.Ticket  `json:"tickets"` // every task on the board; mine have assignee_id == member.id
	Members   []berry.Member  `json:"members"`
	Projects  []berry.Project `json:"projects"`
	// The review inbox ("확인해 주세요")
	Proposals        []berry.Proposal              `json:"proposals"`
	MeetingProposals []berry.MeetingProposal       `json:"meetingProposals"`
	DesignQuestions  []berry.DesignQuestion        `json:"designQuestions"`
	Meetings         []berry.Meeting               `json:"meetings"`
	GitHub           berry.GitHub                  `json:"github"`
	Settings         map[string]bool               `json:"settings"`
	DiscordChannels  []berry.DiscordChannel        `json:"discordChannels"`
	ProjectChannels  []berry.ProjectChannel        `json:"projectChannels"`
	Progress         map[string]berry.TaskProgress `json:"progress"`
	DesignLinks      map[string][]berry.DesignLink `json:"designLinks"`
	Trash            []berry.TrashItem             `json:"trash"`
	New              []string                      `json:"new"` // my open tasks not looked at yet
	Order            []string                      `json:"order"`
	HintDone         bool                          `json:"hintDone"`
	Onboarded        bool                          `json:"onboarded"`
	Version          string                        `json:"version"`
	Offline          bool                          `json:"offline"`
	SyncedAt         string                        `json:"syncedAt"`
	Server           string                        `json:"server"`
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

	onCount     func(open int) // menu bar label
	onShowTasks func()         // open the task window centred (set by main)
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
		Order: d.Order, HintDone: d.HintDone, Onboarded: d.Onboarded, Version: version}
	if !b.syncedAt.IsZero() {
		v.SyncedAt = b.syncedAt.Format(time.RFC3339)
	}
	b.mu.Unlock()
	if s := d.Snapshot; s != nil && v.SignedIn {
		v.Member = &berry.Member{ID: s.User.ID, Name: s.User.Name}
		v.Tickets, v.Members, v.Projects = s.Tasks(), s.Members, s.Projects
		v.Proposals, v.MeetingProposals, v.DesignQuestions = s.Proposals, s.Calendar.Proposals, s.DesignReview.Questions
		v.Meetings = s.Calendar.Meetings
		v.GitHub, v.Settings, v.DiscordChannels, v.ProjectChannels = s.GitHub, s.Settings, s.DiscordChannels, s.ProjectChannels
		v.Progress, v.DesignLinks, v.Trash = s.Progress, s.DesignLinks, s.Trash
		seen := toSet(d.Seen)
		for _, t := range mine(s) {
			if !seen[t.ID] {
				v.New = append(v.New, t.ID)
			}
		}
	}
	if v.Tickets == nil {
		v.Tickets = []berry.Ticket{}
	}
	return v
}

// mine lists my open tasks on the board.
func mine(s *berry.Board) []berry.BoardTicket {
	var out []berry.BoardTicket
	for _, t := range s.Tickets {
		if t.Status != "done" && t.AssigneeID != nil && *t.AssigneeID == s.User.ID {
			out = append(out, t)
		}
	}
	return out
}

func (b *Betty) emit() {
	v := b.View()
	if app := application.Get(); app != nil {
		app.Event.Emit("betty:view", v)
	}
	if b.onCount != nil {
		open := 0
		if s := b.prefs.Get().Snapshot; s != nil && v.SignedIn {
			open = len(mine(s))
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
			rev, err := b.client.BoardRevision(ctx)
			b.mu.Lock()
			same := err == nil && rev == b.revision
			b.mu.Unlock()
			if err != nil {
				if errors.Is(err, berry.ErrUnauthorized) {
					_ = b.Refresh() // decides between "server too old" and a revoked token
				} else {
					b.handle(err)
				}
				continue
			}
			if !same {
				_ = b.Refresh()
			}
		}
	}
}

// Refresh reloads the board now.
func (b *Betty) Refresh() error {
	if vault.Token() == "" {
		b.emit()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	board, err := b.client.Board(ctx)
	if errors.Is(err, berry.ErrUnauthorized) {
		// A server that predates board access for device tokens answers 401 here while the
		// token is still fine; only a token the widget API also rejects really signs us out.
		if _, werr := b.client.State(ctx); werr == nil {
			return errors.New("BerryTicket 서버 업데이트가 필요해요. 관리자에게 알려 주세요")
		}
	}
	if err != nil {
		return b.handle(err)
	}
	if board.User.ID == "" {
		return errors.New("팀원 계정으로 로그인해 주세요")
	}
	b.mu.Lock()
	b.revision, b.offline, b.syncedAt = board.Revision, false, time.Now()
	b.mu.Unlock()

	var announce []berry.BoardTicket
	b.prefs.Update(func(d *prefs.Data) {
		first := d.Snapshot == nil && len(d.Seen) == 0
		var open []string
		for _, t := range mine(&board) {
			open = append(open, t.ID)
		}
		if first {
			// Everything on the list at first sign-in is not "new".
			d.Seen = prefs.Remember(d.Seen, open...)
			d.Notified = prefs.Remember(d.Notified, open...)
		} else {
			notified := toSet(d.Notified)
			for _, t := range mine(&board) {
				if !notified[t.ID] {
					announce = append(announce, t)
				}
			}
			for _, t := range announce {
				d.Notified = prefs.Remember(d.Notified, t.ID)
			}
		}
		snapshot := board
		d.Snapshot = &snapshot
	})
	b.announce(announce)
	b.emit()
	return nil
}

func (b *Betty) announce(tickets []berry.BoardTicket) {
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

func (b *Betty) boardTicket(id string) (berry.BoardTicket, bool) {
	if s := b.prefs.Get().Snapshot; s != nil {
		for _, t := range s.Tickets {
			if t.ID == id {
				return t, true
			}
		}
	}
	return berry.BoardTicket{}, false
}

// Update changes any task's fields (title, description, status, priority, assignee_id,
// due_date, project_id, area), based on the version last loaded. A conflict reloads the board.
func (b *Betty) Update(id string, changes map[string]any) (berry.Ticket, error) {
	t, ok := b.boardTicket(id)
	if !ok {
		return berry.Ticket{}, errors.New("업무를 찾지 못했어요. 새로고침해 주세요")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	updated, err := b.client.UpdateTicket(ctx, id, t.UpdatedAt, changes)
	if err != nil {
		var api *berry.APIError
		if errors.As(err, &api) && api.Status == 409 {
			_ = b.Refresh()
		}
		return berry.Ticket{}, b.handle(err)
	}
	var project *berry.Project
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
		for i := range d.Snapshot.Projects {
			if d.Snapshot.Projects[i].ID == updated.ProjectID {
				project = &d.Snapshot.Projects[i]
			}
		}
	})
	b.emit()
	return updated.Ticket(project), nil
}

// SetStatus is Update with only the status: todo | in_progress | blocked | done.
func (b *Betty) SetStatus(id, status string) (berry.Ticket, error) {
	return b.Update(id, map[string]any{"status": status})
}

// Claim makes a task mine ("내가 할게").
func (b *Betty) Claim(id string) (berry.Ticket, error) {
	s := b.prefs.Get().Snapshot
	if s == nil {
		return berry.Ticket{}, errors.New("새로고침한 뒤 다시 해 주세요")
	}
	return b.Update(id, map[string]any{"assignee_id": s.User.ID})
}

// Create adds a task (mine, or a teammate's when AssigneeID is set).
func (b *Betty) Create(in berry.CreateInput) (berry.Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	created, err := b.client.Create(ctx, in)
	if err != nil {
		return berry.Ticket{}, b.handle(err)
	}
	b.prefs.Update(func(d *prefs.Data) {
		d.Seen = prefs.Remember(d.Seen, created.ID) // my own task is never "new"
		d.Notified = prefs.Remember(d.Notified, created.ID)
	})
	go b.Refresh()
	return created, nil
}

// ---------- review inbox ----------

// ApproveProposal adds a proposed task. projectID is needed only when the proposal's
// project is unconfirmed (payload._routing.confirmed == false).
func (b *Betty) ApproveProposal(id, projectID string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.ApproveProposal(ctx, id, projectID) })
}

func (b *Betty) DismissProposal(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.DismissProposal(ctx, id) })
}

func (b *Betty) ApproveMeeting(id, projectID string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.ApproveMeeting(ctx, id, projectID) })
}

func (b *Betty) DismissMeeting(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.DismissMeeting(ctx, id) })
}

func (b *Betty) DismissDesignQuestion(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.DismissDesignQuestion(ctx, id) })
}

// ---------- calendar ----------

// SaveMeeting creates a meeting (id "") or edits one; editing is based on the version last loaded.
func (b *Betty) SaveMeeting(id string, in berry.MeetingInput) error {
	if id != "" {
		if s := b.prefs.Get().Snapshot; s != nil {
			for _, m := range s.Calendar.Meetings {
				if m.ID == id {
					in.ExpectedUpdatedAt = m.UpdatedAt
				}
			}
		}
	}
	return b.inbox(func(ctx context.Context) error { return b.client.SaveMeeting(ctx, id, in) })
}

// ---------- automation history ----------

// History loads a page of the assistant's automatic changes; status is all | active | undone.
func (b *Betty) History(cursor, status string) (berry.History, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	h, err := b.client.History(ctx, cursor, status, "")
	if err != nil {
		return berry.History{}, b.handle(err)
	}
	return h, nil
}

// TaskHistory lists the automatic changes made to one task, newest first.
func (b *Betty) TaskHistory(ticketID string) ([]berry.Change, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	h, err := b.client.History(ctx, "", "all", ticketID)
	if err != nil {
		return nil, b.handle(err)
	}
	if h.Items == nil {
		h.Items = []berry.Change{}
	}
	return h.Items, nil
}

// Capture sends a pasted conversation to the extractor for this project.
func (b *Betty) Capture(content, projectID string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("붙여 넣은 대화가 없어요")
	}
	if len([]rune(content)) > 40000 {
		return errors.New("대화가 너무 길어요. 4만 자 안으로 나눠 주세요")
	}
	return b.inbox(func(ctx context.Context) error { return b.client.Capture(ctx, content, projectID) })
}

// Undo reverts one automatic change and reloads the board.
func (b *Betty) Undo(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.Undo(ctx, id) })
}

// ---------- GitHub ----------

// LinkGitHub marks a task and a GitHub issue/PR as the same work (or unlinks with remove).
func (b *Betty) LinkGitHub(ticketID, repoID string, number int, allowShared, remove bool) error {
	t, ok := b.boardTicket(ticketID)
	if !ok {
		return errors.New("업무를 찾지 못했어요. 새로고침해 주세요")
	}
	itemVersion := ""
	if s := b.prefs.Get().Snapshot; s != nil {
		for _, i := range s.GitHub.Items {
			if i.RepoID == repoID && i.Number == number {
				itemVersion = i.UpdatedAt
			}
		}
	}
	return b.inbox(func(ctx context.Context) error {
		return b.client.GitHubLink(ctx, ticketID, t.UpdatedAt, repoID, number, itemVersion, allowShared, remove)
	})
}

func (b *Betty) RejectGitHub(ticketID, repoID string, number int) error {
	t, ok := b.boardTicket(ticketID)
	if !ok {
		return errors.New("업무를 찾지 못했어요. 새로고침해 주세요")
	}
	return b.inbox(func(ctx context.Context) error {
		return b.client.GitHubReject(ctx, ticketID, t.UpdatedAt, repoID, number)
	})
}

func (b *Betty) AddRepo(fullName, projectID string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.AddRepo(ctx, fullName, projectID) })
}

func (b *Betty) RemoveRepo(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.RemoveRepo(ctx, id) })
}

// ---------- team settings ----------

func (b *Betty) SaveProject(id, name string, channelIDs []string, color string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.SaveProject(ctx, id, name, channelIDs, color) })
}

// ---------- trash and checklist ----------

// Delete moves a task to the trash (restorable) against the version last loaded.
func (b *Betty) Delete(id string) error {
	t, ok := b.boardTicket(id)
	if !ok {
		return errors.New("업무를 찾지 못했어요. 새로고침해 주세요")
	}
	return b.inbox(func(ctx context.Context) error { return b.client.TrashTicket(ctx, id, t.UpdatedAt) })
}

func (b *Betty) Restore(id string) error {
	return b.inbox(func(ctx context.Context) error { return b.client.RestoreTicket(ctx, id) })
}

// EditChecklist changes one checklist item against the revision last loaded.
func (b *Betty) EditChecklist(id string, e berry.ChecklistEdit) error {
	revision := ""
	if s := b.prefs.Get().Snapshot; s != nil {
		revision = s.Progress[id].Revision
	}
	return b.inbox(func(ctx context.Context) error { return b.client.EditChecklist(ctx, id, revision, e) })
}

func (b *Betty) SaveMember(id string, in berry.MemberInput) error {
	return b.inbox(func(ctx context.Context) error { return b.client.SaveMember(ctx, id, in) })
}

func (b *Betty) SetSetting(key string, on bool) error {
	return b.inbox(func(ctx context.Context) error { return b.client.SetSetting(ctx, key, on) })
}

func (b *Betty) inbox(call func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := call(ctx); err != nil {
		var api *berry.APIError
		if errors.As(err, &api) && api.Status == 409 {
			_ = b.Refresh()
		}
		return b.handle(err)
	}
	return b.Refresh()
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

// ---------- onboarding & settings ----------

// FinishOnboarding records that the first-run tour was seen (Help menu shows it again).
func (b *Betty) FinishOnboarding() {
	b.prefs.Update(func(d *prefs.Data) { d.Onboarded = true; d.HintDone = true })
	b.emit()
}

// Autostart reports whether Betty opens at login.
func (b *Betty) Autostart() bool {
	on, err := application.Get().Autostart.IsEnabled()
	return err == nil && on
}

func (b *Betty) SetAutostart(on bool) error {
	if on {
		return application.Get().Autostart.Enable()
	}
	return application.Get().Autostart.Disable()
}

// NotificationsAllowed reports the macOS notification permission without prompting.
func (b *Betty) NotificationsAllowed() bool {
	ok, err := b.notify.CheckNotificationAuthorization()
	return err == nil && ok
}

// AllowNotifications shows the macOS permission prompt (once; later it only reports).
func (b *Betty) AllowNotifications() (bool, error) {
	return b.notify.RequestNotificationAuthorization()
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

// ShowTasks opens the task window centred on screen, e.g. from the quick add panel.
func (b *Betty) ShowTasks() {
	if b.onShowTasks != nil {
		b.onShowTasks()
	}
}

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

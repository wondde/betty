package berry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSendsTokenAndMapsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Header.Get("Authorization") != "Bearer btd_good":
			w.WriteHeader(http.StatusUnauthorized)
		case r.URL.Path == "/api/widget/state":
			json.NewEncoder(w).Encode(State{Member: Member{ID: "me", Name: "정우"}, Tickets: []Ticket{{ID: "t1", Number: 7, Title: "배너", Status: "todo", UpdatedAt: "v1"}}})
		case r.URL.Path == "/api/widget/tickets/t1" && r.Method == http.MethodPatch:
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["expected_updated_at"] != "v1" {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "다른 곳에서 먼저 바뀌었어요."})
				return
			}
			json.NewEncoder(w).Encode(Ticket{ID: "t1", Number: 7, Status: body["status"], UpdatedAt: "v2"})
		case r.URL.Path == "/api/widget/tickets" && r.Method == http.MethodPost:
			var in CreateInput
			json.NewDecoder(r.Body).Decode(&in)
			if in.AssigneeID == nil || *in.AssigneeID != "b" || in.ProjectID != "p" || in.DueDate != "2026-10-23" {
				t.Errorf("create payload = %+v", in)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Ticket{ID: "t2", Number: 8})
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	token := "btd_good"
	c := New(srv.URL, func() string { return token })
	s, err := c.State(ctx)
	if err != nil || s.Member.Name != "정우" || len(s.Tickets) != 1 {
		t.Fatalf("state = %+v, %v", s, err)
	}
	updated, err := c.SetStatus(ctx, s.Tickets[0], "done")
	if err != nil || updated.Status != "done" {
		t.Fatalf("set status = %+v, %v", updated, err)
	}
	_, err = c.SetStatus(ctx, Ticket{ID: "t1", UpdatedAt: "stale"}, "done")
	var api *APIError
	if !errors.As(err, &api) || api.Status != 409 || api.Message != "다른 곳에서 먼저 바뀌었어요." {
		t.Fatalf("conflict err = %v", err)
	}
	b := "b"
	if _, err := c.Create(ctx, CreateInput{Title: "배너 문구", AssigneeID: &b, ProjectID: "p", DueDate: "2026-10-23"}); err != nil {
		t.Fatal(err)
	}
	token = "btd_revoked"
	if _, err := c.State(ctx); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked token err = %v", err)
	}
	if _, err := New("http://127.0.0.1:1", func() string { return "x" }).State(ctx); !errors.Is(err, ErrOffline) {
		t.Fatalf("offline err = %v", err)
	}
}

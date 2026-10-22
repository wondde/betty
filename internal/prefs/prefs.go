// Package prefs stores non-secret settings and the offline snapshot in
// ~/Library/Application Support/Betty/prefs.json.
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/wondde/betty/internal/berry"
)

const DefaultServer = "https://ticket.blueberry-team.com"

type Data struct {
	Server    string       `json:"server,omitempty"` // override for development
	Seen      []string     `json:"seen"`             // tasks already looked at (NEW badge)
	Notified  []string     `json:"notified"`         // tasks already announced
	Order     []string     `json:"order"`            // card order the user dragged
	HintDone  bool         `json:"hint_done"`
	Onboarded bool         `json:"onboarded"`          // finished the first-run tour
	Snapshot  *berry.State `json:"snapshot,omitempty"` // last loaded state, shown offline
}

type Store struct {
	mu   sync.Mutex
	path string
	data Data
}

func Open() *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	s := &Store{path: filepath.Join(dir, "Betty", "prefs.json")}
	if raw, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	return s
}

// Get returns a copy of the current data.
func (s *Store) Get() Data {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

// Update mutates the data under lock and saves it.
func (s *Store) Update(fn func(*Data)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.data)
	_ = os.MkdirAll(filepath.Dir(s.path), 0o700)
	if raw, err := json.MarshalIndent(s.data, "", "  "); err == nil {
		_ = os.WriteFile(s.path, raw, 0o600)
	}
}

func (s *Store) Server() string {
	if v := os.Getenv("BETTY_SERVER"); v != "" {
		return v
	}
	if d := s.Get(); d.Server != "" {
		return d.Server
	}
	return DefaultServer
}

// Remember appends ids to a capped list, skipping ones already present.
func Remember(list []string, ids ...string) []string {
	have := map[string]bool{}
	for _, id := range list {
		have[id] = true
	}
	for _, id := range ids {
		if !have[id] {
			list = append(list, id)
			have[id] = true
		}
	}
	if len(list) > 1000 {
		list = list[len(list)-1000:]
	}
	return list
}

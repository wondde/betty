package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.13.1", "0.13.0", true}, {"0.14.0", "0.13.9", true}, {"1.0", "0.99.99", true},
		{"0.13.0", "0.13.0", false}, {"0.12.10", "0.13.0", false}, {"0.13", "0.13.0", false},
		{"v0.14.0", "0.13.0", true}, {"0.14.0-beta", "0.13.0", false}, {"", "0.13.0", false},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCheck(t *testing.T) {
	latest := Info{Version: "0.14.0", Zip: "Betty-0.14.0.zip", SHA256: "ab"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download/latest.json" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(latest)
	}))
	defer srv.Close()
	info, err := Check(context.Background(), srv.URL, "0.13.0")
	if err != nil || info == nil || info.Version != "0.14.0" {
		t.Fatalf("newer version not offered: %v %v", info, err)
	}
	if info, err := Check(context.Background(), srv.URL, "0.14.0"); err != nil || info != nil {
		t.Fatalf("same version offered: %v %v", info, err)
	}
	empty := httptest.NewServer(http.NotFoundHandler())
	defer empty.Close()
	if info, err := Check(context.Background(), empty.URL, "0.13.0"); err != nil || info != nil {
		t.Fatalf("no release should be quiet: %v %v", info, err)
	}
}

func TestCheckVersion(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Betty.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleShortVersionString</key><string>0.14.1</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkVersion(app, "0.14.1", "0.14.0"); err != nil {
		t.Fatalf("a real newer version is refused: %v", err)
	}
	for _, c := range [][2]string{{"99.0", "0.14.0"}, {"0.14.1", "0.14.2"}, {"0.14.1", "0.14.1"}} {
		if checkVersion(app, c[0], c[1]) == nil {
			t.Errorf("offered %s on %s should be refused (app is 0.14.1)", c[0], c[1])
		}
	}
}

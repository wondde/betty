package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

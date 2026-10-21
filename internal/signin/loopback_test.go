package signin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The browser is simulated: "open" follows the start URL straight to the loopback callback.
func TestRunReceivesCodeOnLoopbackAndKeepsVerifier(t *testing.T) {
	var challenge string
	res, err := Run(context.Background(), "https://ticket.example", "Betty · test", func(start string) error {
		u, _ := url.Parse(start)
		q := u.Query()
		if u.Path != "/auth/discord/start" || q.Get("client") != "app" || q.Get("device") != "Betty · test" {
			t.Errorf("unexpected start URL %s", start)
		}
		if !strings.HasPrefix(q.Get("redirect"), "http://127.0.0.1:") || !strings.HasSuffix(q.Get("redirect"), "/callback") {
			t.Errorf("redirect must be loopback, got %s", q.Get("redirect"))
		}
		challenge = q.Get("challenge")
		go func() {
			r, err := http.Get(q.Get("redirect") + "?code=one-time")
			if err == nil {
				body, _ := io.ReadAll(r.Body)
				r.Body.Close()
				if !strings.Contains(string(body), "로그인 완료") {
					t.Errorf("browser page should confirm sign-in")
				}
			}
		}()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "one-time" {
		t.Fatalf("code = %q", res.Code)
	}
	sum := sha256.Sum256([]byte(res.Verifier))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
		t.Fatal("verifier does not match the challenge sent to the server")
	}
}

func TestRunReportsServerError(t *testing.T) {
	_, err := Run(context.Background(), "https://ticket.example", "x", func(start string) error {
		u, _ := url.Parse(start)
		go http.Get(u.Query().Get("redirect") + "?error=" + url.QueryEscape("등록된 팀원이 아니에요"))
		return nil
	})
	if err == nil || err.Error() != "등록된 팀원이 아니에요" {
		t.Fatalf("err = %v", err)
	}
}

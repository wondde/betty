// Package signin runs the native-app Discord sign-in (RFC 8252 loopback flow):
// open the system browser, receive the one-time code on 127.0.0.1, and keep a
// PKCE-style verifier so only this app can exchange the code.
package signin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Result struct {
	Code     string
	Verifier string
}

const donePage = `<!doctype html><meta charset="utf-8"><title>Betty</title>
<style>body{font:15px -apple-system,BlinkMacSystemFont,sans-serif;display:grid;place-items:center;height:100vh;margin:0;color:#1d1d1f;background:#f5f5f7}
@media(prefers-color-scheme:dark){body{color:#f5f5f7;background:#1d1d1f}}div{text-align:center}b{font-size:20px}</style>
<div><b>%s</b><p>%s</p></div>`

func randomURLSafe(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Run blocks until the browser comes back to the loopback listener, ctx ends, or 5 minutes pass.
// open is called with the URL to show in the system browser.
func Run(ctx context.Context, server, device string, open func(string) error) (Result, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Result{}, err
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	redirect := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	verifier := randomURLSafe(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	type outcome struct {
		code string
		err  error
	}
	done := make(chan outcome, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if msg := r.URL.Query().Get("error"); msg != "" {
			fmt.Fprintf(w, donePage, "로그인하지 못했어요", msg)
			select {
			case done <- outcome{err: errors.New(msg)}:
			default:
			}
			return
		}
		fmt.Fprintf(w, donePage, "로그인 완료", "Betty로 돌아가세요. 이 탭은 닫아도 돼요.")
		select {
		case done <- outcome{code: r.URL.Query().Get("code")}:
		default:
		}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()

	q := url.Values{"client": {"app"}, "device": {device}, "challenge": {challenge}, "redirect": {redirect}}
	if err := open(server + "/auth/discord/start?" + q.Encode()); err != nil {
		return Result{}, err
	}

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()
	select {
	case o := <-done:
		if o.err != nil {
			return Result{}, o.err
		}
		if o.code == "" {
			return Result{}, errors.New("로그인 정보를 받지 못했어요")
		}
		return Result{Code: o.code, Verifier: verifier}, nil
	case <-timeout.C:
		return Result{}, errors.New("로그인 시간이 지났어요. 다시 시도해 주세요")
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

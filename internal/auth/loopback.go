package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// OpenBrowserFunc is the function used to open the login URL.
// Tests inject a fake.
type OpenBrowserFunc func(url string) error

// Login starts a loopback server and walks the user through the existing
// /user/auth/?state&callback flow on the backend. Returns a *BearerCreds
// on success.
func Login(ctx context.Context, server string, open OpenBrowserFunc) (Creds, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	callback := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	type result struct {
		token string
		err   error
	}
	resultCh := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		gotState := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if gotState != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			resultCh <- result{err: errors.New("loopback: state mismatch")}
			return
		}
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			resultCh <- result{err: errors.New("loopback: empty code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h2>Logged in.</h2><p>You can close this window.</p></body></html>`)
		resultCh <- result{token: code}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln) //nolint:errcheck
	defer srv.Close()

	loginURL := fmt.Sprintf("%s/user/auth/?state=%s&callback=%s",
		server, url.QueryEscape(state), url.QueryEscape(callback))

	if err := open(loginURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return &BearerCreds{
			Token:     res.token,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}
}

func randomState() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLogin_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	// Fake browser: when the CLI opens the URL, parse `callback` and `state`,
	// then GET callback?state=<state>&code=tok123.
	openBrowser := func(rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cb := u.Query().Get("callback")
		go func() {
			// brief delay so the loopback server is up
			time.Sleep(20 * time.Millisecond)
			resp, err := http.Get(cb + "?state=" + url.QueryEscape(state) + "&code=tok123")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	creds, err := Login(context.Background(), "https://server.example", openBrowser)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	b, ok := creds.(*BearerCreds)
	if !ok {
		t.Fatalf("expected *BearerCreds, got %T", creds)
	}
	if b.Token != "tok123" {
		t.Fatalf("token: got %q", b.Token)
	}
	if time.Until(b.ExpiresAt) < 6*24*time.Hour {
		t.Fatalf("expires_at should be ~7 days out, got %v", b.ExpiresAt)
	}
}

func TestLogin_RejectsBadState(t *testing.T) {
	openBrowser := func(rawURL string) error {
		u, _ := url.Parse(rawURL)
		cb := u.Query().Get("callback")
		go func() {
			time.Sleep(20 * time.Millisecond)
			resp, err := http.Get(cb + "?state=WRONG&code=tok")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Login(ctx, "https://server.example", openBrowser)
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}

func TestLogin_ContextCancel(t *testing.T) {
	openBrowser := func(string) error { return nil } // never hits callback
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := Login(ctx, "https://server.example", openBrowser)
	if err == nil {
		t.Fatal("expected error when context expires")
	}
}

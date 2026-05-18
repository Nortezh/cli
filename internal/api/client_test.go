package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeCreds struct{ token string }

func (f *fakeCreds) Apply(r *http.Request) { r.Header.Set("Authorization", "Bearer "+f.token) }

func newClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Creds:      &fakeCreds{token: "tkn"},
	}
}

func TestInvoke_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s", r.Method)
		}
		if r.URL.Path != "/deployment.list" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tkn" {
			t.Errorf("auth: got %q", r.Header.Get("Authorization"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type: got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(bytes.TrimSpace(body), []byte(`{"project_id":"p1"}`)) {
			t.Errorf("body: got %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[{"name":"a"}]}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	var out struct {
		Items []struct{ Name string } `json:"items"`
	}
	if err := c.Invoke(context.Background(), "deployment.list",
		map[string]string{"project_id": "p1"}, &out); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Name != "a" {
		t.Fatalf("decode: %+v", out)
	}
}

func TestInvoke_NilBodySendsEmptyObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if got := string(bytes.TrimSpace(body)); got != "{}" {
			t.Errorf("nil body should send {}, got %s", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":null}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	if err := c.Invoke(context.Background(), "x.y", nil, nil); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
}

func TestInvoke_ErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"BAD_INPUT","message":"name required"}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "BAD_INPUT" || apiErr.Message != "name required" {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestInvoke_UnauthorizedMapsToErrUnauthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"UNAUTHORIZED","message":"no authorization"}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestInvoke_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "http_error" || apiErr.HTTPStatus != 500 {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestInvoke_DebugRedactsAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	c := newClient(t, srv)
	c.Debug = true
	c.DebugWriter = &buf
	_ = c.Invoke(context.Background(), "x.y", nil, nil)

	if strings.Contains(buf.String(), "tkn") {
		t.Fatalf("debug log leaked token: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "POST") {
		t.Fatalf("expected POST in debug log, got %s", buf.String())
	}
}

func TestInvoke_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hang briefly so context cancel wins
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.Invoke(ctx, "x.y", nil, nil)
	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
}

// Confirm marshalling rules: errors.go encodes Error.Error() including code.
func TestError_String(t *testing.T) {
	e := &Error{Code: "X", Message: "y"}
	if got := e.Error(); !strings.Contains(got, "X") || !strings.Contains(got, "y") {
		t.Fatalf("got %q", got)
	}
}

// Sanity: types in types.go compile and marshal as expected.
func TestTypes_MarshalRoundtrip(t *testing.T) {
	in := Deployment{Name: "d1", Revision: 3, Status: "running"}
	b, _ := json.Marshal(in)
	var out Deployment
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "d1" || out.Revision != 3 {
		t.Fatalf("roundtrip: %+v", out)
	}
}

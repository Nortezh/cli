package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nortezh/cli/internal/auth"
)

const debugBodyLimit = 4 * 1024

// Client is an arpc HTTP client that posts to BaseURL/method and unwraps the
// {ok, result, error} envelope.
type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	Creds       auth.Creds
	Debug       bool
	DebugWriter io.Writer // defaults to os.Stderr when nil
}

// envelope matches the arpc response shape.
type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *remoteError    `json:"error"`
}

type remoteError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Invoke POSTs to BaseURL/method with body marshalled as JSON.
// body == nil sends "{}". out == nil or result == null skips decode.
// Non-2xx returns *Error{Code:"http_error"}. UNAUTHORIZED maps to ErrUnauthenticated.
func (c *Client) Invoke(ctx context.Context, method string, body, out any) error {
	var reqBody []byte
	if body == nil {
		reqBody = []byte("{}")
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = b
	}

	url := c.BaseURL + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Creds != nil {
		c.Creds.Apply(req)
	}

	c.debugReq(req, reqBody)

	httpc := c.HTTPClient
	if httpc == nil {
		httpc = http.DefaultClient
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.debugResp(resp, respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Error{Code: "http_error", HTTPStatus: resp.StatusCode, Message: string(truncate(respBody, debugBodyLimit))}
	}

	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if !env.OK {
		if env.Error == nil {
			return &Error{Code: "unknown", Message: "ok=false with no error block"}
		}
		if env.Error.Code == "UNAUTHORIZED" {
			return ErrUnauthenticated
		}
		return &Error{Code: env.Error.Code, Message: env.Error.Message}
	}
	if out == nil || len(env.Result) == 0 || string(env.Result) == "null" {
		return nil
	}
	return json.Unmarshal(env.Result, out)
}

func (c *Client) writer() io.Writer {
	if c.DebugWriter != nil {
		return c.DebugWriter
	}
	return os.Stderr
}

func (c *Client) debugReq(r *http.Request, body []byte) {
	if !c.Debug {
		return
	}
	w := c.writer()
	fmt.Fprintf(w, "[ntzh] -> %s %s\n", r.Method, r.URL)
	for k, v := range r.Header {
		val := v
		if k == "Authorization" {
			val = []string{"[REDACTED]"}
		}
		fmt.Fprintf(w, "[ntzh]    %s: %s\n", k, val)
	}
	fmt.Fprintf(w, "[ntzh]    body: %s\n", truncate(body, debugBodyLimit))
}

func (c *Client) debugResp(r *http.Response, body []byte) {
	if !c.Debug {
		return
	}
	w := c.writer()
	fmt.Fprintf(w, "[ntzh] <- %d %s\n", r.StatusCode, r.Status)
	fmt.Fprintf(w, "[ntzh]    body: %s\n", truncate(body, debugBodyLimit))
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return append(append([]byte{}, b[:n]...), []byte("...[truncated]")...)
}

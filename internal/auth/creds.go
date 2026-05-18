package auth

import (
	"net/http"
	"time"
)

// Creds applies authentication to an outgoing request.
type Creds interface {
	Apply(*http.Request)
}

type BearerCreds struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (c *BearerCreds) Apply(r *http.Request) {
	r.Header.Set("Authorization", "Bearer "+c.Token)
}

type ServiceAccountCreds struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

func (c *ServiceAccountCreds) Apply(r *http.Request) {
	r.SetBasicAuth(c.Email, c.Key)
}

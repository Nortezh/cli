package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/nortezh/cli/internal/config"
)

const credFileName = "credentials.json"

var ErrNoCreds = errors.New("auth: no credentials (run 'ntzh login')")

func credPath() (string, error) {
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, credFileName), nil
}

func Load() (Creds, error) {
	p, err := credPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, ErrNoCreds
	}
	if err != nil {
		return nil, err
	}
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, err
	}
	switch head.Kind {
	case "bearer":
		var c BearerCreds
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		return &c, nil
	case "service_account":
		var c ServiceAccountCreds
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		return &c, nil
	default:
		return nil, errors.New("auth: unknown credentials kind: " + head.Kind)
	}
}

func Save(c Creds) error {
	d, err := config.Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	var payload []byte
	switch v := c.(type) {
	case *BearerCreds:
		payload, err = json.MarshalIndent(struct {
			Kind string `json:"kind"`
			*BearerCreds
		}{"bearer", v}, "", "  ")
	case *ServiceAccountCreds:
		payload, err = json.MarshalIndent(struct {
			Kind string `json:"kind"`
			*ServiceAccountCreds
		}{"service_account", v}, "", "  ")
	default:
		return errors.New("auth: unsupported creds type")
	}
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, credFileName), payload, 0o600)
}

func Wipe() error {
	p, err := credPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

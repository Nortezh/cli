package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultServer = "https://api.nortezh.com"
	envConfigDir  = "NTZH_CONFIG_DIR"
	envServer     = "NTZH_SERVER"
	envProject    = "NTZH_PROJECT"
	fileName      = "config.json"
)

type Config struct {
	Server string `json:"server,omitempty"`
}

// Dir returns the directory where ntzh stores its files.
// Precedence: $NTZH_CONFIG_DIR > $XDG_CONFIG_HOME/ntzh > os.UserConfigDir()/ntzh.
func Dir() (string, error) {
	if d := os.Getenv(envConfigDir); d != "" {
		return d, nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ntzh"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ntzh"), nil
}

func path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fileName), nil
}

func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(c *Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, fileName), b, 0o644)
}

func ResolveServer(flag string, c *Config) string {
	if flag != "" {
		return flag
	}
	if v := os.Getenv(envServer); v != "" {
		return v
	}
	if c != nil && c.Server != "" {
		return c.Server
	}
	return DefaultServer
}

func ResolveProject(flag string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv(envProject)
}

// Package config reads and writes ~/.config/claude-usage/config.toml.
//
// Missing config file is not an error — defaults are returned. Writes
// create the directory if needed.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user-tunable knobs.
type Config struct {
	PollIntervalSeconds int    `toml:"poll_interval_seconds"`
	WarnThreshold       int    `toml:"warn_threshold"`
	AlertThreshold      int    `toml:"alert_threshold"`
	Notify              bool   `toml:"notify"`
	OrgID               string `toml:"org_id"`
}

// Default returns the baseline config.
func Default() Config {
	return Config{
		PollIntervalSeconds: 60,
		WarnThreshold:       80,
		AlertThreshold:      95,
		Notify:              true,
	}
}

// Path returns ~/.config/claude-usage/config.toml (honouring XDG_CONFIG_HOME).
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func configDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "claude-usage"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "claude-usage"), nil
}

// Load returns the config from disk, falling back to Default() if the file
// doesn't exist. Returns an error for malformed TOML.
func Load() (Config, error) {
	c := Default()
	path, err := Path()
	if err != nil {
		return c, err
	}
	_, err = toml.DecodeFile(path, &c)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("read %s: %w", path, err)
	}
	return c, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

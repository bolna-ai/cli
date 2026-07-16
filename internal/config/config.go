// Package config manages bolna-cli's small on-disk settings file (default
// profile, theme, known profile names) at $XDG_CONFIG_HOME/bolna/config.json
// (~/Library/Application Support/bolna on macOS via os.UserConfigDir).
// API keys are never stored here — see internal/auth for that.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	// DefaultProfile is used when --profile is not passed.
	DefaultProfile string `json:"default_profile"`
	// Profiles lists known profile names, for `bolna login`/completions.
	Profiles []string `json:"profiles,omitempty"`
	// Theme is the active Lip Gloss color theme name.
	Theme string `json:"theme,omitempty"`
}

func defaults() Config {
	return Config{DefaultProfile: "default", Theme: "bolna"}
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "bolna"), nil
}

func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the config file, returning sane defaults if it doesn't exist
// yet (first run).
func Load() (Config, error) {
	p, err := path()
	if err != nil {
		return defaults(), err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return defaults(), err
	}
	cfg := defaults()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return defaults(), err
	}
	return cfg, nil
}

// Save writes the config file, creating its directory if needed.
func Save(cfg Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	p, err := path()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o600)
}

// AddProfile records a profile name if not already known, and saves.
func AddProfile(cfg Config, profile string) (Config, error) {
	for _, p := range cfg.Profiles {
		if p == profile {
			return cfg, Save(cfg)
		}
	}
	cfg.Profiles = append(cfg.Profiles, profile)
	return cfg, Save(cfg)
}

// Package auth stores and retrieves Bolna API keys from the OS keychain
// (macOS Keychain, Linux Secret Service via D-Bus, Windows Credential
// Manager), with BOLNA_API_KEY as an env var override for CI/scripting.
package auth

import (
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const service = "bolna-cli"

// EnvVar is checked before the keychain and always wins — CI/scripting
// escape hatch, and lets `BOLNA_API_KEY=... bolna ...` work with no setup.
const EnvVar = "BOLNA_API_KEY"

// Source reports where an API key was loaded from, so callers (bolna doctor,
// whoami) can tell the user what's in effect.
type Source string

const (
	SourceEnv      Source = "env"
	SourceKeychain Source = "keychain"
	SourceNone     Source = "none"
)

// Resolve returns the API key in effect for profile, and where it came from.
// profile "" is normalized to "default".
func Resolve(profile string) (key string, source Source, err error) {
	if v := os.Getenv(EnvVar); v != "" {
		return v, SourceEnv, nil
	}
	key, err = get(profile)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", SourceNone, nil
		}
		return "", SourceNone, fmt.Errorf("reading keychain: %w", err)
	}
	return key, SourceKeychain, nil
}

// Store saves an API key in the OS keychain for the given profile.
func Store(profile, apiKey string) error {
	return keyring.Set(service, account(profile), apiKey)
}

// Delete removes a stored API key for the given profile. It is not an error
// to delete a profile that was never stored.
func Delete(profile string) error {
	err := keyring.Delete(service, account(profile))
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

func get(profile string) (string, error) {
	return keyring.Get(service, account(profile))
}

func account(profile string) string {
	if profile == "" {
		return "default"
	}
	return profile
}

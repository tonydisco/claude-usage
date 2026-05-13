// Package auth stores the claude.ai session cookie in the OS keychain.
//
// On macOS this is Keychain Access; on Windows it's Credential Manager;
// on Linux it's Secret Service (gnome-keyring / kwallet). All via
// zalando/go-keyring.
package auth

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "claude-usage"
	cookieKey      = "session-cookie"
)

// ErrNotLoggedIn is returned when no credential is stored.
var ErrNotLoggedIn = errors.New("not logged in (no credential in keychain)")

// SaveCookie stores the session cookie under our service in the OS keychain.
// Overwrites any previous value.
func SaveCookie(cookie string) error {
	return keyring.Set(keyringService, cookieKey, cookie)
}

// LoadCookie returns the stored session cookie, or ErrNotLoggedIn if none.
func LoadCookie() (string, error) {
	v, err := keyring.Get(keyringService, cookieKey)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotLoggedIn
	}
	return v, err
}

// DeleteCookie removes the stored credential. Idempotent.
func DeleteCookie() error {
	err := keyring.Delete(keyringService, cookieKey)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

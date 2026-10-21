// Package vault keeps the device token in the macOS Keychain (Windows Credential
// Manager / Secret Service elsewhere), never in a plain file.
package vault

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "com.blueberryteam.betty"
const account = "device-token"

func Token() string {
	t, err := keyring.Get(service, account)
	if err != nil {
		return ""
	}
	return t
}

func SetToken(token string) error { return keyring.Set(service, account, token) }

func ClearToken() error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

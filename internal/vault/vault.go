// Package vault keeps the device token in the macOS Keychain (Windows Credential
// Manager / Secret Service elsewhere), never in a plain file.
package vault

import (
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

const account = "device-token"

// service is separate for a development server (BETTY_SERVER), so testing never touches
// the real sign-in.
var service = func() string {
	if os.Getenv("BETTY_SERVER") != "" {
		return "com.blueberryteam.betty.dev"
	}
	return "com.blueberryteam.betty"
}()

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

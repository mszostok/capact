//go:build darwin
// +build darwin

package credstore

import (
	"capact.io/capact/internal/cli/config"
	"github.com/pkg/errors"
)

func openStore() (Keyring, error) {
	backend := config.GetCredentialsStoreBackend()

	switch backend {
	case "keychain":
		return &Keychain{}, nil
	case "file":
		return nil, nil // TODO
	case "pass":
		return nil, nil // TODO
	default:
		return nil, errors.New("not supported")
	}

	return nil, nil // TODO
}

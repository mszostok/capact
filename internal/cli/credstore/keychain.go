//go:build darwin
// +build darwin

package credstore

import (
	"github.com/99designs/keyring"
)

// Keychain is a simple adapter to zalando go-keyring.
type Keychain struct{}

func (k Keychain) Get(key string) (keyring.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (k Keychain) Set(item keyring.Item) error {
	//TODO implement me
	panic("implement me")
}

func (k Keychain) Remove(key string) error {
	//TODO implement me
	panic("implement me")
}

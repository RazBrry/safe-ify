package config

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassphrase returns a bcrypt hash of the given passphrase.
func HashPassphrase(passphrase string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(passphrase), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("cannot hash passphrase: %w", err)
	}
	return string(hash), nil
}

// VerifyPassphrase checks if the given passphrase matches the stored hash.
// Returns nil on success, error on mismatch or other failure.
func VerifyPassphrase(hash, passphrase string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passphrase)); err != nil {
		return fmt.Errorf("incorrect passphrase")
	}
	return nil
}

// HasPassphrase returns true if the global config has a passphrase set.
func (cfg *GlobalConfig) HasPassphrase() bool {
	return cfg.PassphraseHash != ""
}

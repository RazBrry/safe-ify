package config

import "testing"

func TestHashAndVerifyPassphrase(t *testing.T) {
	t.Parallel()
	hash, err := HashPassphrase("test-passphrase-123")
	if err != nil {
		t.Fatalf("HashPassphrase failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Correct passphrase should verify.
	if err := VerifyPassphrase(hash, "test-passphrase-123"); err != nil {
		t.Errorf("VerifyPassphrase with correct passphrase failed: %v", err)
	}

	// Wrong passphrase should fail.
	if err := VerifyPassphrase(hash, "wrong-passphrase"); err == nil {
		t.Error("VerifyPassphrase with wrong passphrase should have failed")
	}
}

func TestGlobalConfig_HasPassphrase(t *testing.T) {
	t.Parallel()

	cfg := &GlobalConfig{}
	if cfg.HasPassphrase() {
		t.Error("empty config should not have passphrase")
	}

	cfg.PassphraseHash = "$2a$10$something"
	if !cfg.HasPassphrase() {
		t.Error("config with hash should have passphrase")
	}
}

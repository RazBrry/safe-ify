package config

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSigningKeys_RoundTrip(t *testing.T) {
	t.Parallel()
	passphrase := "test-passphrase-123"

	sc, err := GenerateSigningKeys(passphrase)
	if err != nil {
		t.Fatalf("GenerateSigningKeys failed: %v", err)
	}

	// Public key should be parseable.
	pubKey, err := sc.ParsePublicKey()
	if err != nil {
		t.Fatalf("ParsePublicKey failed: %v", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		t.Fatalf("unexpected public key size: %d", len(pubKey))
	}

	// Private key should decrypt with correct passphrase.
	privKey, err := sc.DecryptPrivateKey(passphrase)
	if err != nil {
		t.Fatalf("DecryptPrivateKey failed: %v", err)
	}
	if len(privKey) != ed25519.PrivateKeySize {
		t.Fatalf("unexpected private key size: %d", len(privKey))
	}

	// Wrong passphrase should fail.
	_, err = sc.DecryptPrivateKey("wrong-passphrase")
	if err == nil {
		t.Fatal("DecryptPrivateKey with wrong passphrase should have failed")
	}

	// Sign and verify should work.
	message := []byte("test message to sign")
	sig := ed25519.Sign(privKey, message)
	if !ed25519.Verify(pubKey, message, sig) {
		t.Fatal("signature verification failed for correct message")
	}

	// Tampered message should fail verification.
	tampered := []byte("tampered message")
	if ed25519.Verify(pubKey, tampered, sig) {
		t.Fatal("signature verification should have failed for tampered message")
	}
}

func TestSignAndVerifyProjectConfig(t *testing.T) {
	t.Parallel()
	passphrase := "test-passphrase-456"

	sc, err := GenerateSigningKeys(passphrase)
	if err != nil {
		t.Fatalf("GenerateSigningKeys failed: %v", err)
	}

	privKey, err := sc.DecryptPrivateKey(passphrase)
	if err != nil {
		t.Fatalf("DecryptPrivateKey failed: %v", err)
	}

	pubKey, err := sc.ParsePublicKey()
	if err != nil {
		t.Fatalf("ParsePublicKey failed: %v", err)
	}

	// Create a temp project config.
	dir := t.TempDir()
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte("instance: test\napps:\n  api:\n    uuid: abc123\n")
	if err := os.WriteFile(projectPath, content, 0o644); err != nil {
		t.Fatalf("cannot write test project config: %v", err)
	}

	// Verify should fail before signing (no .sig file).
	err = VerifyProjectSignature(projectPath, pubKey)
	if err == nil {
		t.Fatal("expected error for missing signature")
	}
	if _, ok := err.(*SignatureMissingError); !ok {
		t.Fatalf("expected SignatureMissingError, got %T: %v", err, err)
	}

	// Sign the project config.
	if err := SignProjectConfig(projectPath, privKey); err != nil {
		t.Fatalf("SignProjectConfig failed: %v", err)
	}

	// Verify should succeed.
	if err := VerifyProjectSignature(projectPath, pubKey); err != nil {
		t.Fatalf("VerifyProjectSignature failed after signing: %v", err)
	}

	// Tamper with the project config.
	tampered := []byte("instance: hacked\napps:\n  api:\n    uuid: evil\n")
	if err := os.WriteFile(projectPath, tampered, 0o644); err != nil {
		t.Fatalf("cannot write tampered config: %v", err)
	}

	// Verify should fail for tampered content.
	err = VerifyProjectSignature(projectPath, pubKey)
	if err == nil {
		t.Fatal("expected error for tampered config")
	}
	if _, ok := err.(*SignatureInvalidError); !ok {
		t.Fatalf("expected SignatureInvalidError, got %T: %v", err, err)
	}
}

func TestGlobalConfig_HasSigningKeys(t *testing.T) {
	t.Parallel()

	cfg := &GlobalConfig{}
	if cfg.HasSigningKeys() {
		t.Error("empty config should not have signing keys")
	}

	cfg.Signing = SigningConfig{
		PublicKey:    "something",
		EncryptedKey: "something",
	}
	if !cfg.HasSigningKeys() {
		t.Error("config with keys should have signing keys")
	}
}

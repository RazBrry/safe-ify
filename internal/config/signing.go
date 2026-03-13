package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters for key derivation from passphrase.
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32 // AES-256

	// sigFilename is the signature file written alongside .safe-ify.yaml.
	sigFilename = ".safe-ify.sig"
)

// SigningConfig holds the Ed25519 signing key material stored in global config.
type SigningConfig struct {
	PublicKey          string `yaml:"public_key"`           // base64-encoded Ed25519 public key
	EncryptedKey       string `yaml:"encrypted_private_key"` // base64-encoded AES-GCM ciphertext of private key
	KDFSalt            string `yaml:"kdf_salt"`             // base64-encoded Argon2id salt
}

// GenerateSigningKeys creates an Ed25519 keypair and encrypts the private key
// with a key derived from passphrase using Argon2id + AES-GCM.
// Returns a SigningConfig ready to store in global config.
func GenerateSigningKeys(passphrase string) (*SigningConfig, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cannot generate signing keypair: %w", err)
	}

	// Generate a random salt for Argon2id.
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("cannot generate salt: %w", err)
	}

	// Derive encryption key from passphrase.
	encKey := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encrypt the private key with AES-GCM.
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cannot create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("cannot generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(priv), nil)

	return &SigningConfig{
		PublicKey:    base64.StdEncoding.EncodeToString(pub),
		EncryptedKey: base64.StdEncoding.EncodeToString(ciphertext),
		KDFSalt:      base64.StdEncoding.EncodeToString(salt),
	}, nil
}

// DecryptPrivateKey decrypts the Ed25519 private key using the given passphrase.
func (sc *SigningConfig) DecryptPrivateKey(passphrase string) (ed25519.PrivateKey, error) {
	salt, err := base64.StdEncoding.DecodeString(sc.KDFSalt)
	if err != nil {
		return nil, fmt.Errorf("cannot decode KDF salt: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(sc.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("cannot decode encrypted key: %w", err)
	}

	// Derive the same encryption key from passphrase.
	encKey := argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cannot create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("encrypted key too short")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt signing key (wrong passphrase?): %w", err)
	}

	return ed25519.PrivateKey(plaintext), nil
}

// ParsePublicKey decodes the base64-encoded public key.
func (sc *SigningConfig) ParsePublicKey() (ed25519.PublicKey, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(sc.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(pubBytes), nil
}

// SignProjectConfig signs the exact bytes of a project config file and writes
// the signature to a .safe-ify.sig file alongside it.
func SignProjectConfig(projectPath string, privateKey ed25519.PrivateKey) error {
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return fmt.Errorf("cannot read project config for signing: %w", err)
	}

	sig := ed25519.Sign(privateKey, data)

	sigPath := SignaturePath(projectPath)
	encoded := base64.StdEncoding.EncodeToString(sig)
	if err := os.WriteFile(sigPath, []byte(encoded+"\n"), 0o644); err != nil {
		return fmt.Errorf("cannot write signature file: %w", err)
	}

	return nil
}

// VerifyProjectSignature reads the project config and its signature file,
// then verifies the Ed25519 signature using the given public key.
// Returns nil on success, or a descriptive error.
func VerifyProjectSignature(projectPath string, publicKey ed25519.PublicKey) error {
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return fmt.Errorf("cannot read project config: %w", err)
	}

	sigPath := SignaturePath(projectPath)
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SignatureMissingError{ProjectPath: projectPath}
		}
		return fmt.Errorf("cannot read signature file: %w", err)
	}

	// Decode the base64 signature (trim whitespace).
	sigBytes, err := base64.StdEncoding.DecodeString(trimNewlines(string(sigData)))
	if err != nil {
		return &SignatureInvalidError{
			ProjectPath: projectPath,
			Reason:      "cannot decode signature",
		}
	}

	if !ed25519.Verify(publicKey, data, sigBytes) {
		return &SignatureInvalidError{
			ProjectPath: projectPath,
			Reason:      "signature does not match file contents",
		}
	}

	return nil
}

// SignaturePath returns the path to the .safe-ify.sig file for a given project config path.
func SignaturePath(projectPath string) string {
	dir := filepath.Dir(projectPath)
	return filepath.Join(dir, sigFilename)
}

// HasSigningKeys returns true if the global config has signing key material.
func (cfg *GlobalConfig) HasSigningKeys() bool {
	return cfg.Signing.PublicKey != "" && cfg.Signing.EncryptedKey != ""
}

// trimNewlines removes leading/trailing whitespace from a string.
func trimNewlines(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// SecretBox 负责对需要落库的敏感字段做对称加密。
type SecretBox struct {
	key []byte
}

// NewSecretBox 基于配置中的密钥材料派生 AES-256 密钥。
func NewSecretBox(secret string) *SecretBox {
	sum := sha256.Sum256([]byte(secret))
	return &SecretBox{key: sum[:]}
}

// Encrypt 将明文加密成 base64 编码的密文。
func (s *SecretBox) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt 将密文解密成明文。
func (s *SecretBox) Decrypt(cipherText string) (string, error) {
	if cipherText == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("cipher text too short")
	}

	nonce := raw[:gcm.NonceSize()]
	payload := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}

	return string(plain), nil
}

// MaskSecret 用于在响应中返回脱敏后的值。
func MaskSecret(value string) string {
	switch {
	case value == "":
		return ""
	case len(value) <= 4:
		return "****"
	default:
		return value[:2] + "****" + value[len(value)-2:]
	}
}

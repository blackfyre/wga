package postcards

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/blackfyre/wga/internal/config"
)

const recipientTokenEnvelopeVersion = "v1"

var (
	errInvalidRecipientTokenEnvelope = errors.New("postcard recipient access material is invalid")
	errInvalidRecipientTokenKeyring  = errors.New("postcard token keyring is invalid")
)

// sealRecipientToken protects a canonical recipient token with the configured
// active AES-256-GCM key and binds it to its durable delivery identifier.
func sealRecipientToken(keyring config.PostcardTokenKeyring, deliveryID string, token string) (string, error) {
	if deliveryID == "" || !ValidRecipientToken(token) {
		return "", errInvalidRecipientTokenEnvelope
	}

	keyID, key, err := activeRecipientTokenKey(keyring)
	if err != nil {
		return "", err
	}

	gcm, err := recipientTokenGCM(key)
	if err != nil {
		return "", errInvalidRecipientTokenKeyring
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", errors.New("postcard recipient access material could not be protected")
	}

	sealed := gcm.Seal(nil, nonce, []byte(token), []byte(deliveryID))
	payload := make([]byte, 0, len(nonce)+len(sealed))
	payload = append(payload, nonce...)
	payload = append(payload, sealed...)

	return recipientTokenEnvelopeVersion + "." + keyID + "." + base64.RawURLEncoding.EncodeToString(payload), nil
}

// recoverRecipientToken authenticates and opens a delivery-bound envelope,
// then verifies that its canonical token matches the durable lookup digest.
func recoverRecipientToken(keyring config.PostcardTokenKeyring, deliveryID string, envelope string, storedHash string) (string, error) {
	token, err := openRecipientToken(keyring, deliveryID, envelope)
	if err != nil || !recipientTokenHashMatches(token, storedHash) {
		return "", errInvalidRecipientTokenEnvelope
	}

	return token, nil
}

func openRecipientToken(keyring config.PostcardTokenKeyring, deliveryID string, envelope string) (string, error) {
	if deliveryID == "" {
		return "", errInvalidRecipientTokenEnvelope
	}

	keyID, encodedPayload, ok := recipientTokenEnvelopeParts(envelope)
	if !ok {
		return "", errInvalidRecipientTokenEnvelope
	}
	key, ok := keyring.Key(keyID)
	if !ok {
		return "", errInvalidRecipientTokenEnvelope
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil || base64.RawURLEncoding.EncodeToString(payload) != encodedPayload {
		return "", errInvalidRecipientTokenEnvelope
	}
	gcm, err := recipientTokenGCM(key)
	if err != nil {
		return "", errInvalidRecipientTokenEnvelope
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return "", errInvalidRecipientTokenEnvelope
	}

	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(deliveryID))
	if err != nil {
		return "", errInvalidRecipientTokenEnvelope
	}

	token := string(plaintext)
	if !ValidRecipientToken(token) {
		return "", errInvalidRecipientTokenEnvelope
	}

	return token, nil
}

func activeRecipientTokenKey(keyring config.PostcardTokenKeyring) (string, config.PostcardTokenKey, error) {
	keyID := keyring.ActiveKeyID()
	if !validRecipientTokenKeyID(keyID) {
		return "", config.PostcardTokenKey{}, errInvalidRecipientTokenKeyring
	}

	key, ok := keyring.Key(keyID)
	if !ok {
		return "", config.PostcardTokenKey{}, errInvalidRecipientTokenKeyring
	}

	return keyID, key, nil
}

func recipientTokenGCM(key config.PostcardTokenKey) (cipher.AEAD, error) {
	material := key.Bytes()
	defer clear(material)

	block, err := aes.NewCipher(material)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func recipientTokenEnvelopeParts(envelope string) (string, string, bool) {
	parts := strings.Split(envelope, ".")
	if len(parts) != 3 || parts[0] != recipientTokenEnvelopeVersion || !validRecipientTokenKeyID(parts[1]) || parts[2] == "" {
		return "", "", false
	}

	return parts[1], parts[2], true
}

func validRecipientTokenKeyID(keyID string) bool {
	if keyID == "" || len(keyID) > 64 {
		return false
	}
	for index := 0; index < len(keyID); index++ {
		character := keyID[index]
		if character >= 'a' && character <= 'z' {
			continue
		}
		if character >= 'A' && character <= 'Z' {
			continue
		}
		if character >= '0' && character <= '9' {
			continue
		}
		if character == '-' || character == '_' {
			continue
		}
		return false
	}

	return true
}

func recipientTokenHashMatches(token string, storedHash string) bool {
	expected := sha256.Sum256([]byte(token))
	stored, err := hex.DecodeString(storedHash)
	if err != nil || len(stored) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare(expected[:], stored) == 1
}

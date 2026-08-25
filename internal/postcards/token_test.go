package postcards

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
)

func TestRecipientTokenEnvelopeRoundTripUsesFreshNonceAndHidesPlaintext(t *testing.T) {
	keyring := postcardTestKeyring(t)
	token, err := newRecipientToken()
	if err != nil {
		t.Fatal(err)
	}

	first, err := sealRecipientToken(keyring, "delivery-123", token)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sealRecipientToken(keyring, "delivery-123", token)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two envelopes reused the same nonce")
	}
	for _, envelope := range []string{first, second} {
		if !strings.HasPrefix(envelope, "v1.primary.") {
			t.Fatal("envelope does not use the versioned active-key format")
		}
		if strings.Contains(envelope, token) {
			t.Fatal("envelope contains the plaintext bearer token")
		}
		recovered, openErr := openRecipientToken(keyring, "delivery-123", envelope)
		if openErr != nil || recovered != token {
			t.Fatalf("open envelope: token_match=%t err=%v", recovered == token, openErr)
		}
	}
}

func TestRecipientTokenEnvelopeRejectsWrongKeyAADTamperAndVersion(t *testing.T) {
	keyring := postcardTestKeyring(t)
	token, err := newRecipientToken()
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := sealRecipientToken(keyring, "delivery-123", token)
	if err != nil {
		t.Fatal(err)
	}
	wrongKeyring := testPostcardKeyring(t, "primary", map[string][]byte{"primary": bytes.Repeat([]byte{0x24}, 32)})

	for name, open := range map[string]func() error{
		"wrong key": func() error {
			_, err := openRecipientToken(wrongKeyring, "delivery-123", envelope)
			return err
		},
		"wrong delivery AAD": func() error {
			_, err := openRecipientToken(keyring, "delivery-456", envelope)
			return err
		},
		"tampered payload": func() error {
			_, err := openRecipientToken(keyring, "delivery-123", tamperEnvelope(envelope))
			return err
		},
		"unsupported version": func() error {
			_, err := openRecipientToken(keyring, "delivery-123", "v2."+strings.TrimPrefix(envelope, "v1."))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := open(); !errors.Is(err, errInvalidRecipientTokenEnvelope) {
				t.Fatalf("error = %v, want invalid envelope", err)
			}
		})
	}
}

func TestRecipientTokenEnvelopeRequiresCanonicalTokenMatchingStoredHash(t *testing.T) {
	keyring := postcardTestKeyring(t)
	token, err := newRecipientToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sealRecipientToken(keyring, "delivery-123", token+"="); !errors.Is(err, errInvalidRecipientTokenEnvelope) {
		t.Fatalf("non-canonical token error = %v", err)
	}
	envelope, err := sealRecipientToken(keyring, "delivery-123", token)
	if err != nil {
		t.Fatal(err)
	}
	if recovered, err := recoverRecipientToken(keyring, "delivery-123", envelope, HashRecipientToken(token)); err != nil || recovered != token {
		t.Fatalf("recover canonical token: token_match=%t err=%v", recovered == token, err)
	}
	if _, err := recoverRecipientToken(keyring, "delivery-123", envelope, strings.Repeat("0", 64)); !errors.Is(err, errInvalidRecipientTokenEnvelope) {
		t.Fatalf("mismatched hash error = %v", err)
	}
	if recipientTokenHashMatches(token, "not-a-digest") {
		t.Fatal("malformed stored hash matched token")
	}
}

func TestRecipientTokenEnvelopeRejectsZeroValueKeyring(t *testing.T) {
	token, err := newRecipientToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sealRecipientToken(config.PostcardTokenKeyring{}, "delivery-123", token); !errors.Is(err, errInvalidRecipientTokenKeyring) {
		t.Fatalf("zero-value keyring error = %v", err)
	}
}

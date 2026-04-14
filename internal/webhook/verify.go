package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func VerifySignature(payload []byte, signatureHeader string, secret []byte) error {
	if signatureHeader == "" {
		return errors.New("missing X-Hub-Signature-256 header")
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return errors.New("invalid signature format: missing sha256= prefix")
	}

	sigHex := strings.TrimPrefix(signatureHeader, "sha256=")
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %w", err)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return errors.New("signature mismatch")
	}

	return nil
}

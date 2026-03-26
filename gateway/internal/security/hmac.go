package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// Sign produces an HMAC-SHA256 hex signature for the given payload.
func Sign(payload string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that the signature matches the payload using constant-time comparison.
func Verify(payload, signature string, secret []byte) bool {
	expected := Sign(payload, secret)
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	expectedBytes, _ := hex.DecodeString(expected)
	return subtle.ConstantTimeCompare(sigBytes, expectedBytes) == 1
}

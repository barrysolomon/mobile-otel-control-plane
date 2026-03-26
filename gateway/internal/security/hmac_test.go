package security

import (
	"testing"
)

func TestSign_ValidSignature_Verifies(t *testing.T) {
	secret := []byte("test-secret-key-32bytes-long!!")
	payload := `{"alert_id":"fa-001","actions":[]}`

	sig := Sign(payload, secret)
	if sig == "" {
		t.Fatal("Sign returned empty string")
	}

	if !Verify(payload, sig, secret) {
		t.Error("Verify returned false for valid signature")
	}
}

func TestVerify_TamperedPayload_Rejected(t *testing.T) {
	secret := []byte("test-secret-key-32bytes-long!!")
	payload := `{"alert_id":"fa-001"}`
	sig := Sign(payload, secret)

	tampered := `{"alert_id":"fa-002"}`
	if Verify(tampered, sig, secret) {
		t.Error("Verify should reject tampered payload")
	}
}

func TestVerify_WrongSecret_Rejected(t *testing.T) {
	secret1 := []byte("secret-one")
	secret2 := []byte("secret-two")
	payload := `{"alert_id":"fa-001"}`

	sig := Sign(payload, secret1)
	if Verify(payload, sig, secret2) {
		t.Error("Verify should reject wrong secret")
	}
}

func TestVerify_InvalidSignature_Rejected(t *testing.T) {
	secret := []byte("test-secret")
	if Verify("payload", "not-a-valid-sig", secret) {
		t.Error("Verify should reject invalid signature format")
	}
}

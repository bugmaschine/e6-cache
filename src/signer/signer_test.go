package signer

import (
	"testing"
)

func TestSigner(t *testing.T) {
	testData := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	secret := GenerateSecretKey()
	signer := NewSigner(secret)
	signature := signer.Sign(testData)
	if !signer.Verify(testData, signature) {
		t.Errorf("Signature verification failed")
	}

	// Test with invalid signature
	if signer.Verify(testData, "hopefully not a valid signature") {
		t.Errorf("Invalid signature verification passed")
	}

	// Test with valid signature
	if !signer.Verify(testData, signature) {
		t.Errorf("Valid signature verification failed")
	}
}

package signer

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

type Signer struct {
	secret []byte
}

func NewSigner(secret []byte) *Signer {
	return &Signer{secret: secret}
}

func GenerateSecretKey() []byte {
	key := make([]byte, 32) // using over 32 bytes dosen't make sense, as it gets hashed down anyways.
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}

func (s *Signer) Sign(message string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(message))
	signature := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}

func (s *Signer) Verify(message, signature string) bool {
	sigDecoded, err := base64.URLEncoding.DecodeString(signature)
	if err != nil {
		return false // invalid base64 input
	}

	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(message))
	expectedSig := h.Sum(nil)

	return hmac.Equal(sigDecoded, expectedSig)
}

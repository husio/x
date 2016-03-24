package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

func NewRSA256Signer(key *rsa.PrivateKey) Signer {
	return &sigRSA{
		alg:  "RS256",
		key:  key,
		hash: crypto.SHA256,
	}
}

func NewRSA384Signer(key *rsa.PrivateKey) Signer {
	return &sigRSA{
		alg:  "RS384",
		key:  key,
		hash: crypto.SHA384,
	}
}

func NewRSA512Signer(key *rsa.PrivateKey) Signer {
	return &sigRSA{
		alg:  "RS512",
		key:  key,
		hash: crypto.SHA512,
	}
}

type sigRSA struct {
	alg  string
	key  *rsa.PrivateKey
	hash crypto.Hash
}

func (s *sigRSA) Algorithm() string {
	return s.alg
}

func (s *sigRSA) Verify(signature, data []byte) error {
	if !s.hash.Available() {
		return ErrAlgorithmNotAvailable
	}

	hasher := s.hash.New()
	if _, err := hasher.Write(data); err != nil {
		return fmt.Errorf("cannot hash: %s", err)
	}
	b := hasher.Sum(nil)
	if err := rsa.VerifyPKCS1v15(&s.key.PublicKey, s.hash, b, signature); err != nil {
		return ErrInvalidSignature
	}
	return nil
}

func (s *sigRSA) Sign(data []byte) ([]byte, error) {
	if !s.hash.Available() {
		return nil, ErrAlgorithmNotAvailable
	}

	hasher := s.hash.New()
	if _, err := hasher.Write(data); err != nil {
		return nil, fmt.Errorf("cannot hash: %s", err)
	}
	b := hasher.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, s.key, s.hash, b)
}

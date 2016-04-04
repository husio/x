package stamp

import (
	"crypto"
	"crypto/hmac"
	"fmt"
)

// NewHMACSigner return signer using symetric key and SHA256 hashing function.
func NewHMAC256Signer(key []byte) Signer {
	return &sigHMAC{
		alg:  "HS256",
		key:  append([]byte{}, key...),
		hash: crypto.SHA256,
	}
}

// NewHMACSigner return signer using symetric key and SHA384 hashing function.
func NewHMAC384Signer(key []byte) Signer {
	return &sigHMAC{
		alg:  "HS384",
		key:  append([]byte{}, key...),
		hash: crypto.SHA384,
	}
}

// NewHMACSigner return signer using symetric key and SHA512 hashing function.
func NewHMAC512Signer(key []byte) Signer {
	return &sigHMAC{
		alg:  "HS512",
		key:  append([]byte{}, key...),
		hash: crypto.SHA512,
	}
}

type sigHMAC struct {
	alg  string
	key  []byte
	hash crypto.Hash
}

func (s *sigHMAC) Algorithm() string {
	return s.alg
}

func (s *sigHMAC) Verify(signature, data []byte) error {
	if !s.hash.Available() {
		return ErrAlgorithmNotAvailable
	}

	hasher := hmac.New(s.hash.New, s.key)
	if _, err := hasher.Write(data); err != nil {
		return fmt.Errorf("cannot encode data: %s", err)
	}

	if !hmac.Equal(signature, hasher.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}

func (s *sigHMAC) Sign(data []byte) ([]byte, error) {
	if !s.hash.Available() {
		return nil, ErrAlgorithmNotAvailable
	}

	hasher := hmac.New(s.hash.New, s.key)
	if _, err := hasher.Write(data); err != nil {
		return nil, fmt.Errorf("cannot encode data: %s", err)
	}
	return hasher.Sum(nil), nil
}

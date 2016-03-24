package signer

import "errors"

type Signer interface {
	// Algorithm return algorithm used by signer implementation.
	Algorithm() string
	// Verify returns nil if given signature was computed for given data.
	Verify(signature, data []byte) error
	// Sign return signature computed for given data.
	Sign(data []byte) ([]byte, error)
}

var (
	ErrAlgorithmNotAvailable = errors.New("algorithm not available")
	ErrInvalidSignature      = errors.New("invalid signature")
)

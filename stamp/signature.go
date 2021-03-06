package stamp

type Verifier interface {
	// Algorithm return algorithm used by signer implementation.
	Algorithm() string
	// Verify returns nil if given signature was computed for given data.
	Verify(signature, data []byte) error
}

type Signer interface {
	Verifier
	// Sign return signature computed for given data.
	Sign(data []byte) ([]byte, error)
}

package stamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
)

type Vault struct {
	signers atomic.Value
}

type signer struct {
	name      string
	sig       Signer
	validTill time.Time
}

func (v *Vault) namedSigners() []signer {
	sigs := v.signers.Load()
	if sigs == nil {
		return nil
	}
	return sigs.([]signer)
}

func (v *Vault) Add(name string, sig Signer, expireIn time.Duration) {
	old := v.namedSigners()
	new := make([]signer, 0, len(old)+1)

	new = append(new, signer{
		name:      name,
		sig:       sig,
		validTill: time.Now().Add(expireIn),
	})
	// copy all old signers
	for _, sig := range old {
		if sig.name != name {
			new = append(new, signer{
				name:      sig.name,
				sig:       sig.sig,
				validTill: sig.validTill,
			})
		}
	}

	v.signers.Store(new)
}

// signerByID return register signer by name it was registered
func (v *Vault) signerByID(name string) (Signer, error) {
	now := time.Now()
	for _, sig := range v.namedSigners() {
		if sig.name != name {
			continue
		}
		if sig.validTill.Before(now) {
			return nil, ErrExpired
		}
		return sig.sig, nil
	}
	return nil, ErrNoSigner
}

// newestSigner return most recently added, valid signer
func (v *Vault) newestSigner() (string, Signer, bool) {
	now := time.Now()
	for _, sig := range v.namedSigners() {
		if sig.validTill.After(now) {
			return sig.name, sig.sig, true
		}
	}
	return "", nil, false
}

func (v *Vault) Encode(payload interface{}) ([]byte, error) {
	name, sig, ok := v.newestSigner()
	if !ok {
		return nil, ErrNoSigner
	}

	header, err := encodeJSON(tokenHeader{
		Algorithm: sig.Algorithm(),
		KeyID:     name,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot encode header: %s", err)
	}

	content, err := encodeJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("cannot encode payload: %s", err)
	}

	token := bytes.Join([][]byte{header, content}, []byte("."))

	signature, err := sig.Sign(token)
	if err != nil {
		return nil, fmt.Errorf("cannot sign: %s", err)
	}
	signature, err = encode(signature)
	if err != nil {
		return nil, fmt.Errorf("cannot encode signature: %s", err)
	}

	token = bytes.Join([][]byte{token, signature}, []byte("."))
	return token, nil
}

type tokenHeader struct {
	Type      string `json:"typ,omitempty"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid,omitempty"`
}

func (v *Vault) Decode(payload interface{}, token []byte) error {
	chunks := bytes.SplitN(token, []byte("."), 3)
	if len(chunks) != 3 {
		return ErrMalformedToken
	}

	rawHeader := fixPadding(chunks[0])
	rawPayload := fixPadding(chunks[1])
	rawSignature := fixPadding(chunks[2])

	bufsize := enc.DecodedLen(len(rawHeader))
	if size := enc.DecodedLen(len(rawPayload)); size > bufsize {
		bufsize = size
	}
	if size := enc.DecodedLen(len(rawSignature)); size > bufsize {
		bufsize = size
	}
	buf := make([]byte, bufsize)

	// decode header
	b := buf[:enc.DecodedLen(len(rawHeader))]
	if n, err := enc.Decode(b, rawHeader); err != nil {
		return fmt.Errorf("cannot base64 decode header: %s", err)
	} else {
		b = b[:n]
	}

	var header tokenHeader
	if err := json.Unmarshal(bytes.TrimSpace(b), &header); err != nil {
		return fmt.Errorf("cannot JSON decode header: %s", err)
	}

	// decode payload
	b = buf[:enc.DecodedLen(len(rawPayload))]
	if n, err := enc.Decode(b, rawPayload); err != nil {
		return fmt.Errorf("cannot base64 decode payload: %s", err)
	} else {
		b = b[:n]
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return fmt.Errorf("cannot JSON decode payload: %s", err)
	}
	// decode extra claims that will be used later for the validation
	var claims struct {
		ExpirationTime int64 `json:"exp,omitempty"`
		NotBefore      int64 `json:"nbf,omitempty"`
	}
	if err := json.Unmarshal(b, &claims); err != nil {
		return fmt.Errorf("cannot JSON decode payload: %s", err)
	}

	sig, err := v.signerByID(header.KeyID)
	if err != nil {
		return err
	}
	if header.Algorithm != sig.Algorithm() {
		return ErrInvalidSigner
	}

	// validate signature
	b = buf[:enc.DecodedLen(len(rawSignature))]
	if n, err := enc.Decode(b, rawSignature); err != nil {
		return fmt.Errorf("cannot base64 decode signature: %s", err)
	} else {
		b = b[:n]
	}
	beforeSign := token[:len(token)-len(chunks[2])-1]
	if err := sig.Verify(b, beforeSign); err != nil {
		return err
	}

	// make sure token is still valid
	now := time.Now()
	if claims.ExpirationTime != 0 && claims.ExpirationTime < now.Unix() {
		return ErrExpired
	}
	if claims.NotBefore != 0 && claims.NotBefore > now.Unix() {
		return ErrNotReady
	}

	return nil
}

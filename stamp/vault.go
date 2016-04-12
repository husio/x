package stamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
)

// Vault is register holding information about signers and keys used to create
// and verify tokens.
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

// Add registers new signer under given name. If name is already in use, old
// entry is overwritten.
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

// Encode returns token containing given payload and encoded using most recent
// signer.
func (v *Vault) Encode(payload interface{}) ([]byte, error) {
	name, sig, ok := v.newestSigner()
	if !ok {
		return nil, ErrNoSigner
	}

	header, err := encodeJSON(struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid,omitempty"`
	}{
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

// Decode unpack token payload into provided structure. Token signature is
// validated by matching algorithm and key id information from the header with
// vault's signers.
func (v *Vault) Decode(payload interface{}, token []byte) error {
	chunks := bytes.Split(token, []byte("."))
	if len(chunks) != 3 {
		return ErrMalformedToken
	}

	// create big enough buffer
	buf := make([]byte, maxlen(chunks)+3)
	var b []byte

	// decode header
	if n, err := enc.Decode(buf, fixPadding(chunks[0])); err != nil {
		return fmt.Errorf("cannot base64 decode header: %s", err)
	} else {
		b = buf[:n]
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := json.Unmarshal(b, &header); err != nil {
		return fmt.Errorf("cannot JSON decode header: %s", err)
	}

	// decode payload
	if n, err := enc.Decode(buf, fixPadding(chunks[1])); err != nil {
		return fmt.Errorf("cannot base64 decode payload: %s", err)
	} else {
		b = buf[:n]
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return fmt.Errorf("cannot JSON decode payload: %s", err)
	}
	// decode extra claims that will be used later for the validation
	var claims struct {
		ExpirationTime int64 `json:"exp"`
		NotBefore      int64 `json:"nbf"`
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
	if n, err := enc.Decode(buf, fixPadding(chunks[2])); err != nil {
		return fmt.Errorf("cannot base64 decode signature: %s", err)
	} else {
		b = buf[:n]
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

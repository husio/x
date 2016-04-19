package stamp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
)

// Vault is register holding information about signers and keys used to create
// and verify tokens.
type Vault struct {
	signers   atomic.Value
	verifiers atomic.Value
}

func (v *Vault) namedSigners() []signer {
	sigs := v.signers.Load()
	if sigs == nil {
		return nil
	}
	return sigs.([]signer)
}

type signer struct {
	name      string
	sig       Signer
	validTill time.Time
}

func (v *Vault) namedVerifiers() []verifier {
	vers := v.verifiers.Load()
	if vers == nil {
		return nil
	}
	return vers.([]verifier)
}

type verifier struct {
	name string
	ver  Verifier
}

// AddSigner registers new signer under given name. If name is already in use, old
// entry is overwritten.
func (v *Vault) AddSigner(name string, sig Signer, expireIn time.Duration) {
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

// AddVerifier registers new signature verifier under given name. If name is
// ready in use, old entry is overwritten.
func (v *Vault) AddVerifier(name string, ver Verifier) {
	old := v.namedVerifiers()
	new := make([]verifier, 0, len(old)+1)

	new = append(new, verifier{
		name: name,
		ver:  ver,
	})
	// copy all old verifiers
	for _, ver := range old {
		if ver.name != name {
			new = append(new, verifier{
				name: ver.name,
				ver:  ver.ver,
			})
		}
	}

	v.verifiers.Store(new)
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

// verifierByID return register verifier by name it was registered
func (v *Vault) verifierByID(name string) (Verifier, error) {
	for _, v := range v.namedVerifiers() {
		if v.name != name {
			continue
		}
		return v.ver, nil
	}
	sig, err := v.signerByID(name)
	if err != nil {
		return nil, ErrNoVerifier
	}
	return sig, nil
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
// vault's verifiers.
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

	ver, err := v.verifierByID(header.KeyID)
	if err != nil {
		return err
	}
	if header.Algorithm != ver.Algorithm() {
		return ErrInvalidSigner
	}

	// validate signature
	if n, err := enc.Decode(buf, fixPadding(chunks[2])); err != nil {
		return fmt.Errorf("cannot base64 decode signature: %s", err)
	} else {
		b = buf[:n]
	}
	beforeSign := token[:len(token)-len(chunks[2])-1]
	if err := ver.Verify(b, beforeSign); err != nil {
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

// encode serialize given data into JSON and return it's base64 representation
// with base64 padding removed.
func encodeJSON(x interface{}) ([]byte, error) {
	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	return encode(b)
}

func encode(b []byte) ([]byte, error) {
	b64 := make([]byte, base64.URLEncoding.EncodedLen(len(b)))
	enc.Encode(b64, b)
	b64 = bytes.TrimRight(b64, "=")
	return b64, nil
}

// fixPadding return given base64 encoded string with padding characters added
// if necessary.
func fixPadding(b []byte) []byte {
	if n := len(b) % 4; n > 0 {
		res := make([]byte, len(b), len(b)+4)
		copy(res, b)
		return append(res, bytes.Repeat([]byte("="), 4-n)...)
	}
	return b
}

var enc = base64.URLEncoding

func maxlen(a [][]byte) int {
	max := 0
	for _, b := range a {
		if l := len(b); l > max {
			max = l
		}
	}
	return max
}

var (
	ErrAlgorithmNotAvailable = errors.New("algorithm not available")
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrMalformedToken        = errors.New("malformed token")
	ErrInvalidSigner         = errors.New("invalid signer algorithm")
	ErrNoSigner              = errors.New("no signer")
	ErrNoVerifier            = errors.New("no verifier")
	ErrExpired               = errors.New("expired")
	ErrNotReady              = errors.New("token not yet active")
)

func WithVault(ctx context.Context, v *Vault) context.Context {
	return context.WithValue(ctx, "stamp:vault", v)
}

func GetVault(ctx context.Context) *Vault {
	v := ctx.Value("stamp:vault")
	if v == nil {
		panic("vault not present in the context")
	}
	return v.(*Vault)
}

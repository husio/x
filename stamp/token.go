package stamp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Header struct {
	// The "typ" (type) Header Parameter defined by [JWS] and [JWE] is used by
	// JWT applications to declare the media type [IANA.MediaTypes] of this
	// complete JWT.  This is intended for use by the JWT application when
	// values that are not JWTs could also be present in an application data
	// structure that can contain a JWT object; the application can use this
	// value to disambiguate among the different kinds of objects that might be
	// present.  It will typically not be used by applications when it is
	// already known that the object is a JWT.  This parameter is ignored by
	// JWT implementations; any processing of this parameter is performed by
	// the JWT application.  If present, it is RECOMMENDED that its value be
	// "JWT" to indicate that this object is a JWT.  While media type names are
	// not case sensitive, it is RECOMMENDED that "JWT" always be spelled using
	// uppercase characters for compatibility with legacy implementations.
	//
	// Use of this Header Parameter is OPTIONAL.
	Type string `json:"typ,omitempty"`

	// The "alg" (algorithm) Header Parameter identifies the cryptographic
	// algorithm used to secure the JWS. The JWS Signature value is not valid
	// if the "alg" value does not represent a supported algorithm or if there
	// is not a key for use with that algorithm associated with the party that
	// digitally signed or MACed the content. "alg" values should either be
	// registered in the IANA "JSON Web Signature and Encryption Algorithms"
	// registry established by [JWA] or be a value that contains a
	// Collision-Resistant Name.  The "alg" value is a case- sensitive ASCII
	// string containing a StringOrURI value. This Header Parameter MUST be
	// present and MUST be understood and processed by implementations.
	Algorithm string `json:"alg"`
}

type Claims struct {
	// The "iss" (issuer) claim identifies the principal that issued the JWT.
	// The processing of this claim is generally application specific.  The
	// "iss" value is a case-sensitive string containing a StringOrURI value.
	//
	//Use of this claim is OPTIONAL.
	Issuer string `json:"iss,omitempty"`

	// The "sub" (subject) claim identifies the principal that is the subject
	// of the JWT.  The claims in a JWT are normally statements about the
	// subject.  The subject value MUST either be scoped to be locally unique
	// in the context of the issuer or be globally unique.  The processing of
	// this claim is generally application specific.  The "sub" value is a
	// case-sensitive string containing a StringOrURI value.
	//
	//Use of this claim is OPTIONAL.
	Subject string `json:"sub,omitempty"`

	// The "aud" (audience) claim identifies the recipients that the JWT is
	// intended for. Each principal intended to process the JWT MUST identify
	// itself with a value in the audience claim.  If the principal processing
	// the claim does not identify itself with a value in the "aud" claim when
	// this claim is present, then the JWT MUST be rejected.  In the general
	// case, the "aud" value is an array of case- sensitive strings, each
	// containing a StringOrURI value.  In the special case when the JWT has
	// one audience, the "aud" value MAY be a single case-sensitive string
	// containing a StringOrURI value.  The interpretation of audience values
	// is generally application specific.
	//
	//Use of this claim is OPTIONAL.
	Audience string `json:"aud,omitempty"`

	// The "exp" (expiration time) claim identifies the expiration time on or
	// after which the JWT MUST NOT be accepted for processing. The processing
	// of the "exp" claim requires that the current date/time MUST be before
	// the expiration date/time listed in the "exp" claim.
	// Implementers MAY provide for some small leeway, usually no more than a
	// few minutes, to account for clock skew.  Its value MUST be a number
	// containing a NumericDate value.
	//
	//Use of this claim is OPTIONAL.
	ExpirationTime int64 `json:"exp,omitempty"`

	// The "nbf" (not before) claim identifies the time before which the JWT
	// MUST NOT be accepted for processing.  The processing of the "nbf" claim
	// requires that the current date/time MUST be after or equal to the
	// not-before date/time listed in the "nbf" claim.  Implementers MAY
	// provide for some small leeway, usually no more than a few minutes, to
	// account for clock skew.  Its value MUST be a number containing a
	// NumericDate value.
	//
	//Use of this claim is OPTIONAL.
	NotBefore int64 `json:"nbf,omitempty"`

	// The "iat" (issued at) claim identifies the time at which the JWT was
	// issued.  This claim can be used to determine the age of the JWT.  Its
	// value MUST be a number containing a NumericDate value.
	//
	//Use of this claim is OPTIONAL.
	IssuedAt int64 `json:"iat,omitempty"`

	// The "jti" (JWT ID) claim provides a unique identifier for the JWT. The
	// identifier value MUST be assigned in a manner that ensures that there is
	// a negligible probability that the same value will be accidentally
	// assigned to a different data object; if the application uses multiple
	// issuers, collisions MUST be prevented among values produced by different
	// issuers as well.  The "jti" claim can be used to prevent the JWT from
	// being replayed.  The "jti" value is a case- sensitive string.
	//
	//Use of this claim is OPTIONAL.
	JWTID string `json:"jti,omitempty"`
}

func Encode(s Signer, payload interface{}) ([]byte, error) {
	header, err := encode(Header{
		Type:      "JWT",
		Algorithm: s.Algorithm(),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot encode header: %s", err)
	}

	content, err := encode(payload)
	if err != nil {
		return nil, fmt.Errorf("cannot encode payload: %s", err)
	}

	token := bytes.Join([][]byte{header, content}, []byte("."))

	signature, err := s.Sign(token)
	if err != nil {
		return nil, fmt.Errorf("cannot sign: %s", err)
	}

	token = bytes.Join([][]byte{token, signature}, []byte("."))
	return token, nil
}

// encode serialize given data into JSON and return it's base64 representation
// with base64 padding removed.
func encode(x interface{}) ([]byte, error) {
	js, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	b64 := make([]byte, base64.URLEncoding.EncodedLen(len(js)))
	enc.Encode(b64, js)
	return bytes.TrimRight(b64, "="), nil
}

func Decode(s Signer, payload interface{}, token []byte) error {
	chunks := bytes.SplitN(token, []byte("."), 3)
	if len(chunks) != 3 {
		return ErrMalformedToken
	}

	rawHeader := fixPadding(copyb(chunks[0]))
	rawPayload := fixPadding(copyb(chunks[1]))

	bufsize := enc.DecodedLen(len(rawHeader))
	if size := enc.DecodedLen(len(rawPayload)); size > bufsize {
		bufsize = size
	}
	buf := make([]byte, bufsize)

	b := buf[:enc.DecodedLen(len(rawHeader))]
	if n, err := enc.Decode(b, rawHeader); err != nil {
		return fmt.Errorf("cannot base64 decode header: %s", err)
	} else {
		b = b[:n]
	}
	var header Header
	if err := json.Unmarshal(bytes.TrimSpace(b), &header); err != nil {
		return fmt.Errorf("cannot JSON decode header: %s", err)
	}

	if header.Algorithm != s.Algorithm() {
		return ErrInvalidSigner
	}

	b = buf[:enc.DecodedLen(len(rawPayload))]
	if n, err := enc.Decode(b, rawPayload); err != nil {
		return fmt.Errorf("cannot base64 decode payload: %s", err)
	} else {
		b = b[:n]
	}

	beforeSign := token[:len(token)-len(chunks[2])-1]
	if err := s.Verify(chunks[2], beforeSign); err != nil {
		return err
	}

	var claims Claims
	if err := json.Unmarshal(b, &claims); err != nil {
		return fmt.Errorf("cannot base64 decode payload: %s", err)
	}
	now := time.Now()
	if claims.ExpirationTime != 0 && claims.ExpirationTime < now.Unix() {
		return ErrExpired
	}
	if claims.NotBefore != 0 && claims.NotBefore > now.Unix() {
		return ErrNotReady
	}

	if err := json.Unmarshal(b, &payload); err != nil {
		return fmt.Errorf("cannot base64 decode payload: %s", err)
	}

	return nil
}

var (
	ErrMalformedToken = errors.New("malformed token")
	ErrInvalidSigner  = errors.New("invalid signer algorithm")
	ErrExpired        = errors.New("expired")
	ErrNotReady       = errors.New("token not yet active")
)

// fixPadding return given base64 encoded string with padding characters added
// if necessary.
func fixPadding(b []byte) []byte {
	if n := len(b) % 4; n > 0 {
		return append(b, bytes.Repeat([]byte("="), 4-n)...)
	}
	return b
}

func copyb(b []byte) []byte {
	c := make([]byte, len(b), len(b)+4)
	copy(c, b)
	return c
}

var enc = base64.URLEncoding

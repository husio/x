package stamp

import (
	"testing"
	"time"
)

func TestVaultWithSigner(t *testing.T) {
	type payload struct {
		X string
	}

	now := time.Now()
	cases := map[string]struct {
		signers []signer
		payload *payload
		err     error
	}{
		"ok_one_signer": {
			signers: []signer{
				{name: "first", sig: &nsig{"a"}, validTill: now.Add(time.Minute)},
				{name: "second", sig: &nsig{"b"}, validTill: now.Add(time.Minute)},
				{name: "third", sig: &nsig{"c"}, validTill: now.Add(time.Minute)},
			},
			payload: &payload{"first"},
		},
		"expired_signer": {
			signers: []signer{
				{name: "a", sig: &nsig{"a"}, validTill: now.Add(-time.Minute)},
				{name: "b", sig: &nsig{"b"}, validTill: now.Add(-time.Minute)},
			},
			payload: &payload{"foobar"},
			err:     ErrNoSigner,
		},
	}

	for tname, tc := range cases {
		var v Vault

		var tokens [][]byte
		for i, s := range tc.signers {
			v.AddSigner(s.name, s.sig, s.validTill.Sub(now))
			token, err := v.Encode(tc.payload)
			if tc.err != err {
				t.Errorf("%s: encoding with %d signer want %v, got %v", tname, i, tc.err, err)
			} else {
				tokens = append(tokens, token)
			}
		}

		if tc.err != nil {
			continue
		}

		for i, token := range tokens {
			t.Logf("%s: %d token %s", tname, i, string(token))

			x := tc.payload.X
			if err := v.Decode(tc.payload, token); err != nil {
				t.Errorf("%s: cannot decode: %s", tname, err)
			}
			if tc.payload.X != x {
				t.Errorf("%s: want %q payload, got %q", x, tc.payload)
			}
		}
	}
}

func TestVaultWithVerifier(t *testing.T) {
	type payload struct {
		X string
	}

	now := time.Now()
	cases := map[string]struct {
		signers []signer
		payload *payload
		err     error
	}{
		"ok_one_signer": {
			signers: []signer{
				{name: "first", sig: &nsig{"a"}, validTill: now.Add(time.Minute)},
				{name: "second", sig: &nsig{"b"}, validTill: now.Add(time.Minute)},
				{name: "third", sig: &nsig{"c"}, validTill: now.Add(time.Minute)},
			},
			payload: &payload{"first"},
		},
	}

	for tname, tc := range cases {
		var v Vault
		var sigs Vault

		var tokens [][]byte
		for i, s := range tc.signers {
			v.AddVerifier(s.name, s.sig)
			sigs.AddSigner(s.name, s.sig, s.validTill.Sub(now))
			token, err := sigs.Encode(tc.payload)
			if err != nil {
				t.Errorf("%s: cannot create token with %d signer", tname, i)
				continue
			}
			tokens = append(tokens, token)
		}

		if tc.err != nil {
			continue
		}

		for i, token := range tokens {
			t.Logf("%s: %d token %s", tname, i, string(token))

			x := tc.payload.X
			if err := v.Decode(tc.payload, token); err != nil {
				t.Errorf("%s: cannot decode: %s", tname, err)
			}
			if tc.payload.X != x {
				t.Errorf("%s: want %q payload, got %q", x, tc.payload)
			}
		}
	}
}

type nsig struct {
	name string
}

func (ns *nsig) Algorithm() string {
	return ns.name
}

func (ns *nsig) Verify(signature, data []byte) error {
	if string(signature) != ("sig:" + ns.name) {
		return ErrInvalidSignature
	}
	return nil
}

func (ns *nsig) Sign(data []byte) ([]byte, error) {
	return []byte("sig:" + ns.name), nil
}

func BenchmarkVaultDecodeShort(b *testing.B) {
	var v Vault
	v.AddSigner("x", xSigner{}, time.Hour)

	var payload struct {
		IsAdmin bool `json:"isadm"`
	}
	token := []byte(`eyJhbGciOiJ4Iiwia2lkIjoieCJ9.eyJ1c2lkIjoxMjM0NSwiaXNhZG0iOnRydWV9.eA`)
	for i := 0; i < b.N; i++ {
		if err := v.Decode(&payload, token); err != nil {
			b.Fatalf("cannot decode: %s", err)
		}
	}
}

func BenchmarkVaultEncodeSmall(b *testing.B) {
	var v Vault
	v.AddSigner("x", xSigner{}, time.Hour)

	payload := struct {
		UserID  int64 `json:"usid"`
		IsAdmin bool  `json:"isadm"`
	}{
		IsAdmin: true,
		UserID:  12345,
	}
	for i := 0; i < b.N; i++ {
		if _, err := v.Encode(&payload); err != nil {
			b.Fatalf("cannot encode: %s", err)
		}
	}
}

// xSigner is noop signature builder that require signature to be equal to
// signed data.
type xSigner struct {
}

func (xSigner) Algorithm() string {
	return "x"
}

func (xSigner) Verify(signature, data []byte) error {
	if string(signature) != "x" {
		return ErrInvalidSignature
	}
	return nil
}

func (xSigner) Sign(data []byte) ([]byte, error) {
	return []byte("x"), nil
}

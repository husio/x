package stamp

import (
	"testing"
	"time"
)

func TestVault(t *testing.T) {
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
			v.Add(s.name, s.sig, s.validTill.Sub(now))
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

package stamp

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	cases := map[string]struct {
		s       Signer
		payload interface{}
		want    string
	}{
		"nop_sign_simple_data": {
			s:       nopSigner{},
			payload: map[string]string{"foo": "bar"},
			want:    `eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJmb28iOiJiYXIifQ.eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJmb28iOiJiYXIifQ`,
		},
		"nop_sign_complex_data": {
			s: nopSigner{},
			payload: map[string]interface{}{
				"s": "string",
				"i": 124,
				"a": []interface{}{"one", 2, "three"},
			},
			want: `eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0.eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0`,
		},
	}

	for name, tc := range cases {
		token, err := Encode(tc.s, tc.payload)

		if err != nil {
			t.Errorf("%s: cannot encode: %s", name, err)
			continue
		}
		want := []byte(tc.want)
		if !bytes.Equal(want, token) {
			t.Errorf("%s: want \n%q, got \n%q", name, want, token)
		}
	}
}

func TestExtendedClaims(t *testing.T) {
	type MyClaim struct {
		Claims
		UserID  int  `json:"user"`
		IsAdmin bool `json:"isadm"`
	}

	dt, _ := time.Parse("02 Jan 2006", "05 Dec 2014")
	payload := MyClaim{
		Claims: Claims{
			Subject:  "mysub",
			IssuedAt: dt.Unix(),
		},

		UserID:  321,
		IsAdmin: true,
	}

	token, err := Encode(nopSigner{}, &payload)
	if err != nil {
		t.Fatalf("cannot encode: %s", err)
	}

	expected := []byte(`eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJzdWIiOiJteXN1YiIsImlhdCI6MTQxNzczNzYwMCwidXNlciI6MzIxLCJpc2FkbSI6dHJ1ZX0.eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJzdWIiOiJteXN1YiIsImlhdCI6MTQxNzczNzYwMCwidXNlciI6MzIxLCJpc2FkbSI6dHJ1ZX0`)

	if !bytes.Equal(token, expected) {
		t.Fatalf("expected token to be \n%q\nbut got\n%q", expected, token)
	}

	var res MyClaim
	if err := Decode(nopSigner{}, &res, token); err != nil {
		t.Fatalf("cannot decode token: %s", err)
	}
	if !reflect.DeepEqual(res, payload) {
		t.Fatalf("x")
	}
}

func TestClaimExpirationTime(t *testing.T) {
	type MyPayload struct {
		Claims
		Admin bool `json:"isadm"`
	}
	now := time.Now()

	cases := map[string]struct {
		s       Signer
		payload MyPayload
		err     error
	}{
		"not_expired": {
			s: nopSigner{},
			payload: MyPayload{
				Claims: Claims{
					ExpirationTime: now.Add(2 * time.Minute).Unix(),
				},
			},
			err: nil,
		},
		"expired": {
			s: nopSigner{},
			payload: MyPayload{
				Claims: Claims{
					ExpirationTime: now.Add(-2 * time.Minute).Unix(),
				},
			},
			err: ErrExpired,
		},
		"already_active": {
			s: nopSigner{},
			payload: MyPayload{
				Claims: Claims{
					NotBefore: now.Add(-2 * time.Minute).Unix(),
				},
			},
			err: nil,
		},
		"not_yet_active": {
			s: nopSigner{},
			payload: MyPayload{
				Claims: Claims{
					NotBefore: now.Add(2 * time.Minute).Unix(),
				},
			},
			err: ErrNotReady,
		},
	}

	for name, tc := range cases {
		token, err := Encode(tc.s, tc.payload)
		if err != nil {
			t.Errorf("%s: cannot encode: %s", name, err)
			continue
		}
		if err := Decode(tc.s, &MyPayload{}, token); err != tc.err {
			t.Errorf("%s: want %v, got %v", name, tc.err, err)
		}
	}
}

/*
func TestDecode(t *testing.T) {
	cases := map[string]struct {
		s     Signer
		token string
		want  interface{}
	}{
		"nop_sign_simple_data": {
			s:     nopSigner{},
			token: `eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJmb28iOiJiYXIifQ.eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJmb28iOiJiYXIifQ`,
			want:  map[string]string{"foo": "bar"},
		},
		"nop_sign_complex_data": {
			s:     nopSigner{},
			token: `eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0.eyJ0eXAiOiJKV1QiLCJhbGciOiJub3AifQ.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0`,
			want: map[string]interface{}{
				"s": "string",
				"i": 124,
				"a": []interface{}{"one", 2, "three"},
			},
		},
	}

	for name, tc := range cases {
		data, err := Decode(tc.s, tc.token)

		if err != nil {
			t.Errorf("%s: cannot decode: %s", name, err)
			continue
		}
		want := []byte(tc.want)
		if !bytes.Equal(want, token) {
			t.Errorf("%s: want \n%q, got \n%q", name, want, token)
		}
	}
}
*/

// nopSigner is noop signature builder that require signature to be equal to
// signed data.
type nopSigner struct {
}

func (nopSigner) Algorithm() string {
	return "nop"
}

func (nopSigner) Verify(signature, data []byte) error {
	if !bytes.Equal(signature, data) {
		return ErrInvalidSignature
	}
	return nil
}

func (nopSigner) Sign(data []byte) ([]byte, error) {
	sig := make([]byte, len(data))
	copy(sig, data)
	return sig, nil
}

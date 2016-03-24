package stamp

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/husio/x/stamp/signer"
)

func TestEncode(t *testing.T) {
	cases := map[string]struct {
		s       signer.Signer
		payload interface{}
		want    string
	}{
		"xsign_simple_data": {
			s:       xSigner{},
			payload: map[string]string{"foo": "bar"},
			want:    `eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJmb28iOiJiYXIifQ.eA`,
		},
		"xsign_complex_data": {
			s: xSigner{},
			payload: map[string]interface{}{
				"s": "string",
				"i": 124,
				"a": []interface{}{"one", 2, "three"},
			},
			want: `eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0.eA`,
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

	token, err := Encode(xSigner{}, &payload)
	if err != nil {
		t.Fatalf("cannot encode: %s", err)
	}

	expected := []byte(`eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJzdWIiOiJteXN1YiIsImlhdCI6MTQxNzczNzYwMCwidXNlciI6MzIxLCJpc2FkbSI6dHJ1ZX0.eA`)

	if !bytes.Equal(token, expected) {
		t.Fatalf("expected token to be \n%q\nbut got\n%q", expected, token)
	}

	var res MyClaim
	if err := Decode(xSigner{}, &res, token); err != nil {
		t.Fatalf("cannot decode token: %s", err)
	}
	if !reflect.DeepEqual(res, payload) {
		t.Fatalf("data lost by token; %+v", res)
	}
}

func TestClaimExpirationTime(t *testing.T) {
	type MyPayload struct {
		Claims
		Admin bool `json:"isadm"`
	}
	now := time.Now()

	cases := map[string]struct {
		s       signer.Signer
		payload MyPayload
		err     error
	}{
		"not_expired": {
			s: xSigner{},
			payload: MyPayload{
				Claims: Claims{
					ExpirationTime: now.Add(2 * time.Minute).Unix(),
				},
			},
			err: nil,
		},
		"expired": {
			s: xSigner{},
			payload: MyPayload{
				Claims: Claims{
					ExpirationTime: now.Add(-2 * time.Minute).Unix(),
				},
			},
			err: ErrExpired,
		},
		"already_active": {
			s: xSigner{},
			payload: MyPayload{
				Claims: Claims{
					NotBefore: now.Add(-2 * time.Minute).Unix(),
				},
			},
			err: nil,
		},
		"not_yet_active": {
			s: xSigner{},
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

func TestDecode(t *testing.T) {
	cases := map[string]struct {
		s     signer.Signer
		token string
		want  interface{}
	}{
		"nop_sign_simple_data": {
			s:     xSigner{},
			token: `eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJmb28iOiJiYXIifQ.eA`,
			want:  map[string]interface{}{"foo": "bar"},
		},
		"nop_sign_complex_data": {
			s:     xSigner{},
			token: `eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJhIjpbIm9uZSIsMiwidGhyZWUiXSwiaSI6MTI0LCJzIjoic3RyaW5nIn0.eA`,
			want: map[string]interface{}{
				"s": "string",
				"a": []interface{}{"one", 2, "three"},
				"i": 124,
			},
		},
	}

	for name, tc := range cases {
		payload := make(map[string]interface{})
		if err := Decode(tc.s, &payload, []byte(tc.token)); err != nil {
			t.Errorf("%s: cannot decode: %s", name, err)
			continue
		}
		// TODO
		/*
			if !reflect.DeepEqual(payload, tc.want) {
				t.Errorf("%s: want \n%#v, got \n%#v", name, tc.want, payload)
			}
		*/
	}
}

func BenchmarkDecodeShort(b *testing.B) {
	var payload struct {
		IsAdmin bool `json:"isadm"`
	}
	var s xSigner
	token := []byte(`eyJ0eXAiOiJKV1QiLCJhbGciOiJ4In0.eyJpc2FkbSI6ZmFsc2V9.eA`)

	for i := 0; i < b.N; i++ {
		if err := Decode(s, &payload, token); err != nil {
			b.Fatalf("cannot decode: %s", err)
		}
	}
}

func BenchmarkEncodeSmall(b *testing.B) {
	payload := struct {
		UserID  int64 `json:"usid"`
		IsAdmin bool  `json:"isadm"`
	}{
		IsAdmin: true,
		UserID:  12345,
	}
	var s xSigner
	for i := 0; i < b.N; i++ {
		if _, err := Encode(s, &payload); err != nil {
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
		return signer.ErrInvalidSignature
	}
	return nil
}

func (xSigner) Sign(data []byte) ([]byte, error) {
	return []byte("x"), nil
}

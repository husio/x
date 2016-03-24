package signer

import (
	"bytes"
	"crypto/rsa"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

var updateFl = flag.Bool("update", false, "If provided, update golden data")

func TestSigners(t *testing.T) {
	const fixturePath = "fixtures/golden_signers.json"
	golden := loadGoldenData(t, fixturePath)

	if *updateFl {
		t.Logf("updaring gilden data fixture: %s", fixturePath)
		for _, g := range golden {
			g.Key = loadValue(t, g.Key)
			if b, err := g.Signer(t).Sign([]byte(g.Payload)); err != nil {
				t.Errorf("cannot create signature for %s: %s", g, err)
			} else {
				g.Signature = b
			}
		}
		if b, err := json.MarshalIndent(golden, "", "  "); err != nil {
			t.Fatalf("cannot write to golden file: %s", err)
		} else {
			ioutil.WriteFile(fixturePath, b, 0666)
		}
	}

	for _, g := range golden {
		g.Key = loadValue(t, g.Key)
		sig := g.Signer(t)
		if err := sig.Verify(g.Signature, []byte(g.Payload)); err != nil {
			t.Errorf("%s: invalid signature: %s", g, err)
		}

		b, err := sig.Sign([]byte(g.Payload))
		if err != nil {
			t.Errorf("%s: cannot compute signature: %s", g, err)
		} else if !bytes.Equal(b, g.Signature) {
			t.Errorf("%s: signature missmatch: %v != %v", g, b, g.Signature)
		}
	}
}

func loadGoldenData(t *testing.T, path string) []*signerFixture {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %q fixture: %s", path, err)
	}

	var fixtures []*signerFixture
	if err := json.Unmarshal(b, &fixtures); err != nil {
		t.Fatalf("cannot unmarshal %q fixture: %s", path, err)
	}
	return fixtures
}

type signerFixture struct {
	Type      string
	Key       string
	Payload   string
	Signature []byte
}

func (sf *signerFixture) Signer(t *testing.T) Signer {
	switch sf.Type {
	case "HS256":
		return NewHMAC256Signer([]byte(sf.Key))
	case "HS384":
		return NewHMAC384Signer([]byte(sf.Key))
	case "HS512":
		return NewHMAC512Signer([]byte(sf.Key))
	case "RS256":
		key := rsakey(t, strings.TrimSpace(sf.Key))
		return NewRSA256Signer(key)
	case "RS384":
		key := rsakey(t, strings.TrimSpace(sf.Key))
		return NewRSA384Signer(key)
	case "RS512":
		key := rsakey(t, strings.TrimSpace(sf.Key))
		return NewRSA512Signer(key)
	}

	t.Fatalf("unsupported signer type: %s", sf.Type)
	return nil
}

func (sf *signerFixture) String() string {
	payload := sf.Payload
	if len(payload) > 8 {
		payload = payload[:8]
	}
	return fmt.Sprintf("%s: %q", sf.Type, payload)
}

func nonempty(sts ...string) string {
	for _, s := range sts {
		if s != "" {
			return s
		}
	}
	return ""
}

func rsakey(t *testing.T, rawPriv string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(rawPriv))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		t.Fatalf("invalid private key: %s", rawPriv)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("cannot parse private key: %s", err)
	}
	return key
}

func loadValue(t *testing.T, s string) string {
	if len(s) == 0 || s[0] != '@' {
		return s
	}

	b, err := ioutil.ReadFile(s[1:])
	if err != nil {
		t.Fatalf("cannot load fixture %q: %s", s, err)
	}
	return string(b)
}

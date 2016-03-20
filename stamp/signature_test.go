package stamp

import (
	"bytes"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
)

var updateFl = flag.Bool("update", false, "If provided, update golden data")

func TestSigners(t *testing.T) {
	const fixturePath = "fixtures/golden_signers.json"
	golden := loadGoldenData(t, fixturePath)

	if *updateFl {
		t.Logf("updaring gilden data fixture: %s", fixturePath)
		for _, g := range golden {
			g.PubKey = nonempty(g.PubKey, g.PrivKey)
			g.PrivKey = nonempty(g.PrivKey, g.PubKey)
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
	PubKey    string
	PrivKey   string
	Payload   string
	Signature []byte
}

func (sf *signerFixture) Signer(t *testing.T) Signer {
	switch sf.Type {
	case "HS256":
		key := nonempty(sf.PrivKey, sf.PubKey)
		return NewHMAC256Signer([]byte(key))
	case "HS384":
		key := nonempty(sf.PrivKey, sf.PubKey)
		return NewHMAC384Signer([]byte(key))
	case "HS512":
		key := nonempty(sf.PrivKey, sf.PubKey)
		return NewHMAC512Signer([]byte(key))
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

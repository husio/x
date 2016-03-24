package signer

type Vault struct {
	signers map[string]Signer
}

func NewVault(signers map[string]Signer) *Vault {
	v := Vault{
		signers: make(map[string]Signer),
	}
	for method, s := range signers {
		v.signers[method] = s
	}
	return &v
}

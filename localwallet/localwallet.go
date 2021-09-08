package localwallet

import (
	"errors"
	"fmt"

	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/jsign/go-filsigner/wallet"
)

var (
	errPrivateKeyNotFound = errors.New("private key not found")
)

type Wallet struct {
	keys map[string]string
}

func New(pks []string) (*Wallet, error) {
	if len(pks) == 0 {
		return nil, fmt.Errorf("at least one private key should be provided")
	}
	keys := map[string]string{}
	for i := range pks {
		pubKey, err := wallet.PublicKey(pks[i])
		if err != nil {
			return nil, fmt.Errorf("get public key from private key: %s", err)
		}
		keys[pubKey.String()] = pks[i]
	}
	return &Wallet{
		keys: keys,
	}, nil
}

func (w *Wallet) Has(addr string) (bool, error) {
	_, ok := w.keys[addr]
	return ok, nil
}

func (w *Wallet) Sign(addr string, payload []byte) (*crypto.Signature, error) {
	pk, ok := w.keys[addr]
	if !ok {
		return nil, errPrivateKeyNotFound
	}

	sig, err := wallet.WalletSign(pk, payload)
	if err != nil {
		return nil, fmt.Errorf("signing payload: %s", err)
	}

	return sig, nil
}

func (w *Wallet) GetAddresses() []string {
	res := make([]string, 0, len(w.keys))
	for addr := range w.keys {
		res = append(res, addr)
	}
	return res
}

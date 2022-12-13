package gpg

import (
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/pkg/errors"
)

type PublicKey string
type PrivateKey string

func generateFullGPGIdentity(email string) (PublicKey, PrivateKey, error) {
	ecKey, err := crypto.GenerateKey(email, email, "x25519", 0)
	if err != nil {
		return "", "", errors.Wrap(err, "cannot generate key pairs")
	}
	privateKey, privateKeyErr := ecKey.Armor()
	if privateKeyErr != nil {
		return "", "", errors.Wrap(privateKeyErr, "cannot armor private key")
	}
	pubKey, pubKeyErr := ecKey.GetArmoredPublicKey()
	if pubKeyErr != nil {
		return "", "", errors.Wrap(pubKeyErr, "cannot armor public key")
	}
	return PublicKey(pubKey), PrivateKey(privateKey), nil
}

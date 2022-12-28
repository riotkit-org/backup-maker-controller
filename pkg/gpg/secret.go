package gpg

import (
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateNewGPGSecret(name string, namespace string, email string, owners []metav1.OwnerReference, spec *v1alpha1.GPGKeySecretSpec) (*v1.Secret, error) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"riotkit.org/type": "GPGSecret",
			},
			Annotations: map[string]string{
				"riotkit.org/e-mail": email,
			},
			OwnerReferences: owners,
		},

		// filled up later by UpdateGPGSecretWithRecreatedGPGKey()
		StringData: map[string]string{},
	}

	if _, err := UpdateGPGSecretWithRecreatedGPGKey(secret, spec, email, true); err != nil {
		return secret, errors.Wrap(err, "cannot populate secret with new generated key pair")
	}

	return secret, nil
}

func UpdateGPGSecretWithRecreatedGPGKey(secret *v1.Secret, spec *v1alpha1.GPGKeySecretSpec, email string, force bool) (bool, error) {
	if secret.Data == nil {
		secret.Data = make(map[string][]byte, 0)
	}
	secret.Data[spec.GetEmailIndex()] = []byte(email)

	if !ShouldUpdate(secret, spec) && !force {
		logrus.Info("Secret does not need an update")
		return false, nil
	}

	logrus.Info("Generating a new GPG identity for an update")
	pubKey, privateKey, err := generateFullGPGIdentity(email)
	if err != nil {
		return false, errors.Wrap(err, "cannot generate a new identity")
	}

	secret.Data[spec.GetPassphraseIndex()] = []byte("")
	secret.Data[spec.GetPublicKeyIndex()] = []byte(pubKey)
	secret.Data[spec.GetPrivateKeyIndex()] = []byte(privateKey)

	return true, nil
}

func ShouldUpdate(secret *v1.Secret, spec *v1alpha1.GPGKeySecretSpec) bool {
	d := secret.Data
	if d == nil {
		d = make(map[string][]byte, 0)
	}

	// mix to be sure that the secret does not contain that key
	if secret.StringData != nil {
		for k, v := range secret.StringData {
			d[k] = []byte(v)
		}
	}

	// if any of those keys is missing in .data/.stringData, then we generate a full GPG identity from scratch
	indexes := []string{
		spec.GetPrivateKeyIndex(),
		spec.GetPublicKeyIndex(),
		// spec.GetPassphraseIndex(),  // DO NOT ADD: can be empty
		// spec.GetEmailIndex(),       // DO NOT ADD: email is always set
	}

	for _, indexName := range indexes {
		if val, exists := d[indexName]; !exists || len(val) == 0 {
			return true
		}
	}

	return false
}

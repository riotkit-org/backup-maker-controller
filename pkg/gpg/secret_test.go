package gpg

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreateNewGPGSecret_BasicPositivePath(t *testing.T) {
	spec := v1alpha1.GPGKeySecretSpec{
		SecretName:        "antifa",
		PublicKey:         "key.pub",
		PrivateKey:        "key",
		PassphraseKey:     "passphrase",
		Email:             "e-mail.txt",
		CreateIfNotExists: true,
	}

	secret, err := CreateNewGPGSecret("antifa-czech-backup-secret", "antifa", "antifa@antifa.cz", []metav1.OwnerReference{}, &spec)

	assert.Nil(t, err)

	// contains all the necessary files
	assert.Contains(t, secret.StringData["key"], "-----BEGIN PGP PRIVATE KEY BLOCK-----")
	assert.Contains(t, secret.StringData["key.pub"], "-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.Contains(t, secret.StringData["e-mail.txt"], "antifa@antifa.cz")

	// contains labels and annotations
	assert.Equal(t, secret.Labels["riotkit.org/type"], "GPGSecret")
	assert.Equal(t, secret.Annotations["riotkit.org/e-mail"], "antifa@antifa.cz")
}

func TestExistingSecretIsFilledUpWithNewIdentityIfKeysAreMissing(t *testing.T) {
	spec := v1alpha1.GPGKeySecretSpec{
		SecretName:        "antifa",
		PublicKey:         "key.pub",
		PrivateKey:        "key",
		PassphraseKey:     "passphrase",
		Email:             "e-mail.txt",
		CreateIfNotExists: true,
	}

	existingSecret := v1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Immutable:  nil,
		Data:       nil,
		StringData: map[string]string{
			"public-key": "something", // this one does not match "key.pub"
			"key":        "some",
			"passphrase": "thing",
			"email":      "just an e-mail",
		},
		Type: "",
	}

	err := UpdateGPGSecretWithRecreatedGPGKey(&existingSecret, &spec, "antifa@antifa.cz", false)
	assert.Nil(t, err)
	assert.Contains(t, existingSecret.StringData["key"], "-----BEGIN PGP PRIVATE KEY BLOCK-----")
	assert.Contains(t, existingSecret.StringData["key.pub"], "-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.Contains(t, existingSecret.StringData["e-mail.txt"], "antifa@antifa.cz")
}

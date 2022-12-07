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
	assert.Contains(t, string(secret.Data["key"]), "-----BEGIN PGP PRIVATE KEY BLOCK-----")
	assert.Contains(t, string(secret.Data["key.pub"]), "-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.Contains(t, string(secret.Data["e-mail.txt"]), "antifa@antifa.cz")

	// contains labels and annotations
	assert.Equal(t, secret.Labels["riotkit.org/type"], "GPGSecret")
	assert.Equal(t, secret.Annotations["riotkit.org/e-mail"], "antifa@antifa.cz")
}

func TestExistingSecretIsFilledUpWithNewIdentityIfKeysAreMissing(t *testing.T) {
	spec := v1alpha1.GPGKeySecretSpec{
		SecretName:        "antifa",
		PublicKey:         "keyfile.pub",
		PrivateKey:        "keyfile",
		PassphraseKey:     "passphrase",
		Email:             "e-mail.txt",
		CreateIfNotExists: true,
	}

	testData := []map[string]string{
		{
			"public-key": "something", // this one does not match "key.pub"
			"key":        "some",
			"passphrase": "thing",
			"email":      "just an e-mail",
		},
		{
			"key.pub":    "something",
			"key.priv":   "some", // this one does not match "key"
			"passphrase": "thing",
			"email":      "just an e-mail",
		},
		// empty public key
		{
			"key.pub":    "", // empty
			"key":        "some",
			"passphrase": "thing",
			"email":      "just an e-mail",
		},
		// empty private key
		{
			"key.pub":    "non-empty",
			"key":        "", // empty
			"passphrase": "thing",
			"email":      "just an e-mail",
		},
	}

	for _, testCase := range testData {
		existingSecret := v1.Secret{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Immutable:  nil,
			Data:       nil,
			StringData: testCase,
			Type:       "",
		}

		err := UpdateGPGSecretWithRecreatedGPGKey(&existingSecret, &spec, "antifa@antifa.cz", false)
		assert.Nil(t, err)
		assert.Contains(t, string(existingSecret.Data["keyfile"]), "-----BEGIN PGP PRIVATE KEY BLOCK-----")
		assert.Contains(t, string(existingSecret.Data["keyfile.pub"]), "-----BEGIN PGP PUBLIC KEY BLOCK-----")
		assert.Contains(t, string(existingSecret.Data["e-mail.txt"]), "antifa@antifa.cz")
	}
}

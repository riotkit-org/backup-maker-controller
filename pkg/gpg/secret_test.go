package gpg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateNewGPGSecret_BasicPositivePath(t *testing.T) {
	secret, err := CreateNewGPGSecret("antifa-czech-backup-secret", "antifa", "antifa@antifa.cz", nil)

	assert.Nil(t, err)

	// contains all the necessary files
	assert.Contains(t, secret.StringData["key"], "-----BEGIN PGP PRIVATE KEY BLOCK-----")
	assert.Contains(t, secret.StringData["key.pub"], "-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.Contains(t, secret.StringData["e-mail.txt"], "antifa@antifa.cz")

	// contains labels and annotations
	assert.Equal(t, secret.Labels["riotkit.org/type"], "GPGSecret")
	assert.Equal(t, secret.Annotations["riotkit.org/e-mail"], "antifa@antifa.cz")
}

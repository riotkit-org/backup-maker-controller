package gpg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateFullGPGIdentity(t *testing.T) {
	pub, private, err := generateFullGPGIdentity("example@iwa-ait.org")
	assert.Nil(t, err)

	assert.Contains(t, private, "-----BEGIN PGP PRIVATE KEY BLOCK-----")
	assert.Contains(t, private, "-----END PGP PRIVATE KEY BLOCK-----")

	assert.Contains(t, pub, "-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.Contains(t, pub, "-----END PGP PUBLIC KEY BLOCK-----")
}

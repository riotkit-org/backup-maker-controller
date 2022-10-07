package gpg

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateNewGPGSecret(name string, namespace string, email string, owners []metav1.OwnerReference) (*v1.Secret, error) {
	pubKey, privateKey, err := generateFullGPGIdentity(email)
	if err != nil {
		return &v1.Secret{}, errors.Wrap(err, "cannot generate a new GPG identity")
	}

	return &v1.Secret{
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
		StringData: map[string]string{
			"key":        string(privateKey),
			"key.pub":    string(pubKey),
			"e-mail.txt": email,
		},
	}, nil
}

package factory

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-controller/pkg/domain"
	"github.com/riotkit-org/backup-maker-controller/pkg/gpg"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrorActionRequeue = errors.New("REQUEUE")

type Factory struct {
	client.Client
	fetcher CachedFetcher
	logger  *logrus.Entry
}

func NewFactory(client client.Client, fetcher CachedFetcher, logger *logrus.Entry) *Factory {
	return &Factory{Client: client, fetcher: fetcher, logger: logger}
}

// CreateScheduledBackupAggregate is creating a fully hydrated object (aggregate) with all dependencies inside
func (c *Factory) CreateScheduledBackupAggregate(ctx context.Context, backup *v1alpha1.ScheduledBackup, operation string) (*domain.ScheduledBackupAggregate, error, error) {
	aggregate := domain.ScheduledBackupAggregate{ScheduledBackup: backup}
	aggregate.AdditionalVarsList = make(map[string][]byte)

	if err := c.hydrateGPGSecret(ctx, &aggregate); err != nil {
		return &aggregate, ErrorActionRequeue, err
	}
	if err := c.hydrateTemplate(ctx, &aggregate); err != nil {
		return &aggregate, ErrorActionRequeue, err
	}
	if err := c.hydrateAccessToken(ctx, &aggregate); err != nil {
		return &aggregate, ErrorActionRequeue, err
	}
	if err := c.hydrateVarsSecret(ctx, &aggregate); err != nil {
		return &aggregate, ErrorActionRequeue, err
	}

	//
	// .spec.tokenSecretRef: Extract access token from `kind: Secret` and put into the .spec.vars.Repository.token
	//
	if aggregate.TokenSecret != nil && aggregate.TokenSecret.Data != nil {
		key := backup.Spec.TokenSecretRef.TokenKey
		if val, exists := aggregate.TokenSecret.Data[key]; exists && string(val) != "" {
			aggregate.AdditionalVarsList["Repository.token"] = val
		}
	}

	//
	// .Values.gpgKeyContent is a source of GPG key (public or private - depends on the operation)
	//
	if operation == "" {
		operation = aggregate.Spec.Operation
	}
	if operation == "backup" {
		aggregate.AdditionalVarsList["HelmValues.gpgKeyContent"] = aggregate.GPGSecret.Data[aggregate.Spec.GPGKeySecretRef.PublicKey]
	} else {
		aggregate.AdditionalVarsList["HelmValues.gpgKeyContent"] = aggregate.GPGSecret.Data[aggregate.Spec.GPGKeySecretRef.PrivateKey]
	}

	return &aggregate, nil, nil
}

// CreateRequestedBackupActionAggregate is creating a fully hydrated object (aggregate) with all dependencies inside
func (c *Factory) CreateRequestedBackupActionAggregate(ctx context.Context, action *v1alpha1.RequestedBackupAction, scheduledBackup *v1alpha1.ScheduledBackup) (*domain.RequestedBackupActionAggregate, error, error) {
	scheduledBackupAggregate, _, fetchErr := c.CreateScheduledBackupAggregate(ctx, scheduledBackup, action.Spec.Action)
	a := domain.NewRequestedBackupActionAggregate(action, scheduledBackupAggregate)
	if fetchErr != nil {
		return a, ErrorActionRequeue, fetchErr
	}

	// modify it according to the current ACTION
	scheduledBackup.Spec.CronJob.Enabled = false        // we cannot generate a CronJob in this case :-)
	scheduledBackup.Spec.Operation = action.Spec.Action // we should enforce an action as RequestedBackupAction is a manual TRIGGER for ScheduledBackup

	return a, nil, nil
}

// GPG secrets [Secret]
//
//	This secret can be automatically generated when: .spec.gpgKeySecretRef.createIfNotExists == "true"
//	NOTICE: Backup of this key is on your side. Better approach is to generate it by your own and use e.g. SealedSecrets to keep in GIT
//	        or to fetch it with kubectl, encrypt and store in the repository
func (c *Factory) hydrateGPGSecret(ctx context.Context, a *domain.ScheduledBackupAggregate) error {
	secret, gpgErr := c.fetcher.fetchSecret(ctx, a.Spec.GPGKeySecretRef.SecretName, a.Namespace)

	//
	// Create new if Secret does not exist at all
	//
	if apierrors.IsNotFound(gpgErr) {
		c.logger.Info("No GPG secret found")

		if !a.Spec.GPGKeySecretRef.CreateIfNotExists {
			c.logger.Info("Referenced secret does not exist, .spec.gpgKeySecretRef.createIfNotExists is set to false, waiting for a secret")
			return errors.Wrap(gpgErr, "cannot fetch GPG containing Secret")
		} else {
			c.logger.Info("Creating a new GPG key pair and storing as a Secret. Notice: Copy that Secret, encrypt it and put into your GIT repository. If you loose the keys you will not restore backups")
			secret, gpgErr = gpg.CreateNewGPGSecret(
				a.Spec.GPGKeySecretRef.SecretName,
				a.Namespace,
				a.Spec.GPGKeySecretRef.Email,
				[]metav1.OwnerReference{
					{APIVersion: "v1alpha1", Kind: "ScheduledBackup", Name: a.Name, UID: a.UID},
				},
				&a.Spec.GPGKeySecretRef,
			)
			if err := c.Client.Create(ctx, secret); err != nil {
				c.logger.Error(err, "cannot apply a Kubernetes secret for generated GPG key, will try again")
				return errors.Wrap(err, "cannot apply a Secret to Kubernetes")
			}
		}

		//
		// Update existing Secret
		//
	} else if a.Spec.GPGKeySecretRef.CreateIfNotExists {
		c.logger.Info("Updating existing GPG secret if necessary")

		if gpg.ShouldUpdate(secret, &a.Spec.GPGKeySecretRef) {
			// fetch a fresh secret to avoid: "the object has been modified; please apply your changes to the latest version and try again"
			fetchErr := c.Client.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, secret)
			if fetchErr != nil {
				return errors.Wrapf(fetchErr, "cannot fetch existing secret from API - %s/%s", secret.Name, secret.Namespace)
			}

			// Update existing Secret with new GPG identity, in case it is incorrectly formatted or missing
			updated, err := gpg.UpdateGPGSecretWithRecreatedGPGKey(secret, &a.Spec.GPGKeySecretRef, a.Spec.GPGKeySecretRef.Email, false)
			if err != nil {
				return errors.Wrap(err, "cannot update existing secret with new identity (existing secret was missing specified keys in .data/.stringData section)")
			}
			if updated {
				if err := c.Client.Update(ctx, secret); err != nil {
					return errors.Wrap(err, "cannot append GPG identity to the secret")
				}
			}
		}
	}
	a.GPGSecret = secret
	return nil
}

// Fetch an associated template [ScheduledBackup]
func (c *Factory) hydrateTemplate(ctx context.Context, a *domain.ScheduledBackupAggregate) error {
	//
	// InternalTemplate allows to use templates bundled with Backup Maker Generator
	//
	if a.ScheduledBackup.Spec.TemplateRef.Kind == "internal" {
		a.Template = domain.InternalTemplate{Name: a.ScheduledBackup.Spec.TemplateRef.Name}
		return nil
	}

	//
	// Cluster Templates are defined as CRD in Kubernetes
	//
	tpl, tplErr := c.fetcher.fetchTemplate(ctx, a.ScheduledBackup)
	if tplErr != nil {
		return errors.Wrap(tplErr, "cannot fetch ClusterBackupProcedureTemplate type object")
	}
	a.Template = tpl
	c.logger.Info(fmt.Sprintf("Fetched '%s' template", tpl.Name))
	return nil
}

// Token: Access token with access to the Backup Repository server instance
func (c *Factory) hydrateAccessToken(ctx context.Context, a *domain.ScheduledBackupAggregate) error {
	tokenSecret, tokenErr := c.fetcher.fetchSecret(ctx, a.Spec.TokenSecretRef.SecretName, a.Namespace)
	if tokenErr != nil {
		return errors.Wrap(tokenErr, "cannot fetch access token Secret (access token to access Backup Repository server)")
	}

	a.TokenSecret = tokenSecret
	return nil
}

// Vars from Secret (optional)
func (c *Factory) hydrateVarsSecret(ctx context.Context, a *domain.ScheduledBackupAggregate) error {
	if a.Spec.VarsSecretRef.SecretName != "" {
		varsSecret, varsSecretErr := c.fetcher.fetchSecret(ctx, a.Spec.VarsSecretRef.SecretName, a.Namespace)
		if varsSecretErr != nil {
			return errors.Wrap(varsSecretErr, "warning - cannot find Secret containing vars - but secret name was referenced, will try again")
		}
		a.VarsListSecret = varsSecret
	}
	return nil
}

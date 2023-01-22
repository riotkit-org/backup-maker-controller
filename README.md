backup-maker-controller
=======================

Part of Backup Repository ecosystem. Operates Kubernetes Jobs performing automated backup & restore procedures.

### Roadmap

#### v0.1

- [x] `ScheduledBackup` CRD for defining backup parameters and optionally scheduling a `CronJob`
- [x] `RequestedBackupAction` CRD for immediately triggering a backup or restore on demand
- [x] [Helm Chart](./charts/backup-maker-controller)
- [x] Support for reporting `kind: Job` status to the main CRD's `.status` field
- [x] [Safe concurrency with locking mechanism using in-memory and distributed Redis lockers. Thanks to this multiple sub-controllers will not process in parallel the same resource which reduces compute complexity and saves lots of resources](./pkg/locking)
- [x] Support for internal _Backup Maker_ templates, so the templates would not be necessary to be redefined as `ClusterBackupProcedureTemplate` CRD

#### v1.0

- [ ] Watch status of CronJobs and it's last executions, report to `.status` field

#### v1.1

- [ ] Integration with ArgoCD to see correct health status
- [ ] Integration with Argo Workflows to use Workflows instead of Kubernetes Jobs

### Getting started

Install controller using Helm

//todo: instruction with CRD inside Helm and CRD installed manually as manifests

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

#### ScheduledBackup

Controller is watching `ScheduledBackups` and creating `Secret`, `ConfigMap` and `CronJob` type resources. It makes sure all dependencies are there for the backup and restore procedures to happen.

**Rules:**
- When there is no GPG key created, it can create it and store as `Secret`
- Can create `CronJob` optionally. When `CronJob` is disabled, then `ScheduledBackup` acts as a parent to `RequestedBackupAction` for manually triggered actions

**Example reference:**

```yaml
---
apiVersion: riotkit.org/v1alpha1
kind: ScheduledBackup
metadata:
    name: app1
    namespace: default
spec:
    # Operation that will be scheduled: backup or restore
    operation: backup

    # Is this one time operation or a scheduled one?
    cronJob:
        enabled: true
        scheduleEvery: "00 02 * * *"

    # Collection ID is an unique identifier for the Backup Collection at server side
    # Read more about the concept there: https://github.com/riotkit-org/backup-repository/blob/main/docs/api/collections/README.md
    collectionId: 1111-2222-3333-444465

    # GPG keys needs to be stored in a separate Secret placed in same namespace
    # Those keys are needed to encrypt and decrypt backups
    gpgKeySecretRef:
        createIfNotExists: true
        email: example@example.org
        passphraseKey: passphrase
        privateKey: private
        publicKey: public
        secretName: backup-keys

    # Access token (JWT) to access the Backup Repository server
    tokenSecretRef:
        secretName: backup-keys
        tokenKey: token

    # Backup scripts placed as a template
    # Those scripts will run inside Job/CronJob as your backup/restore procedure
    templateRef:
        kind: ClusterBackupProcedureTemplate
        name: pg13

    # Input variables that will go into the template
    # See example templates: https://github.com/riotkit-org/br-backup-maker/tree/main/generate/templates/backup
    vars: |
        # System-specific variables, in this case specific to PostgreSQL
        # ${...} and $(...) syntax will be evaluated in target environment e.g. Kubernetes POD
        Params:
          hostname: postgres-postgresql.backup-repository.svc.cluster.local
          port: 5432
          db: backup-repository
          user: riotkit
          password: "putinchuj" # injects a shell-syntax, put your password in a `kind: Secret` and mount as environment variable. You can also use $(cat /mnt/secret) syntax, be aware of newlines!
        
        # Generic repository access details. Everything here will land AS IS into the bash script.
        # This means that any ${...} and $(...) will be executed in target environment e.g. inside Kubernetes POD
        Repository:
          url: "http://my-example.org"
          token: "xxxxxx-yyyy-zzzz" # Is ignored and overwritten if using .spec.tokenSecretRef
          passphrase: "riotkit"
          recipient: "test@riotkit.org"
          collectionId: "iwa-ait"
        
        # Generic values for Helm used to generate jobs/pods. Those values will overwrite others.
        # Notice: Environment variables with '${...}' and '$(...)' will be evaluated in LOCAL SHELL DURING BUILD
        HelmValues:
          env: { }
            # if specified, then will be added to `kind: Secret` and injected into POD as environment
            # the value from ${GPG_PASSPHRASE} will be retrieved from the SHELL DURING THE BUILD
            #GPG_PASSPHRASE: "${GPG_PASSPHRASE}"
        
          # most secure way for Kubernetes is to not provide secrets there, but define them as environment variables
          # inside SealedSecrets - all encryptedData keys will be accessible as environment variables inside container

    # Imports secrets from Kubernetes Secret, those secrets will cover the keys in "vars"
    # so you can hide sensitive data
    varsSecretRef:
        importOnlyKeys:
            - Params.password
        secretName: backup-keys

```

#### RequestedBackupAction

Spawns `Jobs` instantly to perform a `backup` or `restore` action.

**Rules:**
- Manages only `Jobs`. All other resources like `ConfigMaps`, `Secrets` are managed by `ScheduledBackup`
- All spawned `Jobs` are watched and its status is reported to the `.status` field of the `RequestedBackupAction`
- Requires `ScheduledBackup` to be defined to refer to

**Example reference:**

```yaml
---
apiVersion: riotkit.org/v1alpha1
kind: RequestedBackupAction
metadata:
    generateName: app1-backup-
    namespace: default
spec:
    action: backup
    scheduledBackupRef:
        name: app1
```

### FAQ

1. Map has no entry for key password

```
Cannot template or apply objects to the cluster: cannot apply rendered
        objects to the cluster: error while generating manifests: cannot render
        template, execution failed. Error: template: pg14:7:46: executing "pg14"
        at <.Params.password>: map has no entry for key "password"
```

Check that your `kind: Secret` referenced in ScheduledBackup contains a key "Params.password", the reference is defined at this section:

```yaml
# (...)
varsSecretRef:
    importOnlyKeys:
      - Params.password
    secretName: postgres-test-1
```

### License

Copyright 2022 Riotkit.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

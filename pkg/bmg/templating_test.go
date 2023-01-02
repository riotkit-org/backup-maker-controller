package bmg

import (
	"context"
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-controller/pkg/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

// TestWriteTemplate is checking if the template is written, when using ClusterBackupProcedureTemplate
func TestWriteTemplate_ClusterBackupProcedureTemplate(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "br-fdi")
	if err != nil {
		logrus.Fatal(err)
	}

	tpl := v1alpha1.ClusterBackupProcedureTemplate{
		TypeMeta:   v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{},
		Spec: v1alpha1.ClusterBackupProcedureTemplateSpec{
			Image:   "putin:is-a-dickbag",
			Backup:  "#!/bin/bash\necho 'Hello backup'",
			Restore: "#!/bin/bash\necho 'Hello restore'",
		},
	}

	//
	// ASSERT: operation == backup
	//
	writeErr := writeTemplate(logrus.WithContext(context.TODO()), &tpl, domain.Backup, dir+"/backup/redis.sh")
	assert.Nil(t, writeErr)
	bc, _ := os.ReadFile(dir + "/backup/redis.sh")
	assert.Equal(t, string(bc), "#!/bin/bash\necho 'Hello backup'")

	//
	// ASSERT operation == restore
	//
	writeErr = writeTemplate(logrus.WithContext(context.TODO()), &tpl, domain.Restore, dir+"/restore/redis.sh")
	assert.Nil(t, writeErr)
	bc, _ = os.ReadFile(dir + "/restore/redis.sh")
	assert.Equal(t, string(bc), "#!/bin/bash\necho 'Hello restore'")
}

// TestWriteTemplate_InternalTemplate is checking that NO ANY FILE is written, when the input template is of InternalTemplate type
func TestWriteTemplate_InternalTemplate(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "br-fdi")
	if err != nil {
		logrus.Fatal(err)
	}

	tpl := domain.InternalTemplate{Name: "sqs"}
	writeErr := writeTemplate(logrus.WithContext(context.TODO()), &tpl, domain.Backup, dir+"/backup/sqs.sh")
	assert.Nil(t, writeErr)

	// The file should not exist
	_, existsErr := os.Stat(dir + "/backup/sqs.sh")
	assert.NotNil(t, existsErr)
	assert.Contains(t, existsErr.Error(), "no such file or directory")
}

func TestParseRenderedManifests_FiltersOutKinds_RBACase(t *testing.T) {
	content := `---
apiVersion: v1
kind: ConfigMap
metadata:
    name: app1-backup
data:
    hello: world
    goodbye: capitalism

---
apiVersion: batch/v1
kind: Job
metadata:
    name: app1-backup
`

	objects, err := parseRenderedManifests(logrus.WithContext(context.TODO()), content,
		domain.NewResourceTypesFilterForRequestedBackupAction())

	assert.Len(t, objects, 1)
	assert.Equal(t, "Job", objects[0].GroupVersionKind().Kind)
	assert.Nil(t, err)
}

func TestParseRenderedManifests_FiltersOutKinds_SBCase(t *testing.T) {
	content := `---
apiVersion: v1
kind: ConfigMap
metadata:
    name: app1-backup
data:
    hello: world
    goodbye: capitalism

---
apiVersion: v1
kind: Secret
metadata:
    name: app1-backup

---
apiVersion: batch/v1
kind: Job
metadata:
    name: app1-backup

---
apiVersion: batch/v1
kind: CronJob
metadata:
    name: app1-backup
`

	objects, err := parseRenderedManifests(logrus.WithContext(context.TODO()), content,
		domain.NewResourceTypesFilterForScheduledBackup(
			&domain.ScheduledBackupAggregate{
				ScheduledBackup: &v1alpha1.ScheduledBackup{
					Spec: v1alpha1.ScheduledBackupSpec{
						Operation: "backup",
						CronJob: v1alpha1.CronJobSpec{
							Enabled:       false,
							ScheduleEvery: "",
						},
					},
				},
			},
			domain.Backup,
		))

	assert.Len(t, objects, 2)
	assert.Equal(t, "ConfigMap", objects[0].GroupVersionKind().Kind)
	assert.Equal(t, "Secret", objects[1].GroupVersionKind().Kind)
	assert.Nil(t, err)
}

func TestParseRenderedManifests_FiltersOutKinds_SBCase_WithCronJob(t *testing.T) {
	content := `---
apiVersion: v1
kind: ConfigMap
metadata:
    name: app1-backup
data:
    hello: world
    goodbye: capitalism

---
apiVersion: v1
kind: Secret
metadata:
    name: app1-backup

---
apiVersion: batch/v1
kind: Job
metadata:
    name: app1-backup

---
apiVersion: batch/v1
kind: CronJob
metadata:
    name: app1-backup
`

	objects, err := parseRenderedManifests(logrus.WithContext(context.TODO()), content,
		domain.NewResourceTypesFilterForScheduledBackup(
			&domain.ScheduledBackupAggregate{
				ScheduledBackup: &v1alpha1.ScheduledBackup{
					Spec: v1alpha1.ScheduledBackupSpec{
						Operation: "backup",
						CronJob: v1alpha1.CronJobSpec{
							Enabled:       true, // NOTICE HERE: CronJob is enabled
							ScheduleEvery: "*/3 * * * *",
						},
					},
				},
			},
			domain.Backup,
		))

	assert.Len(t, objects, 3)
	assert.Equal(t, "ConfigMap", objects[0].GroupVersionKind().Kind)
	assert.Equal(t, "Secret", objects[1].GroupVersionKind().Kind)
	assert.Equal(t, "CronJob", objects[2].GroupVersionKind().Kind)
	assert.Nil(t, err)
}

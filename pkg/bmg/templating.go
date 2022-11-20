package bmg

import (
	"fmt"
	"github.com/ohler55/ojg/jp"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/domain"
	"github.com/riotkit-org/br-backup-maker/generate"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"os"
	"path/filepath"
	"strings"
)

// RenderKubernetesResourcesFor is rendering Kubernetes resources like CronJob, Job, Secret, ConfigMap using Backup Maker Generator (BMG), which is using Helm under the hood
func RenderKubernetesResourcesFor(logger *logrus.Entry, backup domain.Renderable) ([]unstructured.Unstructured, error) {
	operation := backup.GetOperation()

	// Create a temporary workspace directory
	pwd, _ := os.Getwd()
	defer func(dir string) {
		_ = os.Chdir(dir)
	}(pwd)

	dir, tempDirErr := os.MkdirTemp("/tmp", "bmg")
	if tempDirErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(tempDirErr, "cannot create a temporary directory to run Backup Maker Generator (BMG)")
	}
	_ = os.MkdirAll(dir+"/output", 0755)

	// Delete temporary directory after the generating ends
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Write backup/restore procedure template
	templatePath := "./templates/backup/" + backup.GetTemplate().Name + ".tmpl"
	if writeErr := writeTemplate(backup.GetTemplate(), operation, templatePath); writeErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(writeErr, fmt.Sprintf("cannot write template at path '%s'", templatePath))
	}

	// Write GPG to desired temporary path
	gpgPath := dir + "/gpg.key"
	if writeErr := writeGPGKey(backup.GetBackupAggregate(), gpgPath, operation); writeErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(writeErr, "cannot write GPG key to temporary file")
	}

	// Write vars into definition.yaml
	definitionPath := dir + "/definition.yaml"
	if writeErr := writeDefinition(logger, backup.GetBackupAggregate(), definitionPath); writeErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(writeErr, "cannot write definition.yaml to temporary file")
	}

	// Run BMG to generate YAML manifests
	if err := generate.ExtractRequiredResources(); err != nil {
		return []unstructured.Unstructured{}, errors.Wrap(err, "cannot populate ~/.bm with templates")
	}
	cmd := generate.SnippetGenerationCommand{
		Template:       backup.GetTemplate().Name,
		DefinitionFile: definitionPath,
		IsKubernetes:   true,
		KeyPath:        gpgPath,
		OutputDir:      dir + "/output",
		Schedule:       backup.GetScheduledBackup().Spec.CronJob.ScheduleEvery,
		JobName:        backup.GetScheduledBackup().Name,
		Image:          backup.GetTemplate().Spec.Image,
		Operation:      operation,
		Namespace:      backup.GetScheduledBackup().Namespace,
	}
	genErr := cmd.Run()
	if genErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(genErr, "error while generating manifests")
	}
	return readRenderedManifests(logger, dir+"/output/"+operation+".yaml", backup.AcceptedResourceTypes())
}

// writeTemplate is writing the backup/restore procedure template
func writeTemplate(template *v1alpha1.ClusterBackupProcedureTemplate, operation string, path string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0700)

	content := template.Spec.Restore
	// todo: enum
	if operation == "backup" {
		content = template.Spec.Backup
	}
	return os.WriteFile(path, []byte(content), 0700)
}

// readRenderedManifests is reading manifests from YAML into []UnstructuredObject
func readRenderedManifests(logger *logrus.Entry, manifestPath string, kindsToRender []v1.GroupVersionKind) ([]unstructured.Unstructured, error) {
	// reading
	content, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(readErr, fmt.Sprintf(
			"cannot read rendered manifest file at path '%s'", manifestPath))
	}

	decoder := apiyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// parsing
	var objects []unstructured.Unstructured
	docsContent := strings.Split(string(content), "---\n")

	for _, doc := range docsContent {
		doc = strings.Trim(strings.Replace(doc, "---\n", "", 1), " \n")
		// empty document
		if len(doc) == 0 {
			continue
		}

		// YAML as string -> Unstructured object
		var obj unstructured.Unstructured
		if _, _, unmarshalErr := decoder.Decode([]byte(doc), nil, &obj); unmarshalErr != nil {
			return []unstructured.Unstructured{}, errors.Wrap(unmarshalErr, "cannot parse rendered YAML")
		}

		// Optionally: We can be rendering only selected types of objects, e.g. only "kind: Job"
		if len(kindsToRender) > 0 {
			logger.Debug("Going to use a filter to keep only selected types of resources")

			gvk := obj.GroupVersionKind()
			found := false
			logger.Debugf("Current kind: %v, allowed kinds: %v", gvk.String(), kindsToRender)

			for _, search := range kindsToRender {
				if gvk.String() == search.String() {
					logger.Debug("Matched.")
					found = true
					break
				}
			}
			if !found {
				logger.Debugf("Skipping not matching the filter: %v", gvk.String())
				continue
			}
		}

		objects = append(objects, obj)
	}
	return objects, nil
}

// writeGPGKey is extracting a proper GPG key from Kubernetes Secret and writing down to the temporary file
func writeGPGKey(backup *domain.ScheduledBackupAggregate, writeToPath string, operation string) error {
	keyName := backup.Spec.GPGKeySecretRef.PrivateKey
	if operation == "backup" {
		keyName = backup.Spec.GPGKeySecretRef.PublicKey
	}
	return os.WriteFile(writeToPath, backup.GPGSecret.Data[keyName], 0700)
}

// writeDefinition is writing the definition.yaml into the workspace
func writeDefinition(logger *logrus.Entry, backup *domain.ScheduledBackupAggregate, writeToPath string) error {
	var vars map[string]interface{}
	if err := yaml.Unmarshal([]byte(backup.Spec.Vars), &vars); err != nil {
		return errors.Wrap(err, "cannot parse .spec.vars as YAML")
	}

	logger.Debugf("backup.AdditionalVarsList = %v", backup.AdditionalVarsList)

	// each entry from Kubernetes secret convert into a YAML value
	// by converting a dotted path into a map
	varSources := make([]map[string][]byte, 0)
	if len(backup.AdditionalVarsList) > 0 {
		varSources = append(varSources, backup.AdditionalVarsList)
	}
	if backup.VarsListSecret.Data != nil && len(backup.VarsListSecret.Data) > 0 {
		varSources = append(varSources, backup.VarsListSecret.Data)
	}
	logger.Debugf("Copying vars from referenced secret")
	for sourceNum, source := range varSources {
		for path, value := range source {
			// if only specific keys should be used from the secret, then the rest should be skipped
			if sourceNum == 1 && len(backup.Spec.VarsSecretRef.ImportOnlyKeys) > 0 {
				if !contains(backup.Spec.VarsSecretRef.ImportOnlyKeys, path) {
					continue
				}
			}

			logger.Debugf("Setting '%s' -> '%v'", path, string(value))
			expression, err := jp.ParseString("$." + path)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("cannot parse dot-notation path to convert from some.path.dot format. Name: '%s'", path))
			}
			if setErr := expression.Set(vars, string(value)); setErr != nil {
				return errors.Wrap(err, fmt.Sprintf("cannot merge value from Secret into the vars, name: '%s'", path))
			}
		}
	}

	logger.Debug("Serializing definition.yaml")
	asYaml, marshalingErr := yaml.Marshal(vars)
	logger.Debug(string(asYaml))
	if marshalingErr != nil {
		return errors.Wrap(marshalingErr, "cannot serialize vars to YAML as a definition.yaml")
	}
	return os.WriteFile(writeToPath, asYaml, 0700)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

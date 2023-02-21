package bmg

import (
	"encoding/json"
	"fmt"
	"github.com/ohler55/ojg/jp"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-controller/pkg/domain"
	"github.com/riotkit-org/br-backup-maker/generate"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"os"
	"path/filepath"
	"strings"
)

func RenderKubernetesResourcesFor(logger *logrus.Entry, backup domain.Renderable) ([]unstructured.Unstructured, error) {
	//
	// Render helper resources for all operation types: "restore" + "backup"
	// This in effect will create e.g. separate ConfigMap, Secret for "restore" and separate for "backup"
	//
	// Typically a `kind: ScheduledBackup` is rendering all helper objects (Secret/ConfigMap)
	// And `kind: CronJob` for `kind: ScheduledBackup`
	//
	if backup.ShouldRenderDependentObjectsForAllOperationTypes() {
		var rendered []unstructured.Unstructured
		for _, operation := range []domain.Operation{domain.Backup, domain.Restore} {
			opRendered, renderErr := RenderKubernetesResourcesForOperation(logger, backup, operation,
				domain.NewResourceTypesFilterForScheduledBackup(backup, operation))

			if renderErr != nil {
				return []unstructured.Unstructured{}, renderErr
			}
			rendered = append(rendered, opRendered...)
		}
		return rendered, nil
	}
	//
	// Render runtime resources e.g. `kind: Job` that runs immediately
	//
	return RenderKubernetesResourcesForOperation(logger, backup, backup.GetOperation(),
		domain.NewResourceTypesFilterForRequestedBackupAction())
}

// RenderKubernetesResourcesForOperation is rendering Kubernetes resources like CronJob, Job, Secret, ConfigMap using Backup Maker Generator (BMG), which is using Helm under the hood
func RenderKubernetesResourcesForOperation(logger *logrus.Entry, backup domain.Renderable,
	operation domain.Operation, acceptedResourceTypes domain.ResourceTypes) ([]unstructured.Unstructured, error) {

	logger.Debugln("RenderKubernetesResourcesForOperation()")

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
	// Extracts `kind: ClusterBackupProcedureTemplate` into a local file
	_ = os.MkdirAll("./templates/"+string(operation), 0755)
	templatePath := "./templates/" + string(operation) + "/" + backup.GetTemplate().GetName() + ".tmpl"
	if writeErr := writeTemplate(logger, backup.GetTemplate(), operation, templatePath); writeErr != nil {
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
		TemplateName:   backup.GetTemplate().GetName(),
		UserConfigFile: definitionPath,
		IsKubernetes:   true,
		KeyPath:        gpgPath,
		OutputDir:      dir + "/output",
		Schedule:       backup.GetScheduledBackup().Spec.CronJob.ScheduleEvery,
		JobName:        backup.GetScheduledBackup().Name,
		Image:          backup.GetTemplate().GetImage(),
		Operation:      string(operation),
		Namespace:      backup.GetScheduledBackup().Namespace,
	}
	genErr := cmd.Run()
	if genErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(genErr, "error while generating manifests")
	}

	// read what was generated
	manifestsPath := dir + "/output/" + string(operation) + ".yaml"
	content, readErr := os.ReadFile(manifestsPath)
	if readErr != nil {
		return []unstructured.Unstructured{}, errors.Wrap(readErr, fmt.Sprintf(
			"cannot read rendered manifest file at path '%s'", manifestsPath))
	}

	return parseRenderedManifests(logger, string(content), acceptedResourceTypes)
}

// writeTemplate is writing the backup/restore procedure template
func writeTemplate(logger *logrus.Entry, template domain.Template, operation domain.Operation, path string) error {
	if !template.ProvidesScript() {
		logger.Debugln("Skipping script provision: !ProvidesScript()")
		return nil
	}
	logger.Debugf("writeTemplate(name: %v)", template)

	content := template.GetRestoreScript()
	if operation == domain.Backup {
		content = template.GetBackupScript()
	}

	// write JSON metadata file
	tplDef := generate.TemplateDefinition{
		DefaultImage: "this-should-be-overridden", // image is always overridden, so we do not set default value

		// operator writes both scripts under same name, but different subdirectories of course
		RestoreTemplate: template.GetName() + ".tmpl",
		BackupTemplate:  template.GetName() + ".tmpl",
	}
	defAsStr, defMarshalErr := json.Marshal(&tplDef)
	if defMarshalErr != nil {
		return errors.Wrap(defMarshalErr, "cannot marshal TemplateDefinition{} to JSON")
	}

	definitionDir := filepath.Dir(path) + "/../definition"
	if mkdirErr := os.MkdirAll(definitionDir, 0700); mkdirErr != nil {
		return errors.Wrapf(mkdirErr, "cannot create definition dir - %s", definitionDir)
	}

	definitionPath := definitionDir + "/" + template.GetName() + ".json"
	logger.Debugf("Writing a JSON definition file: %s", definitionPath)
	if writeErr := os.WriteFile(definitionPath, defAsStr, 0700); writeErr != nil {
		return errors.Wrapf(writeErr, "cannot write a file - %s", definitionPath)
	}

	return os.WriteFile(path, []byte(content), 0700)
}

// parseRenderedManifests is reading manifests from YAML into []UnstructuredObject
func parseRenderedManifests(logger *logrus.Entry, content string, kindsToRender domain.ResourceTypes) ([]unstructured.Unstructured, error) {
	decoder := apiyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// parsing
	var objects []unstructured.Unstructured
	docsContent := strings.Split(content, "---\n")

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
		if len(kindsToRender.GetKinds()) > 0 {
			logger.Debug("Going to use a filter to keep only selected types of resources")

			gvk := obj.GroupVersionKind()
			found := false
			logger.Debugf("Current kind: %v, allowed kinds: %v", gvk.String(), kindsToRender)

			for _, search := range kindsToRender.GetKinds() {
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
func writeGPGKey(backup *domain.ScheduledBackupAggregate, writeToPath string, operation domain.Operation) error {
	keyName := backup.Spec.GPGKeySecretRef.PrivateKey
	backup.AdditionalVarsList["HelmValues.gpgKeyContent"] = backup.GPGSecret.Data[backup.Spec.GPGKeySecretRef.PrivateKey]
	if operation == domain.Backup {
		keyName = backup.Spec.GPGKeySecretRef.PublicKey
		backup.AdditionalVarsList["HelmValues.gpgKeyContent"] = backup.GPGSecret.Data[backup.Spec.GPGKeySecretRef.PublicKey]
	}
	return os.WriteFile(writeToPath, backup.GPGSecret.Data[keyName], 0700)
}

// writeDefinition is writing the definition.yaml into the workspace
func writeDefinition(logger *logrus.Entry, backup *domain.ScheduledBackupAggregate, writeToPath string) error {
	var vars map[string]interface{}
	if err := yaml.Unmarshal([]byte(backup.Spec.Vars), &vars); err != nil {
		return errors.Wrap(err, "cannot parse .spec.vars as YAML")
	}

	type VarSource struct {
		data       map[string][]byte
		sourceType string
	}

	// each entry from Kubernetes secret convert into a YAML value
	// by converting a dotted path into a map
	varSources := make([]VarSource, 0)
	if len(backup.AdditionalVarsList) > 0 {
		varSources = append(varSources, VarSource{data: backup.AdditionalVarsList, sourceType: "AdditionalVars"})
	}
	if backup.VarsListSecret != nil && backup.VarsListSecret.Data != nil && len(backup.VarsListSecret.Data) > 0 {
		varSources = append(varSources, VarSource{data: backup.VarsListSecret.Data, sourceType: "Secret"})
	}
	logger.Debugf("Copying vars from referenced secret")
	for _, source := range varSources {
		for path, value := range source.data {
			// if only specific keys should be used from the secret, then the rest should be skipped
			if source.sourceType == "Secret" && len(backup.Spec.VarsSecretRef.ImportOnlyKeys) > 0 {
				if !contains(backup.Spec.VarsSecretRef.ImportOnlyKeys, path) {
					logger.Debugf("Skipping var '%s'", path)
					continue
				}
			}

			logger.Debugf("Setting '%s' -> '%v'", path, string(value))
			expression, jpErr := jp.ParseString("$." + path)
			if jpErr != nil {
				return errors.Wrap(jpErr, fmt.Sprintf("cannot parse dot-notation path to convert from some.path.dot format. Name: '%s'", path))
			}
			if setErr := expression.Set(vars, string(value)); setErr != nil {
				return errors.Wrap(setErr, fmt.Sprintf("cannot merge value from Secret into the vars, name: '%s'", path))
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

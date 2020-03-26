package kots

import (
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	goyaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/template"
)

func (files SpecFiles) render(config *kotsv1beta1.Config) (SpecFiles, error) {
	renderedFiles := SpecFiles{}

	for _, file := range files {
		if !file.isYAML() {
			renderedFiles = append(renderedFiles, file)
			continue
		}

		s, err := file.shouldBeRendered()
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if file should be rendered")
		}
		if !s {
			renderedFiles = append(renderedFiles, file)
			continue
		}

		renderedContent, err := file.render(config)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to render file %s", file.Path)
		}

		file.Content = string(renderedContent)

		renderedFiles = append(renderedFiles, file)
	}

	return renderedFiles, nil
}

func (f SpecFile) render(config *kotsv1beta1.Config) ([]byte, error) {
	if !f.isYAML() {
		return nil, errors.New("not a yaml file")
	}

	fileContent, err := fixUpYAML([]byte(f.Content))
	if err != nil {
		return nil, errors.Wrap(err, "failed to fix up yaml")
	}

	localRegistry := template.LocalRegistry{}
	templateContextValues := make(map[string]template.ItemValue)

	configGroups := []kotsv1beta1.ConfigGroup{}
	if config != nil && config.Spec.Groups != nil {
		configGroups = config.Spec.Groups
	}

	builder, _, err := template.NewBuilder(configGroups, templateContextValues, localRegistry, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	rendered, err := builder.RenderTemplate(string(fileContent), string(fileContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

// fixUpYAML is a general purpose function that will ensure that YAML is copmatible with KOTS
// This ensures that lines aren't wrapped at 80 chars which breaks template functions
func fixUpYAML(inputContent []byte) ([]byte, error) {
	yamlObj := map[string]interface{}{}

	err := yaml.Unmarshal(inputContent, &yamlObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	inputContent, err = util.MarshalIndent(2, yamlObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal yaml")
	}

	return inputContent, nil
}

func (files SpecFiles) findAndValidateConfig() (*kotsv1beta1.Config, string, error) {
	var config *kotsv1beta1.Config
	var path string

	for _, file := range files {
		document := &GVKDoc{}
		if err := goyaml.Unmarshal([]byte(file.Content), document); err != nil {
			continue
		}

		if document.APIVersion != "kots.io/v1beta1" || document.Kind != "Config" {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(file.Content), nil, nil)
		if err != nil {
			return nil, file.Path, errors.Wrap(err, "failed to decode config content")
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
			config = obj.(*kotsv1beta1.Config)
			path = file.Path
		}
	}

	return config, path, nil
}

func (f SpecFile) shouldBeRendered() (bool, error) {
	document := &GVKDoc{}
	if err := goyaml.Unmarshal([]byte(f.Content), document); err != nil {
		return false, errors.Wrap(err, "failed to unmarshal file content")
	}

	if document.APIVersion == "kots.io/v1beta1" && document.Kind == "Config" {
		return false, nil
	}

	if document.APIVersion == "kots.io/v1beta1" && document.Kind == "Application" {
		return false, nil
	}

	if document.APIVersion == "app.k8s.io/v1beta1" && document.Kind == "Application" {
		return false, nil
	}

	return true, nil
}

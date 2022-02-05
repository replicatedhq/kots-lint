package kots

import (
	"strconv"
	"strings"

	"github.com/replicatedhq/kots-lint/pkg/util"
	goyaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/template"
)

type RenderTemplateError struct {
	message string
	match   string
}

func (r RenderTemplateError) Error() string {
	return r.message
}

func (r RenderTemplateError) Match() string {
	return r.match
}

func (files SpecFiles) render() (SpecFiles, error) {
	config, _, err := files.findAndValidateConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find and validate config")
	}

	builder, err := getTemplateBuilder(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template builder")
	}

	renderedFiles := SpecFiles{}
	for _, file := range files {
		renderedContent, err := file.renderContent(builder)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to render spec file %s", file.Path)
		}
		file.Content = string(renderedContent)
		renderedFiles = append(renderedFiles, file)
	}

	return renderedFiles, nil
}

func (f SpecFile) renderContent(builder *template.Builder) ([]byte, error) {
	if !f.isYAML() {
		return nil, errors.New("not a yaml file")
	}

	s, err := f.shouldBeRendered()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if file should be rendered")
	}
	if !s {
		return []byte(f.Content), nil
	}

	// add new line so that parsing the render template error is easier (possible)
	content := f.Content + "\n"

	rendered, err := builder.RenderTemplate(content, content)
	if err != nil {
		return nil, parseRenderTemplateError(f, err.Error())
	}

	// remove the new line that was added to make parsing template error easier (possible)
	rendered = strings.TrimSuffix(rendered, "\n")

	return []byte(rendered), nil
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

	if config != nil {
		// if config was found, validate that it renders successfully
		configCopy := config.DeepCopy()
		if _, err := renderConfig(configCopy); err != nil {
			return config, path, errors.Wrap(err, "failed to render config")
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

	return true, nil
}

func renderConfig(config *kotsv1beta1.Config) ([]byte, error) {
	localRegistry := template.LocalRegistry{}
	appInfo := template.ApplicationInfo{}
	configValues := map[string]template.ItemValue{}

	renderedConfig, err := kotsconfig.TemplateConfigObjects(config, configValues, nil, nil, localRegistry, nil, &appInfo, nil, "", false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config objects")
	}

	b, err := goyaml.Marshal(renderedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rendered config")
	}

	return b, nil
}

func getTemplateBuilder(config *kotsv1beta1.Config) (*template.Builder, error) {
	localRegistry := template.LocalRegistry{}
	templateContextValues := make(map[string]template.ItemValue)

	configGroups := []kotsv1beta1.ConfigGroup{}
	if config != nil && config.Spec.Groups != nil {
		configGroups = config.Spec.Groups
	}

	opts := template.BuilderOptions{
		ConfigGroups:   configGroups,
		ExistingValues: templateContextValues,
		LocalRegistry:  localRegistry,
		ApplicationInfo: &template.ApplicationInfo{ // Kots 1.56.0 calls ApplicationInfo.Slug, this is required
			Slug: "app-slug",
		},
	}
	builder, _, err := template.NewBuilder(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	return &builder, nil
}

func parseRenderTemplateError(file SpecFile, value string) RenderTemplateError {
	/*
		** SAMPLE **
		failed to get template: template: apiVersion: v1
		data:
			ENV_VAR_1: fake
			ENV_VAR_2: '{{repl ConfigOptionEquals "test}}'
		kind: ConfigMap
		metadata:
			name: example-config
		:4: unterminated quoted string
	*/

	renderTemplateError := RenderTemplateError{
		match:   "",
		message: value,
	}

	parts := strings.Split(value, "\n:")
	if len(parts) == 1 {
		return renderTemplateError
	}

	lineAndMsg := parts[len(parts)-1]
	lineAndMsgParts := strings.SplitN(lineAndMsg, ":", 2)

	if len(lineAndMsgParts) == 1 {
		return renderTemplateError
	}

	// in some cases, the message contains the whole file content which is noisy and difficult to read
	msg := lineAndMsgParts[1]
	if i := strings.Index(msg, `\n"`); i != -1 {
		msg = msg[i+len(`\n"`):]
	}
	renderTemplateError.message = strings.TrimSpace(msg)

	// get the line number in the remarshalled (keys rearranged) data
	lineNumber, err := strconv.Atoi(lineAndMsgParts[0])
	if err != nil {
		return renderTemplateError
	}

	// try to find the data after it's been remarshalled (keys rearranged)
	data := util.GetStringInBetween(value, ": template: ", "\n:")
	if data == "" {
		return renderTemplateError
	}

	// find error line from data
	match := ""
	for index, line := range strings.Split(data, "\n") {
		if index == lineNumber-1 {
			match = line
			break
		}
	}
	renderTemplateError.match = match

	return renderTemplateError
}

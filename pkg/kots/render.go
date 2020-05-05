package kots

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/replicatedhq/kots-lint/pkg/util"
	goyaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/template"
)

type RenderTemplateError struct {
	message string
	line    int
}

func (r RenderTemplateError) Error() string {
	return r.message
}

func (r RenderTemplateError) Line() int {
	return r.line
}

func (f SpecFile) renderContent(config *kotsv1beta1.Config) ([]byte, error) {
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
		line:    -1,
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
	var errorLine interface{}
	for index, line := range strings.Split(data, "\n") {
		if index != lineNumber-1 {
			continue
		}
		err := goyaml.Unmarshal([]byte(line), &errorLine)
		if err != nil {
			return renderTemplateError
		}
		break
	}

	if errorLine == nil {
		return renderTemplateError
	}

	// find line number in original content
	originalLineIndex := -1
	for index, line := range strings.Split(file.Content, "\n") {
		var unmarshalledLine interface{}
		err := goyaml.Unmarshal([]byte(line), &unmarshalledLine)
		if err != nil {
			return renderTemplateError
		}
		if reflect.DeepEqual(unmarshalledLine, errorLine) {
			originalLineIndex = index
			break
		}
	}

	if originalLineIndex == -1 {
		return renderTemplateError
	}

	renderTemplateError.line = originalLineIndex + 1

	return renderTemplateError
}

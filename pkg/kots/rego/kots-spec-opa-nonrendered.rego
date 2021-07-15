## IMPORTANT ##
# This file should only contain rules for linting NON-rendered spec files
# Rego playground: https://play.openpolicyagent.org/

package kots.spec.nonrendered

## Secrets with template functions are excluded in the rule logic
secrets_regular_expressions = [
  # connection strings with username and password
  # http://user:password@host:8888
  "(?i)(https?|ftp)(:\\/\\/)[^:\"\\/]+(:)[^@\"\/]+@[^:\\/\\s\"]+:[\\d]+",
  # user:password@tcp(host:3309)/db-name
  "\\b[^:\"\\/]*(:)[^:\"\\/]*(@tcp\\()[^:\"\\/]*:[\\d]*?(\\)\\/)[\\w\\d\\S-_]+\\b",
  # passwords & tokens (stringified jsons)
  "(?i)(\\\"name\\\":\\\"[^\"]*password[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*token[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*database[^\"]*\\\",\\\"value\\\":\\\")",
  "(?i)(\\\"name\\\":\\\"[^\"]*user[^\"]*\\\",\\\"value\\\":\\\")",
  # passwords & tokens (in YAMLs)
  "(?i)(name: [\"']{0,1}password[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}token[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}database[\"']{0,1})\n\\s*(value:)",
  "(?i)(name: [\"']{0,1}user[\"']{0,1})\n\\s*(value:)",
  "(?i)password: .*",
  "(?i)token: .*",
  "(?i)database: .*",
  "(?i)user: .*",
  # standard postgres and mysql connnection strings
  "(?i)(Data Source *= *)[^\\;]+(;)",
  "(?i)(location *= *)[^\\;]+(;)",
  "(?i)(User ID *= *)[^\\;]+(;)",
  "(?i)(password *= *)[^\\;]+(;)",
  "(?i)(Server *= *)[^\\;]+(;)",
  "(?i)(Database *= *)[^\\;]+(;)",
  "(?i)(Uid *= *)[^\\;]+(;)",
  "(?i)(Pwd *= *)[^\\;]+(;)",
  # AWS secrets
  "SECRET_?ACCESS_?KEY",
  "ACCESS_?KEY_?ID",
  "OWNER_?ACCOUNT",
]

# Files set with the contents of each file as json
files[output] {
  file := input[_]
  output := {
    "name": file.name,
    "path": file.path,
    "content": yaml.unmarshal(file.content),
    "docIndex": object.get(file, "docIndex", 0)
  }
}

# Returns the string value of x
string(x) = y {
	y := split(yaml.marshal(x), "\n")[0]
}

# A set containing ALL the specs for each file
# 3 levels deep. "specs" rule for each level
specs[output] {
  file := files[_]
  spec := file.content.spec # 1st level
  output := {
    "path": file.path,
    "spec": spec,
    "field": "spec",
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key].spec # 2nd level
  field := concat(".", [string(key), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}
specs[output] {
  file := files[_]
  spec := file.content[key1][key2].spec # 3rd level
  field := concat(".", [string(key1), string(key2), "spec"])
  output := {
    "path": file.path,
    "spec": spec,
    "field": field,
    "docIndex": file.docIndex
  }
}

# A rule that returns the config file path
config_file_path = file.path {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
}

# A rule that returns the config data
config_data = output {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
  output := {
    "config": file.content.spec,
    "field": "spec",
    "docIndex": file.docIndex
  }
}

# A set containing all of the config groups, config items and child items
# Config Groups
config_options[output] {
  item := config_data.config.groups[index]
  field := concat(".", [config_data.field, "groups", string(index)])
  output := {
    "item": item,
    "field": field
  }
}
# Config Items
config_options[output] {
  item := config_data.config.groups[groupIndex].items[itemIndex]
  field := concat(".", [config_data.field, "groups", string(groupIndex), "items", string(itemIndex)])
  output := {
    "item": item,
    "field": field
  }
}
# Config Child Items
config_options[output] {
  item := config_data.config.groups[groupIndex].items[itemIndex].items[childItemIndex]
  field := concat(".", [config_data.field, "groups", string(groupIndex), "items", string(itemIndex), "items", string(childItemIndex)])
  output := {
    "item": item,
    "field": field
  }
}

# A function that checks if a config option exists in config
config_option_exists(option_name) {
  option := config_options[_].item
  option.name == option_name
}

# A function that checks if a config option is repeatable
config_option_is_repeatable(option_name) {
  option := config_options[_].item
  option.repeatable
}

# Check if any files are missing "kind"
lint[output] {
  file := files[_]
  not file.content.kind
  output := {
    "rule": "missing-kind-field",
    "type": "error",
    "message": "Missing \"kind\" field",
    "path": file.path,
    "docIndex": file.docIndex
  }
}

# Check if any files are missing "apiVersion"
lint[output] {
  file := files[_]
  not file.content.apiVersion
  output := {
    "rule": "missing-api-version-field",
    "type": "error",
    "message": "Missing \"apiVersion\" field",
    "path": file.path,
    "docIndex": file.docIndex
  }
}

# Check if Preflight spec exists
v1beta1_preflight_spec_exists {
  file := files[_]
  file.content.kind == "Preflight"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_preflight_spec_exists {
  file := files[_]
  file.content.kind == "Preflight"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
lint[output] {
  not v1beta1_preflight_spec_exists
  not v1beta2_preflight_spec_exists
  output := {
    "rule": "preflight-spec",
    "type": "warn",
    "message": "Missing preflight spec"
  }
}

# Check if Config spec exists
config_spec_exists {
  file := files[_]
  file.content.kind == "Config"
  file.content.apiVersion == "kots.io/v1beta1"
}
lint[output] {
  not config_spec_exists
  output := {
    "rule": "config-spec",
    "type": "warn",
    "message": "Missing config spec"
  }
}

# Check if Troubleshoot spec exists
v1beta1_troubleshoot_spec_exists {
  file := files[_]
  file.content.kind == "Collector"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_troubleshoot_spec_exists {
  file := files[_]
  file.content.kind == "Collector"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
v1beta1_supportbundle_spec_exists {
  file := files[_]
  file.content.kind == "SupportBundle"
  file.content.apiVersion == "troubleshoot.replicated.com/v1beta1"
}
v1beta2_supportbundle_spec_exists {
  file := files[_]
  file.content.kind == "SupportBundle"
  file.content.apiVersion == "troubleshoot.sh/v1beta2"
}
lint[output] {
  not v1beta1_troubleshoot_spec_exists
  not v1beta2_troubleshoot_spec_exists
  not v1beta1_supportbundle_spec_exists
  not v1beta2_supportbundle_spec_exists
  output := {
    "rule": "troubleshoot-spec",
    "type": "warn",
    "message": "Missing troubleshoot spec"
  }
}

# Check if Application spec exists
application_spec_exists {
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
}
lint[output] {
  not application_spec_exists
  output := {
    "rule": "application-spec",
    "type": "warn",
    "message": "Missing application spec"
  }
}

# Check if Application icon exists
lint[output] {
  file := files[_]
  file.content.kind == "Application"
  file.content.apiVersion == "kots.io/v1beta1"
  not file.content.spec.icon
  output := {
    "rule": "application-icon",
    "type": "warn",
    "message": "Missing application icon",
    "path": file.path,
    "field": "spec",
    "docIndex": file.docIndex
  }
}

# Check if any spec has "replicas" set to 1
lint[output] {
  spec := specs[_]
  spec.spec.replicas == 1
  field := concat(".", [spec.field, "replicas"])
  output := {
    "rule": "replicas-1",
    "type": "info",
    "message": "Found Replicas 1",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any spec has "privileged" set to true
lint[output] {
  spec := specs[_]
  spec.spec.privileged == true
  field := concat(".", [spec.field, "privileged"])
  output := {
    "rule": "privileged",
    "type": "info",
    "message": "Found privileged spec",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any spec has "allowPrivilegeEscalation" set to true
lint[output] {
  spec := specs[_]
  spec.spec.allowPrivilegeEscalation == true
  field := concat(".", [spec.field, "allowPrivilegeEscalation"])
  output := {
    "rule": "allow-privilege-escalation",
    "type": "info",
    "message": "Allows privilege escalation",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container Image" contains the tag ":latest"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  is_string(container.image)
  endswith(container.image, ":latest")
  field := concat(".", [spec.field, "containers", string(index), "image"])
  output := {
    "rule": "container-image-latest-tag",
    "type": "info",
    "message": "Container has image with tag 'latest'",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container Image" uses "LocalImageName"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  is_string(container.image)
  re_match("^(repl{{|{{repl)\\s*LocalImageName", container.image)
  field := concat(".", [spec.field, "containers", string(index), "image"])
  output := {
    "rule": "container-image-local-image-name",
    "type": "error",
    "message": "Container image utilizes LocalImageName",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Container" of a spec doesn’t have field "resources"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  not container.resources
  field := concat(".", [spec.field, "containers", string(index)])
  output := {
    "rule": "container-resources",
    "type": "info",
    "message": "Missing container resources",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource" doesn’t have field "limits"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources
  not container.resources.limits
  field := concat(".", [spec.field, "containers", string(index), "resources"])
  output := {
    "rule": "container-resource-limits",
    "type": "info",
    "message": "Missing resource limits",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource" doesn’t have field "requests"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources
  not container.resources.requests
  field := concat(".", [spec.field, "containers", string(index), "resources"])
  output := {
    "rule": "container-resource-requests",
    "type": "info",
    "message": "Missing resource requests",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Limits" doesn’t have field "cpu"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.limits
  not container.resources.limits.cpu
  field := concat(".", [spec.field, "containers", string(index), "resources", "limits"])
  output := {
    "rule": "resource-limits-cpu",
    "type": "info",
    "message": "Missing resource cpu limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Limits" doesn’t have field "memory"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.limits
  not container.resources.limits.memory
  field := concat(".", [spec.field, "containers", string(index), "resources", "limits"])
  output := {
    "rule": "resource-limits-memory",
    "type": "info",
    "message": "Missing resource memory limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Requests" doesn’t have field "cpu"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.requests
  not container.resources.requests.cpu
  field := concat(".", [spec.field, "containers", string(index), "resources", "requests"])
  output := {
    "rule": "resource-requests-cpu",
    "type": "info",
    "message": "Missing requests cpu limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Resource Requests" doesn’t have field "memory"
lint[output] {
  spec := specs[_]
  container := spec.spec.containers[index]
  container.resources.requests
  not container.resources.requests.memory
  field := concat(".", [spec.field, "containers", string(index), "resources", "requests"])
  output := {
    "rule": "resource-requests-memory",
    "type": "info",
    "message": "Missing requests memory limit",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Volume" of a spec has field "hostPath"
lint[output] {
  spec := specs[_]
  volume := spec.spec.volumes[index]
  volume.hostPath
  field := concat(".", [spec.field, "volumes", string(index), "hostPath"])
  output := {
    "rule": "volumes-host-paths",
    "type": "info",
    "message": "Volume has hostpath",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "Volume" of a spec has field "hostPath" set to "docker.sock"
lint[output] {
  spec := specs[_]
  volume := spec.spec.volumes[index]
  volume.hostPath.path == "/var/run/docker.sock"
  field := concat(".", [spec.field, "volumes", string(index), "hostPath", "path"])
  output := {
    "rule": "volume-docker-sock",
    "type": "info",
    "message": "Volume mounts docker.sock",
    "path": spec.path,
    "field": field,
    "docIndex": spec.docIndex
  }
}

# Check if any "namespace" is hardcoded
lint[output] {
  file := files[_]
  namespace := file.content.metadata.namespace
  is_string(namespace)
  not re_match("^(repl{{|{{repl)", namespace)
  output := {
    "rule": "hardcoded-namespace",
    "type": "info",
    "message": "Found a hardcoded namepsace",
    "path": file.path,
    "field": "metadata.namespace",
    "docIndex": file.docIndex
  }
}

# Check if any file may contain secrets
lint[output] {
  file := input[_] # using "input" instead if "files" because "file.content" is string in "input"
  expression := secrets_regular_expressions[_]
  expression_matches := regex.find_n(expression, file.content, -1)
  count(expression_matches) > 0
  match := expression_matches[_]
  not re_match("repl{{|{{repl", match) # exclude if template function
  output := {
    "rule": "may-contain-secrets",
    "type": "info",
    "message": "It looks like there might be secrets in this file",
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": match
  }
}

# Check if ConfigOption has a valid type
lint[output] {
  config_option := config_options[_]
  item := config_option.item
  item.type
  is_string(item.type)
  not re_match("^(text|label|password|file|bool|select_one|select_many|textarea|select|heading)$", item.type)
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Config option \"%s\" has an invalid type", [string(item.name)])
  output := {
    "rule": "config-option-invalid-type",
    "type": "error",
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if ConfigOption should have a "password" type
lint[output] {
  config_option := config_options[_]
  item := config_option.item
  is_string(item.name)
  re_match("password|secret|token", item.name)
  item.type != "password"
  field := concat(".", [config_option.field, "type"])
  message := sprintf("Config option \"%s\" should have type \"password\"", [item.name])
  output := {
    "rule": "config-option-password-type",
    "type": "warn",
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if all ConfigOptions exist
lint[output] {
  file := input[_]

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)?([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, file.content, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  not config_option_exists(option_name)

  message := sprintf("Config option \"%s\" not found", [option_name])
  output := {
    "rule": "config-option-not-found",
    "type": "warn",
    "message": message,
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": capture_groups[0]
  }
}

# Check if ConfigOption is circular (references itself)
lint[output] {
  config_option := config_options[_]
  item := config_option.item
  value := item[key]

  key != "items"

  marshalled_value := yaml.marshal(value)

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)?([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, marshalled_value, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  item.name == option_name

  field := concat(".", [config_option.field, string(key)])

  message := sprintf("Config option \"%s\" references itself", [option_name])
  output := {
    "rule": "config-option-is-circular",
    "type": "error",
    "message": message,
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

# Check if sub-templated ConfigOptions are repeatable
lint[output] {
  file := input[_]

  expression := "(ConfigOption|ConfigOptionName|ConfigOptionEquals|ConfigOptionNotEquals)\\W+?(repl\\W+?)([\\w\\d_-]+)"
  expression_matches := regex.find_all_string_submatch_n(expression, file.content, -1)

  capture_groups := expression_matches[_]
  option_name := capture_groups[3]
  not config_option_is_repeatable(option_name)

  message := sprintf("Config option \"%s\" not repeatable", [option_name])
  output := {
    "rule": "config-option-not-repeatable",
    "type": "error",
    "message": message,
    "path": file.path,
    "docIndex": object.get(file, "docIndex", 0),
    "match": capture_groups[0]
  }
}

# Check if "when" is valid
is_when_valid(when) {
  is_boolean(when)
} else {
  is_string(when)
  expression := "^((repl{{|{{repl).*[^}]}}$)|([tT]rue|[fF]alse)$"
  re_match(expression, when)
}
lint[output] {
  config_option := config_options[_]
  item := config_option.item

  not is_when_valid(item.when)

  field := concat(".", [config_option.field, "when"])

  output := {
    "rule": "config-option-when-is-invalid",
    "type": "error",
    "message": "Invalid \"when\" expression",
    "path": config_file_path,
    "field": field,
    "docIndex": config_data.docIndex
  }
}

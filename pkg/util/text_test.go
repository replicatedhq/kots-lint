package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tryGetLineNumberFromValue(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		hasErr  bool
	}{
		{
			name:    "double digits",
			content: `yaml: line 17: mapping values are not allowed in this context`,
			line:    17,
			hasErr:  false,
		},
		{
			name:    "single digit",
			content: `yaml: line 8: mapping values are not allowed in this context`,
			line:    8,
			hasErr:  false,
		},
		{
			name:    "not an integer",
			content: `yaml: line abc: mapping values are not allowed in this context`,
			line:    -1,
			hasErr:  true,
		},
		{
			name:    "no line number",
			content: `yaml: mapping values are not allowed in this context`,
			line:    -1,
			hasErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, err := TryGetLineNumberFromValue(test.content)
			assert.Equal(t, line, test.line)
			if err != nil {
				assert.True(t, test.hasErr)
			} else {
				assert.False(t, test.hasErr)
			}
		})
	}
}

func Test_getLineNumberFromYamlPath(t *testing.T) {
	tests := []struct {
		name     string
		yamlPath string
		docIndex int
		hasErr   bool
		line     int
		content  string
	}{
		{
			name:     "basic",
			yamlPath: "metadata.labels.app",
			docIndex: 0,
			hasErr:   false,
			line:     6,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
	name: another-nginx
	labels:
		app: another
		component: nginx`,
		},
		{
			name:     "empty lines and comments",
			yamlPath: "metadata.labels.component",
			docIndex: 0,
			hasErr:   false,
			line:     12,
			content: `apiVersion: apps/v1
kind: Deployment

metadata:

	name: another-nginx
	labels:
			
		# this is a comment
		
		app: another
		component: nginx`,
		},
		{
			name:     "single part",
			yamlPath: "kind",
			docIndex: 0,
			hasErr:   false,
			line:     2,
			content: `apiVersion: apps/v1
kind: Deployment`,
		},
		{
			name:     "empty yaml path",
			yamlPath: "",
			docIndex: 0,
			hasErr:   true,
			line:     -1,
			content:  "",
		},
		{
			name:     "empty content",
			yamlPath: "test.yaml.path",
			docIndex: 0,
			hasErr:   true,
			line:     -1,
			content:  "",
		},
		{
			name:     "negative document index",
			yamlPath: "test.yaml.path",
			docIndex: -1,
			hasErr:   true,
			line:     -1,
			content:  "test content",
		},
		{
			name:     "arrays index 0",
			yamlPath: "spec.containers.0",
			docIndex: 0,
			hasErr:   false,
			line:     3,
			content: `spec:
	containers:
		- image: nginx
			envFrom:
			- configMapRef:
					name: example-config
			resources:
				limits:
					memory: '256Mi'
					cpu: '500m'`,
		},
		{
			name:     "first key of array",
			yamlPath: "spec.containers.0.image",
			docIndex: 0,
			hasErr:   false,
			line:     3,
			content: `spec:
	containers:
		- image: nginx
			envFrom:
			- configMapRef:
					name: example-config
			resources:
				limits:
					memory: '256Mi'
					cpu: '500m'`,
		},
		{
			name:     "not first key of array",
			yamlPath: "spec.containers.0.resources",
			docIndex: 0,
			hasErr:   false,
			line:     7,
			content: `spec:
	containers:
		- image: nginx
			envFrom:
			- configMapRef:
					name: example-config
			resources:
				limits:
					memory: '256Mi'
					cpu: '500m'`,
		},
		{
			name:     "array index greater than 0",
			yamlPath: "spec.containers.1.resources.limits.cpu",
			docIndex: 0,
			hasErr:   false,
			line:     18,
			content: `spec:
	containers:
		- image: nginx
			envFrom:
			- configMapRef:
					name: example-config
			resources:
				limits:
					memory: '256Mi'
					cpu: '500m'
		- image: name
			envFrom:
			- configMapRef:
					name: example-config
			resources:
				limits:
					memory: '256Mi'
					cpu: '500m'`,
		},
		{
			name:     "dynamic indentation",
			yamlPath: "spec.containers.1.envFrom.0.configMapRef",
			docIndex: 0,
			hasErr:   false,
			line:     13,
			content: `spec:
 containers:
	-  image: nginx
		 envFrom:
		 - configMapRef:
						name: example-config
		 resources:
			limits:
				  memory: '256Mi'
				  cpu: '500m'
	-   image: nginx
		  envFrom:
		  - configMapRef:
						name: example-config
		  resources:
			 limits:
				  memory: '256Mi'
				  cpu: '500m'`,
		},
		{
			name:     "multi yaml doc",
			yamlPath: "spec.containers.1.envFrom.0.configMapRef",
			docIndex: 1,
			hasErr:   false,
			line:     32,
			content: `spec:
 containers:
	-  image: nginx
		 envFrom:
		 - configMapRef:
						name: example-config
		 resources:
			limits:
				  memory: '256Mi'
				  cpu: '500m'
	-   image: nginx
		  envFrom:
		  - configMapRef:
						name: example-config
		  resources:
			 limits:
				  memory: '256Mi'
					cpu: '500m'
---
spec:
 containers:
	-  image: another-nginx
		 envFrom:
		 - configMapRef:
						name: another-config
		 resources:
			limits:
				  memory: '256Mi'
				  cpu: '500m'
	-   image: nginx
		  envFrom:
		  - configMapRef:
						name: another-config
		  resources:
			 limits:
				  memory: '256Mi'
					cpu: '500m'`,
		},
		{
			name:     "multi yaml doc with comments",
			yamlPath: "spec.containers.1.envFrom.0.configMapRef",
			docIndex: 0,
			hasErr:   false,
			line:     19,
			content: `

#---
# another comment here

---
spec:
 containers:
	-  image: nginx
		 envFrom:
		 - configMapRef:
						name: example-config
		 resources:
			limits:
				  memory: '256Mi'
				  cpu: '500m'
	-   image: nginx
		  envFrom:
		  - configMapRef:
						name: example-config
		  resources:
			 limits:
				  memory: '256Mi'
					cpu: '500m'
---
spec:
 containers:
	-  image: another-nginx
		 envFrom:
		 - configMapRef:
						name: another-config
		 resources:
			limits:
				  memory: '256Mi'
				  cpu: '500m'
	-   image: nginx
		  envFrom:
		  - configMapRef:
						name: another-config
		  resources:
			 limits:
				  memory: '256Mi'
					cpu: '500m'`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, err := GetLineNumberFromYamlPath(test.content, test.yamlPath, test.docIndex)
			assert.Equal(t, line, test.line)
			if err != nil {
				assert.True(t, test.hasErr)
			} else {
				assert.False(t, test.hasErr)
			}
		})
	}
}

func Test_getLineNumberForDoc(t *testing.T) {
	tests := []struct {
		name     string
		docIndex int
		hasErr   bool
		line     int
		content  string
	}{
		{
			name:     "basic",
			docIndex: 0,
			hasErr:   false,
			line:     1,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
	name: example-nginx
	labels:
		app: example
		component: nginx`,
		},
		{
			name:     "first doc with empty lines and comments",
			docIndex: 0,
			hasErr:   false,
			line:     7,
			content: ` # comment

---

# comment

apiVersion: apps/v1
kind: Deployment

metadata:

	name: example-nginx
	labels:
			
		# this is a comment
		
		app: example
		component: nginx`,
		},
		{
			name:     "multi yaml doc",
			docIndex: 1,
			hasErr:   false,
			line:     9,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
	name: example-nginx
	labels:
		app: example
		component: nginx'
---
apiVersion: apps/v1
kind: Deployment
metadata:
	name: another-nginx
	labels:
		app: another
		component: nginx`,
		},
		{
			name:     "multi yaml doc with comments",
			docIndex: 1,
			hasErr:   false,
			line:     22,
			content: `

#---
# another comment here

---

apiVersion: apps/v1
kind: Deployment
metadata:
	name: example-nginx
	labels:
		app: example
		component: nginx'

# another comment	
				
---

# another comment

apiVersion: apps/v1
kind: Deployment
metadata:
	name: another-nginx
	labels:
		app: another
		component: nginx`,
		},
		{
			name:     "empty content",
			docIndex: 0,
			hasErr:   true,
			line:     -1,
			content:  "",
		},
		{
			name:     "negative document index",
			docIndex: -1,
			hasErr:   true,
			line:     -1,
			content:  `test content`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, err := GetLineNumberForDoc(test.content, test.docIndex)
			assert.Equal(t, line, test.line)
			if err != nil {
				assert.True(t, test.hasErr)
			} else {
				assert.False(t, test.hasErr)
			}
		})
	}
}

func Test_isLineEmpty(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "basic empty line",
			line: "",
			want: true,
		},
		{
			name: "comment without spaces",
			line: "# comment",
			want: true,
		},
		{
			name: "comment with spaces",
			line: "    # comment",
			want: true,
		},
		{
			name: "comment with tabs",
			line: "			# comment",
			want: true,
		},
		{
			name: "comment with tabs and spaces",
			line: "    			# comment",
			want: true,
		},
		{
			name: "tabs only",
			line: "				",
			want: true,
		},
		{
			name: "spaces only",
			line: "        ",
			want: true,
		},
		{
			name: "spaces and tabs only",
			line: "       					",
			want: true,
		},
		{
			name: "basic non-empty line",
			line: "key: value",
			want: false,
		},
		{
			name: "non-empty line with tabs and spaces",
			line: "    				key: value",
			want: false,
		},
		{
			name: "non-empty line with comments",
			line: "    				key: value # comment",
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isEmpty := IsLineEmpty(test.line)
			assert.Equal(t, isEmpty, test.want)
		})
	}
}

func Test_getLineIndentation(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "basic empty line",
			line: "",
			want: "",
		},
		{
			name: "indentation with spaces only",
			line: "    key: value",
			want: "    ",
		},
		{
			name: "indentation with tabs only",
			line: "			- key: value",
			want: "			",
		},
		{
			name: "indentation with tabs and spaces",
			line: "    			- key: value",
			want: "    			",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			indentation := GetLineIndentation(test.line)
			assert.Equal(t, indentation, test.want)
		})
	}
}

func Test_getStringInBetween(t *testing.T) {
	tests := []struct {
		name  string
		str   string
		start string
		end   string
		want  string
	}{
		{
			name:  "basic",
			str:   "any random text",
			start: "any",
			end:   "text",
			want:  " random ",
		},
		{
			name: "multi line text",
			str: `my
:multi
line
text`,
			start: "\n:",
			end:   "text",
			want: `multi
line
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetStringInBetween(test.str, test.start, test.end)
			assert.Equal(t, actual, test.want)
		})
	}
}

func Test_cleanYaml(t *testing.T) {
	tests := []struct {
		name  string
		str   string
		start string
		end   string
		want  string
	}{
		{
			name: "multi line empty",
			str: `# this is a comment

# this is another comment
`,
			want: "",
		},
		{
			name: "add new line if ends with '---' 1",
			str: `key0: value0
---`,
			want: `key0: value0
---
`,
		},
		{
			name: "add new line if ends with '---' 2",
			str: `key0: value0
---
key1: value1
---`,
			want: `key0: value0
---
key1: value1
---
`,
		},
		{
			name: "multi line value with comments",
			str: `key0: value0
# comment
key1: value1
# another comment
`,
			want: `key0: value0
key1: value1`,
		},
		{
			name: "multi line value with comments and new lines",
			str: `key0: value0

# comment

key1: value1

# another comment
`,
			want: `key0: value0
key1: value1`,
		},
		{
			name: "comment after separator",
			str: `key0: value0

# comment
--- # this is a comment after a separator

key1: value1

---# this is another comment after another separator

key2: value2
`,
			want: `key0: value0
---
key1: value1
---
key2: value2`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := CleanYaml(test.str)
			assert.Equal(t, actual, test.want)
		})
	}
}

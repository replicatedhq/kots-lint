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

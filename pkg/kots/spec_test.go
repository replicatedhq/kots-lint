package kots

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fileHasContent(t *testing.T) {
	tests := []struct {
		name string
		file SpecFile
		want bool
	}{
		{
			name: "basic empty file",
			file: SpecFile{
				Name:    "a.yaml",
				Path:    "a.yaml",
				Content: "",
			},
			want: false,
		},
		{
			name: "basic with content",
			file: SpecFile{
				Name:    "a.yaml",
				Path:    "a.yaml",
				Content: "key: value",
			},
			want: true,
		},
		{
			name: "only spaces and comments",
			file: SpecFile{
				Name: "a.yaml",
				Path: "a.yaml",
				Content: `# comment
    
# another comment`,
			},
			want: false,
		},
		{
			name: "empty but multi doc",
			file: SpecFile{
				Name: "a.yaml",
				Path: "a.yaml",
				Content: `# comment
    
# another comment
    
---
    
# another comment`,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasContent := tt.file.hasContent()
			assert.Equal(t, hasContent, tt.want)
		})
	}
}

func Test_unnestSpecFiles(t *testing.T) {
	tests := []struct {
		name  string
		files SpecFiles
		want  SpecFiles
	}{
		{
			name: "basic",
			files: SpecFiles{
				{
					Name: "a",
					Path: "a",
					Children: SpecFiles{
						{
							Name: "b",
							Path: "a/b",
						},
						{
							Name: "c",
							Path: "a/c",
							Children: SpecFiles{
								{
									Name: "d",
									Path: "a/c/d",
								},
								{
									Name: "e",
									Path: "a/c/e",
								},
							},
						},
					},
				},
				{
					Name: "b",
					Path: "b",
					Children: SpecFiles{
						{
							Name: "c",
							Path: "b/c",
							Children: SpecFiles{
								{
									Name: "d",
									Path: "b/c/d",
								},
							},
						},
					},
				},
			},
			want: SpecFiles{
				{
					Name: "b",
					Path: "a/b",
				},
				{
					Name: "d",
					Path: "a/c/d",
				},
				{
					Name: "e",
					Path: "a/c/e",
				},
				{
					Name: "d",
					Path: "b/c/d",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unnestedFiles := tt.files.unnest()
			assert.ElementsMatch(t, unnestedFiles, tt.want)
		})
	}
}

func Test_separateSpecFiles(t *testing.T) {
	tests := []struct {
		name  string
		files SpecFiles
		want  SpecFiles
	}{
		{
			name: "basic",
			files: SpecFiles{
				{
					Name:    "a.yaml",
					Path:    "a.yaml",
					Content: "key0: value0",
				},
				{
					Name: "b.yaml",
					Path: "b.yaml",
					Content: `key0: value0
---
key1: value1`,
				},
				{
					Name: "c.yaml",
					Path: "c.yaml",
					Content: `---
key0: value0
---
key1: value1`,
				},
				{
					Name: "d.yaml",
					Path: "d.yaml",
					Content: `---
key0: value0
---
key1: value1
---
key2: value2`,
				},
				{
					Name:    "e.yaml",
					Path:    "e.yaml",
					Content: `---`,
				},
				{
					Name: "f.yaml",
					Path: "f.yaml",
					Content: `# comment
    
---
# another comment`,
				},
				{
					Name: "g.yaml",
					Path: "g.yaml",
					Content: `# comment
    
# another comment`,
				},
				{
					Name:    "h.yaml",
					Path:    "h.yaml",
					Content: "",
				},
				{
					Name: "i.yaml",
					Path: "i.yaml",
					Content: `key0: value0
---
# comment`,
				},
				{
					Name: "j.yaml",
					Path: "j.yaml",
					Content: `---
# comment
    
---
# another comment`,
				},
				{
					Name:    "preserve-quotes.yaml",
					Path:    "preserve-quotes.yaml",
					Content: `key: 'repl{{ConfigOption "my_config_option"}}'`,
				},
			},
			want: SpecFiles{
				{
					Name:     "a.yaml",
					Path:     "a.yaml",
					Content:  "key0: value0",
					DocIndex: 0,
				},
				{
					Name:     "b.yaml",
					Path:     "b.yaml",
					Content:  "key0: value0",
					DocIndex: 0,
				},
				{
					Name:     "b.yaml",
					Path:     "b.yaml",
					Content:  "key1: value1",
					DocIndex: 1,
				},
				{
					Name:     "c.yaml",
					Path:     "c.yaml",
					Content:  "key0: value0",
					DocIndex: 0,
				},
				{
					Name:     "c.yaml",
					Path:     "c.yaml",
					Content:  "key1: value1",
					DocIndex: 1,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key0: value0",
					DocIndex: 0,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key1: value1",
					DocIndex: 1,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key2: value2",
					DocIndex: 2,
				},
				{
					Name:     "i.yaml",
					Path:     "i.yaml",
					Content:  "key0: value0",
					DocIndex: 0,
				},
				{
					Name:     "preserve-quotes.yaml",
					Path:     "preserve-quotes.yaml",
					Content:  `key: 'repl{{ConfigOption "my_config_option"}}'`,
					DocIndex: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			separatedFiles, err := tt.files.separate()
			require.NoError(t, err)
			assert.ElementsMatch(t, separatedFiles, tt.want)
		})
	}
}

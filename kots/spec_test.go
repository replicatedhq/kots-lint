package kots

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			},
			want: SpecFiles{
				{
					Name:     "a.yaml",
					Path:     "a.yaml",
					Content:  "key0: value0\n",
					DocIndex: 0,
				},
				{
					Name:     "b.yaml",
					Path:     "b.yaml",
					Content:  "key0: value0\n",
					DocIndex: 0,
				},
				{
					Name:     "b.yaml",
					Path:     "b.yaml",
					Content:  "key1: value1\n",
					DocIndex: 1,
				},
				{
					Name:     "c.yaml",
					Path:     "c.yaml",
					Content:  "key0: value0\n",
					DocIndex: 0,
				},
				{
					Name:     "c.yaml",
					Path:     "c.yaml",
					Content:  "key1: value1\n",
					DocIndex: 1,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key0: value0\n",
					DocIndex: 0,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key1: value1\n",
					DocIndex: 1,
				},
				{
					Name:     "d.yaml",
					Path:     "d.yaml",
					Content:  "key2: value2\n",
					DocIndex: 2,
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

package kots

import (
	"context"
	"embed"
	_ "embed"
	"io"
	"sort"
	"testing"

	"gopkg.in/stretchr/testify.v1/assert"
)

//go:embed test-data/*
var testdata embed.FS

func TestLintBuilders(t *testing.T) {
	tests := []struct {
		name        string
		chartReader func() io.Reader
		wantErr     bool
		want        []LintExpression
	}{
		{
			name: "chart with all recommended labels present",
			chartReader: func() io.Reader {
				f, err := testdata.Open("test-data/builders/testchart-with-labels-16.2.2.tgz")
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}
				return f
			},
			wantErr: false,
			want: []LintExpression{
				{
					Rule:      "preflight-spec",
					Type:      "warn",
					Message:   "Missing preflight spec",
					Path:      "",
					Positions: nil,
				},
			},
		},
		{
			name: "chart with all recommended labels missing",
			chartReader: func() io.Reader {
				f, err := testdata.Open("test-data/builders/testchart-without-labels-16.2.2.tgz")
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}
				return f
			},
			wantErr: false,
			want: []LintExpression{
				{
					Rule:      "informers-labels-not-found",
					Type:      "warn",
					Message:   "No informer labels found on any resources",
					Path:      "",
					Positions: nil,
				},
				{
					Rule:      "preflight-spec",
					Type:      "warn",
					Message:   "Missing preflight spec",
					Path:      "",
					Positions: nil,
				},
			},
		},
		{
			name: "chart with some recommended labels missing, but with preflight spec",
			chartReader: func() io.Reader {
				f, err := testdata.Open("test-data/builders/testchart-without-labels-with-preflight-16.2.3.tgz")
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}
				return f
			},
			wantErr: false,
			want: []LintExpression{
				{
					Rule:      "informers-labels-not-found",
					Type:      "warn",
					Message:   "No informer labels found on any resources",
					Path:      "",
					Positions: nil,
				},
			},
		},
		{
			name: "invalid chart",
			chartReader: func() io.Reader {
				f, err := testdata.Open("test-data/builders/not-a-chart.tgz")
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}
				return f
			},
			wantErr: true,
			want:    nil,
		},
	}

	if err := InitOPALinting(); err != nil {
		t.Fatalf("failed to initialize OPA linting: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFiles, err := GetFilesFromChartReader(tt.chartReader())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			got, err := LintBuilders(context.Background(), gotFiles)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			sort.Sort(LintExpressionsByRule(tt.want))
			sort.Sort(LintExpressionsByRule(got))
			assert.Equal(t, tt.want, got)
		})
	}
}

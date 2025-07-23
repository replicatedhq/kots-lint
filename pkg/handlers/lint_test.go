package handlers

import (
	"archive/tar"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LintRelease(t *testing.T) {

	type resultType struct {
		LintExpressions []domain.LintExpression `json:"lintExpressions"`
	}

	getTarReader := func(filesNames []string) io.Reader {
		pipeReader, pipeWriter := io.Pipe()

		go func() {
			defer pipeWriter.Close()

			tarWriter := tar.NewWriter(pipeWriter)
			defer tarWriter.Close()

			for _, fileName := range filesNames {
				data, err := testdata.ReadFile(fileName)
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}

				if strings.HasSuffix(fileName, ".tgz") {
					data = []byte(base64.StdEncoding.EncodeToString(data))
				}

				header := &tar.Header{
					Name: filepath.Base(fileName),
					Mode: 0644,
					Size: int64(len(data)),
				}

				tarWriter.WriteHeader(header)
				tarWriter.Write(data)
			}
		}()

		return pipeReader
	}

	tests := []struct {
		name        string
		chartReader func(t *testing.T) io.ReadCloser
		contentType string
		want        resultType
	}{
		{
			name: "one valid chart without kotskinds",
			chartReader: func(t *testing.T) io.ReadCloser {
				files := []string{
					"test-data/builders/testchart-with-labels-16.2.2.tgz",
					"test-data/kots/chart-crds/testchart-with-labels-16.2.2.yaml",
				}

				return io.NopCloser(getTarReader(files))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []domain.LintExpression{
					{
						Rule:      "preflight-spec",
						Type:      "warn",
						Message:   "Missing preflight spec",
						Path:      "",
						Positions: nil,
					},
					{
						Rule:      "application-spec",
						Type:      "warn",
						Message:   "Missing application spec",
						Path:      "",
						Positions: nil,
					},
					{
						Rule:      "config-spec",
						Type:      "warn",
						Message:   "Missing config spec",
						Path:      "",
						Positions: nil,
					},
					{
						Rule:      "troubleshoot-spec",
						Type:      "warn",
						Message:   "Missing troubleshoot spec",
						Path:      "",
						Positions: nil,
					},
				},
			},
		},
		{
			name: "one valid chart with preflights but without kotskinds in release",
			chartReader: func(t *testing.T) io.ReadCloser {
				files := []string{
					"test-data/builders/testchart-with-labels-with-preflightspec-in-secret-16.2.2.tgz",
					"test-data/kots/chart-crds/testchart-with-labels-with-preflightspec-in-secret-16.2.2.yaml",
				}

				return io.NopCloser(getTarReader(files))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []domain.LintExpression{
					{
						Rule:      "application-spec",
						Type:      "warn",
						Message:   "Missing application spec",
						Path:      "",
						Positions: nil,
					},
					{
						Rule:      "config-spec",
						Type:      "warn",
						Message:   "Missing config spec",
						Path:      "",
						Positions: nil,
					},
					{
						Rule:      "troubleshoot-spec",
						Type:      "warn",
						Message:   "Missing troubleshoot spec",
						Path:      "",
						Positions: nil,
					},
				},
			},
		},
		{
			name: "one valid chart without preflights but with kotskinds in release",
			chartReader: func(t *testing.T) io.ReadCloser {
				yamlFiles, err := testdata.ReadDir("test-data/kots/kots-kinds")
				assert.NoError(t, err)

				files := []string{
					"test-data/builders/testchart-with-labels-16.2.2.tgz",
					"test-data/kots/chart-crds/testchart-with-labels-16.2.2.yaml",
				}
				for _, f := range yamlFiles {
					files = append(files, filepath.Join("test-data/kots/kots-kinds", f.Name()))
				}

				return io.NopCloser(getTarReader(files))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []domain.LintExpression{
					{
						Rule:    "application-statusInformers",
						Type:    "warn",
						Message: "Missing application statusInformers",
						Path:    "kots-app.yaml",
						Positions: []domain.LintExpressionItemPosition{
							{
								Start: domain.LintExpressionItemLinePosition{
									Line: 5,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "one valid chart with preflights and with kotskinds in release",
			chartReader: func(t *testing.T) io.ReadCloser {
				yamlFiles, err := testdata.ReadDir("test-data/kots/kots-kinds")
				assert.NoError(t, err)

				files := []string{
					"test-data/builders/testchart-with-labels-with-preflightspec-in-secret-16.2.2.tgz",
					"test-data/kots/chart-crds/testchart-with-labels-with-preflightspec-in-secret-16.2.2.yaml",
				}
				for _, f := range yamlFiles {
					files = append(files, filepath.Join("test-data/kots/kots-kinds", f.Name()))
				}

				return io.NopCloser(getTarReader(files))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []domain.LintExpression{
					{
						Rule:    "application-statusInformers",
						Type:    "warn",
						Message: "Missing application statusInformers",
						Path:    "kots-app.yaml",
						Positions: []domain.LintExpressionItemPosition{
							{
								Start: domain.LintExpressionItemLinePosition{
									Line: 5,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			clientRequest := &http.Request{
				Body: tt.chartReader(t),
				Header: http.Header{
					"Content-Type": []string{tt.contentType},
				},
			}
			respWriter := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(respWriter)
			c.Request = clientRequest

			LintRelease(c)

			req.Equal(http.StatusOK, respWriter.Result().StatusCode)

			body, err := io.ReadAll(respWriter.Body)
			req.NoError(err)

			var got resultType
			err = json.Unmarshal(body, &got)
			req.NoError(err)
			req.ElementsMatch(tt.want.LintExpressions, got.LintExpressions)
		})
	}
}

package handlers

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	"github.com/stretchr/testify/require"
)

func Test_LintBuildersRelease(t *testing.T) {

	type resultType struct {
		LintExpressions []kots.LintExpression `json:"lintExpressions"`
	}

	getTarReader := func(filesNames []string) io.Reader {
		pipeReader, pipeWriter := io.Pipe()

		go func() {
			defer pipeWriter.Close()

			tarWriter := tar.NewWriter(pipeWriter)
			defer tarWriter.Close()

			for _, fileName := range filesNames {
				data, err := testdata.ReadFile(fmt.Sprintf("test-data/builders/%s", fileName))
				if err != nil {
					t.Fatalf("failed to open file: %v", err)
				}

				header := &tar.Header{
					Name: fileName,
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
		chartReader func() io.ReadCloser
		contentType string
		want        resultType
	}{
		{
			name: "one valid chart without preflights",
			chartReader: func() io.ReadCloser {
				return io.NopCloser(getTarReader([]string{"testchart-with-labels-16.2.2.tgz"}))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []kots.LintExpression{
					{
						Rule:      "preflight-spec",
						Type:      "warn",
						Message:   "Missing preflight spec",
						Path:      "",
						Positions: nil,
					},
				},
			},
		},
		{
			name: "one valid chart with preflights",
			chartReader: func() io.ReadCloser {
				return io.NopCloser(getTarReader([]string{"testchart-with-labels-with-preflightspec-in-secret-16.2.2.tgz"}))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: nil,
			},
		},
		{
			name: "one valid chart without preflights and one invalid chart",
			chartReader: func() io.ReadCloser {
				return io.NopCloser(getTarReader([]string{"testchart-with-labels-16.2.2.tgz", "not-a-chart.tgz"}))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []kots.LintExpression{
					{
						Rule:      "rendering",
						Type:      "error",
						Message:   "load chart archive: EOF",
						Path:      "not-a-chart.tgz",
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
		},
		{
			name: "one invalid chart",
			chartReader: func() io.ReadCloser {
				return io.NopCloser(getTarReader([]string{"not-a-chart.tgz"}))
			},
			contentType: "application/tar",
			want: resultType{
				LintExpressions: []kots.LintExpression{
					{
						Rule:      "rendering",
						Type:      "error",
						Message:   "load chart archive: EOF",
						Path:      "not-a-chart.tgz",
						Positions: nil,
					},
				},
			},
		},
		{
			name: "one invalid chart",
			chartReader: func() io.ReadCloser {
				r, _ := testdata.Open("test-data/builders/not-a-chart.tgz")
				return r
			},
			contentType: "application/gzip",
			want: resultType{
				LintExpressions: []kots.LintExpression{
					{
						Rule:      "rendering",
						Type:      "error",
						Message:   "load chart archive: EOF",
						Positions: nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			clientRequest := &http.Request{
				Body: tt.chartReader(),
				Header: http.Header{
					"Content-Type": []string{tt.contentType},
				},
			}
			respWriter := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(respWriter)
			c.Request = clientRequest

			LintBuildersRelease(c)

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

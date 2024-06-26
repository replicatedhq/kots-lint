package handlers

import (
	"archive/tar"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kots-lint/pkg/domain" // Add this line
	"github.com/replicatedhq/kots-lint/pkg/kots"
	log "github.com/sirupsen/logrus"
)

// LintBuildersReleaseParameters contains parameters to lint a release for an app
type LintBuildersReleaseParameters struct {
}

// LintBuildersReleaseResponse contains the lint expressions
type LintBuildersReleaseResponse struct {
	// JSON payload
	// Required: true
	// In: body
	Body struct {
		LintExpressions []domain.LintExpression `json:"lintExpressions"`
	}
}

// LintBuildersRelease http handler for linting a release
func LintBuildersRelease(c *gin.Context) {
	log.Infof("Received builders lint request with content-length=%s, content-type=%s, client-ip=%s", c.GetHeader("content-length"), c.ContentType(), c.ClientIP())

	ctx := c.Request.Context()

	specFiles := domain.SpecFiles{}
	numChartsRendered := 0

	// Include rendering errors in the lint results (even though pedantically they're not lint expressions)
	var lintExpressions []domain.LintExpression
	if c.ContentType() == "application/tar" {
		tarReader := tar.NewReader(c.Request.Body)
		for {
			header, err := tarReader.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Errorf("failed to read tar input: %v", err)
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			if header.Typeflag != tar.TypeReg {
				continue
			}

			log.Debugf("adding files for chart %s", header.Name)
			files, err := kots.GetFilesFromChartReader(ctx, tarReader)
			if err != nil {
				log.Infof("failed to get files from chart %s: %v", header.Name, err)
				lintExpressions = append(lintExpressions, domain.LintExpression{
					Rule:    "rendering",
					Type:    "error",
					Message: err.Error(),
					Path:    header.Name,
				})
				continue
			}

			troubleshootSpecs := kots.GetEmbeddedTroubleshootSpecs(ctx, files)

			numChartsRendered += 1
			specFiles = append(specFiles, files...)
			specFiles = append(specFiles, troubleshootSpecs...)
		}
	} else if c.ContentType() == "application/gzip" {
		files, err := kots.GetFilesFromChartReader(ctx, c.Request.Body)
		if err != nil {
			log.Infof("failed to get files from request: %v", err)
			lintExpressions = append(lintExpressions, domain.LintExpression{
				Rule:    "rendering",
				Type:    "error",
				Message: err.Error(),
			})
		} else {
			troubleshootSpecs := kots.GetEmbeddedTroubleshootSpecs(ctx, files)

			numChartsRendered += 1
			specFiles = append(specFiles, files...)
			specFiles = append(specFiles, troubleshootSpecs...)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content type must be application/gzip or application/tar"})
		return
	}

	// Only lint if at least one chart was rendered, otherwise we get missing spec warnings/errors
	if numChartsRendered > 0 {
		lint, err := kots.LintBuilders(ctx, specFiles)
		if err != nil {
			log.Errorf("failed to lint builders charts: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		lintExpressions = append(lintExpressions, lint...)
	}

	response := LintBuildersReleaseResponse{}
	response.Body.LintExpressions = lintExpressions

	c.JSON(http.StatusOK, response.Body)
}

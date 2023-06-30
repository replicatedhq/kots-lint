package handlers

import (
	"archive/tar"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
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
		LintExpressions []kots.LintExpression `json:"lintExpressions"`
	}
}

// LintBuildersRelease http handler for linting a release
func LintBuildersRelease(c *gin.Context) {
	log.Infof("Received builders lint request with content-length=%s, content-type=%s, client-ip=%s", c.GetHeader("content-length"), c.ContentType(), c.ClientIP())

	specFiles := kots.SpecFiles{}
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
			files, err := kots.GetFilesFromChartReader(tarReader)
			if err != nil {
				log.Infof("failed to get files from chart %s: %v", header.Name, err)
				continue
			}

			specFiles = append(specFiles, files...)
		}
	} else if c.ContentType() == "application/gzip" {
		files, err := kots.GetFilesFromChartReader(c.Request.Body)
		if err != nil {
			log.Errorf("failed to get files from chart: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		specFiles = append(specFiles, files...)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content type must be application/gzip or application/tar"})
		return
	}

	lintExpressions, err := kots.LintBuilders(c.Request.Context(), specFiles)
	if err != nil {
		log.Errorf("failed to lint builders charts: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	response := LintReleaseResponse{}
	response.Body.LintExpressions = lintExpressions

	c.JSON(http.StatusOK, response.Body)
}

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	"github.com/replicatedhq/kots-lint/pkg/util"
)

// LintReleaseParameters contains parameters to lint a release for an app
type LintReleaseParameters struct {
	// Lint release parameters
	// In: body
	Body struct {
		// The spec to lint
		Spec string `json:"spec"`
	}
}

// LintReleaseResponse contains the lint expressions
type LintReleaseResponse struct {
	// JSON payload
	// Required: true
	// In: body
	Body struct {
		LintExpressions   []kots.LintExpression `json:"lintExpressions"`
		IsLintingComplete bool                  `json:"isLintingComplete"`
	}
}

// LintRelease http handler for linting a release
func LintRelease(c *gin.Context) {
	// read before binding to check if body is a tar stream
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	specFiles := kots.SpecFiles{}
	if util.IsTarFile(data) {
		f, err := kots.SpecFilesFromTarFile(data)
		if err != nil {
			log.Errorf("failed to get spec files from tar file: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		specFiles = f
	} else {
		// restore request body to its original state to be able to bind it
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(data))

		var request LintReleaseParameters
		if err := c.Bind(&request.Body); err != nil {
			log.Errorf("failed to bind to lint release parameters: %v", err)
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if err := json.Unmarshal([]byte(request.Body.Spec), &specFiles); err != nil {
			log.Errorf("failed to unmarshal spec: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	lintExpressions, isComplete, err := kots.LintSpecFiles(specFiles)
	if err != nil {
		fmt.Printf("failed to lint spec files: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	response := LintReleaseResponse{}
	response.Body.LintExpressions = lintExpressions
	response.Body.IsLintingComplete = isComplete

	c.JSON(http.StatusOK, response.Body)
}

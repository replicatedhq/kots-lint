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

// LintReleaseResponse contains the release properties
type LintReleaseResponse struct {
	// JSON payload
	// Required: true
	// In: body
	Body struct {
		LintExpressions   []kots.LintExpression `json:"lintExpressions"`
		IsLintingComplete bool                  `json:"isLintingComplete"`
	}
	Error struct {
		Message string `json:"message"`
	}
}

// Bind binds the gin context to the path parameters.
func (p *LintReleaseParameters) Bind(c *gin.Context) (err error) {
	if err = c.Bind(&p.Body); err != nil {
		return
	}
	return
}

// JSON serializes the AppResponse.Body as JSON into the response body.
// TODO: abstract this out
func (r LintReleaseResponse) JSON(c *gin.Context) {
	if r.Error.Message != "" {
		c.JSON(http.StatusBadRequest, r.Error)
	} else {
		c.JSON(http.StatusOK, r.Body)
	}
}

// LintRelease http handler for linting a release
func LintRelease(c *gin.Context) {
	// read before binding to check if body is a tar stream
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Infof("failed to read request body: %v", err)
		return
	}

	specFiles := []kots.SpecFile{}
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
		if err := request.Bind(c); err != nil {
			log.Infof("failed to bind to lint release parameters: %v", err)
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
		fmt.Printf("failed to lint app spec: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	response := LintReleaseResponse{}
	response.Body.LintExpressions = lintExpressions
	response.Body.IsLintingComplete = isComplete

	response.JSON(c)
}

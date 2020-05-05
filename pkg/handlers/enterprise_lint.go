package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/kots"
)

// EnterpriseLintReleaseParameters contains parameters to lint a release for an app
type EnterpriseLintReleaseParameters struct {
	// Lint release parameters
	// In: body
	Body struct {
		// The spec to lint
		Spec string `json:"spec" binding:"required"`

		// The policies to lint against
		Policies string `json:"policies" binding:"required"`
	}
}

// EnterpriseLintReleaseResponse contains the release properties
type EnterpriseLintReleaseResponse struct {
	// JSON payload
	// Required: true
	// In: body
	Body struct {
		LintExpressions []kots.LintExpression `json:"lintExpressions"`
	}
	Error struct {
		Message string `json:"message"`
	}
}

// EnterpriseLintRelease http handler for linting a release
func EnterpriseLintRelease(c *gin.Context) {
	var request EnterpriseLintReleaseParameters
	if err := c.Bind(&request.Body); err != nil {
		log.Infof("failed to bind to enterprise lint release parameters: %v", err)
		return
	}

	specFiles := kots.SpecFiles{}
	if err := json.Unmarshal([]byte(request.Body.Spec), &specFiles); err != nil {
		log.Errorf("failed to unmarshal spec: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	policies := []kots.EnterprisePolicy{}
	if err := json.Unmarshal([]byte(request.Body.Policies), &policies); err != nil {
		log.Errorf("failed to unmarshal policies: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	lintExpressions, err := kots.EnterpriseLintSpecFiles(specFiles, policies)
	if err != nil {
		fmt.Printf("failed to enterprise lint spec files: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	response := EnterpriseLintReleaseResponse{}
	response.Body.LintExpressions = lintExpressions

	c.JSON(http.StatusOK, response.Body)
}

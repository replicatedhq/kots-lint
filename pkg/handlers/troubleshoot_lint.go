package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	log "github.com/sirupsen/logrus"
)

// TroubleshootLintSpecParameters contains parameters to lint a troubleshoot spec
type TroubleshootLintSpecParameters struct {
	// Lint release parameters
	// In: body
	Body struct {
		// The spec to lint
		Spec string `json:"spec" binding:"required"`
	}
}

// TroubleshootLintSpecResponse contains the lint expressions
type TroubleshootLintSpecResponse struct {
	// JSON payload
	// Required: true
	// In: body
	Body struct {
		LintExpressions []kots.LintExpression `json:"lintExpressions"`
	}
}

// TroubleshootLintSpec http handler for linting a release
func TroubleshootLintSpec(c *gin.Context) {
	var request TroubleshootLintSpecParameters
	if err := c.Bind(&request.Body); err != nil {
		log.Errorf("failed to bind to troubleshoot lint spec parameters: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	lintExpressions, err := kots.TroubleshootLintSpec(request.Body.Spec)
	if err != nil {
		fmt.Printf("failed to troubleshoot lint spec: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	response := TroubleshootLintSpecResponse{}
	response.Body.LintExpressions = lintExpressions

	c.JSON(http.StatusOK, response.Body)
}

package daemon

import (
	"os"

	"github.com/gin-gonic/gin"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrgin/v1"
	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/handlers"
	cors "github.com/tommy351/gin-cors"
)

// Run is the main entry point of the kots lint.
func Run(newrelicApp newrelic.Application) {
	debugMode := os.Getenv("DEBUG_MODE")
	if debugMode != "on" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	if newrelicApp != nil {
		r.Use(nrgin.Middleware(newrelicApp))
	}

	r.RedirectTrailingSlash = false
	r.Use(
		cors.Middleware(cors.Options{
			AllowOrigins:  []string{"*"},
			AllowMethods:  []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Accept", "Content-Type"},
			ExposeHeaders: []string{"Content-Length"},
		}),
		log.GinLoggerWithWriter(
			gin.DefaultWriter,
			"/livez",
		),
		gin.Recovery(),
	)

	r.GET("/livez", handlers.GetLivez)

	v1 := r.Group("/v1")

	v1.POST("/lint", handlers.LintRelease)

	// Listen and Server on 0.0.0.0:8082
	err := r.Run(":8082")
	log.Errorf("Server exited unexpectedly: %v", err)
}

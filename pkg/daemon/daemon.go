package daemon

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/replicatedcom/saaskit/tracing/datadog"
	"github.com/replicatedhq/kots-lint/pkg/handlers"
	"github.com/replicatedhq/kots-lint/pkg/version"
	log "github.com/sirupsen/logrus"
	cors "github.com/tommy351/gin-cors"
)

// Run is the main entry point of the kots lint.
func Run() {
	datadog.StartTracer("kots-lint", version.GitSHA())
	defer datadog.StopTracer()

	debugMode := os.Getenv("DEBUG_MODE")
	if debugMode != "on" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: []string{"/livez"},
		}),
		gin.Recovery(),
		datadog.GinMiddleware("kots-lint"),
	)

	r.RedirectTrailingSlash = false
	r.Use(
		cors.Middleware(cors.Options{
			AllowOrigins:  []string{"*"},
			AllowMethods:  []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Accept", "Content-Type", "X-Datadog-Trace-Id", "X-Datadog-Parent-Id", "X-Datadog-Sampling-Priority", "X-Datadog-Origin", "Traceparent"},
			ExposeHeaders: []string{"Content-Length"},
		}),
	)

	r.GET("/livez", handlers.GetLivez)

	v1 := r.Group("/v1")

	v1.POST("/lint", handlers.LintRelease)
	v1.POST("/builders-lint", handlers.LintBuildersRelease)
	v1.POST("/enterprise-lint", handlers.EnterpriseLintRelease)
	v1.POST("/troubleshoot-lint", handlers.TroubleshootLintSpec)

	// Listen and Server on 0.0.0.0:8082
	err := r.Run(":8082")
	log.Errorf("Server exited unexpectedly: %v", err)
}

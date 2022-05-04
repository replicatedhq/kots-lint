package daemon

import (
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
	"github.com/replicatedhq/kots-lint/pkg/handlers"
	log "github.com/sirupsen/logrus"
	cors "github.com/tommy351/gin-cors"
)

// Run is the main entry point of the kots lint.
func Run() {
	tracer.Start(
		tracer.WithService("kots-lint"), 
		tracer.WithServiceVersion(version.GitSHA),
		tracer.WithAgentAddr("dd-agent.internal:8126"),
	)
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
		gintrace.Middleware("kots-lint"),
	)

	r.RedirectTrailingSlash = false
	r.Use(
		cors.Middleware(cors.Options{
			AllowOrigins:  []string{"*"},
			AllowMethods:  []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Accept", "Content-Type"},
			ExposeHeaders: []string{"Content-Length"},
		}),
	)

	r.GET("/livez", handlers.GetLivez)

	v1 := r.Group("/v1")

	v1.POST("/lint", handlers.LintRelease)
	v1.POST("/enterprise-lint", handlers.EnterpriseLintRelease)
	v1.POST("/troubleshoot-lint", handlers.TroubleshootLintSpec)

	// Listen and Server on 0.0.0.0:8082
	err := r.Run(":8082")
	log.Errorf("Server exited unexpectedly: %v", err)
}

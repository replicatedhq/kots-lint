package daemon

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/handlers"
	cors "github.com/tommy351/gin-cors"
)

// Run is the main entry point of the kots lint.
func Run() {
	debugMode := os.Getenv("DEBUG_MODE")
	if debugMode != "on" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

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

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetLivez(c *gin.Context) {
	c.Status(http.StatusOK)
}

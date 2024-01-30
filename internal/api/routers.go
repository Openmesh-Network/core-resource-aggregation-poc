package api

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

// HTTPHealthCheck provide an API for manually check the status of the peer
func HTTPHealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"OK": true})
}

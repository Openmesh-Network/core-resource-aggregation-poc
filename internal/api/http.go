package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"openmesh.network/aggregationpoc/internal/ipfs"
)

// HTTPInstance is a Gin HTTP server
type HTTPInstance struct {
	httpPort     int
	server       *http.Server
	GinServer    *gin.Engine
	globInstance *ipfs.Instance
}

// NewHTTPInstance create and return a new HTTP server
func NewHTTPInstance(httpPort int, ipfsInstance *ipfs.Instance) *HTTPInstance {
	s := gin.Default()

	// Register routers
	s.GET("/health", HTTPHealthCheck)
	s.GET("/ipfsidentity", func(c *gin.Context) {
		// This returns ipfs instance's libp2p address
		c.Data(http.StatusOK, "text/plain", []byte(ipfs.HostToString(ipfsInstance.Host)))
	})

	return &HTTPInstance{
		httpPort:  httpPort,
		GinServer: s,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", httpPort),
			Handler: s,
		},
	}
}

// Start starting the Gin HTTP server
func (i *HTTPInstance) Start() {
	go func() {
		// Start HTTP server
		if err := i.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP server: %s\n", err)
		}
	}()
}

// Stop shutdown the HTTP server
func (i *HTTPInstance) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := i.server.Shutdown(ctx); err != nil {
		log.Fatal("Failed to shutdown server:", err)
	}
}

package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// NewHTTPInstance create and return a new HTTP server
func NewHTTPInstance(httpPort int, ipfsInstance *ipfs.Instance) *HTTPInstance {
	gin.DefaultWriter = io.Discard

	s := gin.Default()
	s.Use(CORSMiddleware())

	// Register routers
	s.GET("/health", HTTPHealthCheck)
	s.GET("/ipfsidentity", func(c *gin.Context) {
		// This returns ipfs instance's libp2p address
		c.Data(http.StatusOK, "text/plain", []byte(ipfs.HostToString(ipfsInstance.Host)))
	})
	s.GET("/dashboard", func(c *gin.Context) {
		bytes, _ := os.ReadFile("index.html")
		c.Data(http.StatusOK, "text/html", []byte(bytes))
	})
	// XXX: if this goes anywhere near a production delete
	s.GET("/htmx/resize", func(c *gin.Context) {
		c.Request.ParseForm()
		for key, value := range c.Request.Form {
			if key == "size" {
				size, err := strconv.Atoi(value[0])

				if err == nil && size > 1 {
					ipfsInstance.StorageSize = size * 1024 * 1024
				}

				break
			}
		}

		s := ""
		// TODO(Tom): This is really gross, clean this up
		s += "<form hx-get=\"http://" + os.Getenv("XNODE_IP") + ":9080/htmx/resize\">\n"
		s += "<input class=\"sizeinput\" type=number name=\"size\" placeholder=\"Storage size in megabytes\" value=\"" + strconv.Itoa(ipfsInstance.StorageSize/(1024*1024)) + "\"/>"
		s += "</form>\n"

		c.Data(286, "text/html", []byte(s))
	})
	s.GET("/kill", func(c *gin.Context) {
		go func() {
			// so that we return the empty string before crashing
			time.Sleep(1 * time.Second)
			os.Exit(-1)
		}()

		c.Data(http.StatusOK, "text/html", []byte(""))
	})
	s.GET("/htmx/name", func(c *gin.Context) {
		// NOTE(Tom), I return 286 status for HTMX to stop polling this endpoint
		c.Data(286, "text/html", []byte(os.Getenv("XNODE_NAME")))
	})
	s.GET("/htmx/summary", func(c *gin.Context) {
		var s string

		bytesTotal := 0
		sourcesTotal := 0
		blocksTotal := 0
		ipfsInstance.BlockMapsMutex.Lock()
		for _, s := range ipfsInstance.Sources {
			hasABlockInSource := false
			for _, i := range ipfsInstance.BlocksSeeding[s.Name] {
				newBytes, _ := s.BlockSize(i)
				bytesTotal += int(newBytes)
				hasABlockInSource = true
				blocksTotal += 1
			}

			if hasABlockInSource {
				sourcesTotal += 1
			}
		}
		ipfsInstance.BlockMapsMutex.Unlock()

		s += "<p>Seeding " + strconv.Itoa(bytesTotal/1024) + "KB in " + strconv.Itoa(blocksTotal) + " blocks. "
		s += "For " + strconv.Itoa(sourcesTotal) + " sources.</p>\n"
		c.Data(http.StatusOK, "text/html", []byte(s))
	})
	s.GET("/htmx/blocks", func(c *gin.Context) {
		s := ""

		// NOTE(Tom): Have to do this to avoid race condition
		var blocksToSeed map[string][]int
		var blocksSeeding map[string][]int
		{
			ipfsInstance.BlockMapsMutex.Lock()
			defer ipfsInstance.BlockMapsMutex.Unlock()

			blocksToSeed = make(map[string][]int, len(ipfsInstance.BlocksToSeed))
			for k, v := range ipfsInstance.BlocksToSeed {
				blocksToSeed[k] = v
			}

			blocksSeeding = make(map[string][]int, len(ipfsInstance.BlocksSeeding))
			for k, v := range ipfsInstance.BlocksSeeding {
				blocksSeeding[k] = v
			}
		}

		for _, source := range ipfsInstance.Sources {
			s += "<div class=\"blockcontainer\">\n"
			for i := 0; i < int(source.BlockCount()); i++ {
				class := "offblock"

				for _, j := range blocksToSeed[source.Name] {
					if i == j {
						class = "wantblock"
						break
					}
				}

				for _, j := range blocksSeeding[source.Name] {
					if i == j {
						class = "onblock"
						break
					}
				}

				s += "<div class=\"" + class + "\"></div>"
			}
			s += "</div>\n"
		}

		c.Data(http.StatusOK, "text/html", []byte(s))
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

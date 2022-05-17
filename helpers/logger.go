package helpers

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

func DebugLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		// before request
		log.Printf("Service: %s\n", c.GetHeader("X-Service"))
		log.Printf("ContentType: %s\n", c.ContentType())
		log.Printf("Authorization: %s\n", c.GetHeader("Authorization"))
		log.Printf("Original URL: %s\n", c.GetHeader("X-Original-URI"))

		c.Next()
		// after request
		latency := time.Since(t)
		// access the status we are sending
		status := c.Writer.Status()
		log.Println("Processed time:", latency, "with status:", status)
	}
}

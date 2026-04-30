package ra

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

const readinessDialTimeout = 2 * time.Second

func (ra *Ra) readinessError() error {
	if ra == nil || ra.config == nil {
		return fmt.Errorf("ra config is not initialized")
	}
	if !ra.config.Proxy.Enabled {
		return nil
	}

	proxyURL, err := url.Parse(ra.config.Proxy.ProxyHost)
	if err != nil {
		return fmt.Errorf("parse proxy host: %w", err)
	}

	host := proxyURL.Host
	if host == "" {
		host = proxyURL.Path
	}
	if host == "" {
		return fmt.Errorf("proxy host is empty")
	}

	dialer := &net.Dialer{Timeout: readinessDialTimeout}
	conn, err := dialer.DialContext(context.Background(), "tcp", host)
	if err != nil {
		return fmt.Errorf("proxy host connection check failed: %w", err)
	}
	_ = conn.Close()
	return nil
}

func (ra *Ra) HandleHealthGin(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func (ra *Ra) HandleReadyGin(c *gin.Context) {
	if err := ra.readinessError(); err != nil {
		WriteJSONErrorGin(
			c,
			http.StatusServiceUnavailable,
			ErrorCodeReadyUnavailable,
			"Ra: service is not ready",
			err.Error(),
			DetailsWithReason(GinTraceID(c), err),
		)
		return
	}
	c.String(http.StatusOK, "ready")
}

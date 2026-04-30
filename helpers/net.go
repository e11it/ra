package helpers

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"
)

func IsAlive(targetURL *url.URL) error {
	dialer := net.Dialer{Timeout: time.Minute}
	conn, err := dialer.DialContext(context.Background(), "tcp", targetURL.Host)
	if err != nil {
		return fmt.Errorf("unreachable to %v, error: %v", targetURL.Host, err.Error())
	}
	defer func() { _ = conn.Close() }()
	return nil
}

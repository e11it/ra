package helpers

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

func IsAlive(url *url.URL) error {
	conn, err := net.DialTimeout("tcp", url.Host, time.Minute)
	if err != nil {
		return fmt.Errorf("Unreachable to %v, error: %v", url.Host, err.Error())
	}
	defer conn.Close()
	return nil
}

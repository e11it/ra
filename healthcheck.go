package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

const defaultHealthcheckURL = "http://127.0.0.1:8080/ready"

func resolveHealthcheckURL(cliURL string) string {
	if cliURL != "" {
		return cliURL
	}
	if envURL := os.Getenv("RA_HEALTHCHECK_URL"); envURL != "" {
		return envURL
	}
	return defaultHealthcheckURL
}

func runHealthcheck(rawURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("build healthcheck request: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do healthcheck request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected healthcheck status: %d", resp.StatusCode)
	}
	return nil
}

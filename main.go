package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/e11it/ra/helpers"
	RA "github.com/e11it/ra/internal/app/ra"
	"github.com/gin-gonic/gin"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	// log.SetLevel(log.DebugLevel)
}

func main() {
	ra, err := RA.NewRA(helpers.GetEnv("RA_CONFIG_FILE", "config.yml"))
	if err != nil {
		log.Err(err).Msg("config load error")
		os.Exit(1)
	}
	metrics := RA.NewMetrics()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metrics.GinMiddleware())
	router.Use(RA.ErrAbortHandlerSilenceMiddleware())
	router.Use(RA.AccessLogMiddleware(ra.AccessLogExcludePaths()))
	// router.Use(helpers.DebugLogger())

	router.GET("/auth",
		RA.GetUUIDMiddlerware(),
		ra.GetAuthMiddlerware(false),
		func(c *gin.Context) {
			c.String(http.StatusOK, "Auth")
		})

	if ra.ProxyEnabled() {
		router.Any("/topics/*proxyPath",
			RA.GetUUIDMiddlerware(),
			ra.GetAuthMiddlerware(true),
			ra.GetProxyHandler())
	}

	router.GET("/reload", func(c *gin.Context) {
		reloaded, err := ra.ReloadHandler()
		if err != nil {
			metrics.ObserveReload("error")
			RA.WriteJSONErrorGin(
				c,
				http.StatusBadRequest,
				RA.ErrorCodeReloadFailed,
				"Ra: config reload failed",
				err.Error(),
				RA.DetailsWithReason(RA.GinTraceID(c), err),
			)
			return
		}
		if reloaded {
			metrics.ObserveReload("changed")
			log.Info().Msg("config reloaded")
			c.String(http.StatusOK, "Reload")
		} else {
			metrics.ObserveReload("not_changed")
			log.Info().Msg("config not changed")
			c.String(http.StatusNotModified, "config not changed")
		}
	})
	router.GET("/metrics", gin.WrapH(metrics.Handler()))
	router.GET("/api/openapi/ra.yaml", func(c *gin.Context) {
		c.File(resolveOpenAPIFile("ra.yaml"))
	})
	router.GET("/api/openapi", func(c *gin.Context) {
		c.File(resolveOpenAPIFile("index.html"))
	})

	srv := &http.Server{
		Addr:              ra.GetServerAddr(),
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
		ErrorLog:          RA.HTTPServerErrorLog(),
	}
	log.Info().Msgf("Starting server on: %s", ra.GetServerAddr())
	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Msgf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	for {
		s, ok := <-quit
		if !ok {
			break
		}
		switch s {
		case syscall.SIGHUP:
			// TODO: перегрузка конфига
			if _, err := ra.ReloadHandler(); err != nil {
				log.Warn().Err(err).Msg("reload on SIGHUP failed")
			}
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info().Msg("shuting down server...")

			// The context is used to inform the server it has 5 seconds to finish
			// the request it is currently handling
			timeout, err := time.ParseDuration(fmt.Sprintf("%ds", ra.GetShutdownTimeout()))
			if err != nil {
				timeout = 5 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Error().Msgf("server forced to shutdown: %s", err.Error())
			}

			log.Info().Msg("server exiting")
			return
		}
	}
}

func resolveOpenAPIFile(name string) string {
	candidates := []string{
		filepath.Clean(filepath.Join("/app/api/openapi", name)),
		filepath.Clean(filepath.Join("api/openapi", name)),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

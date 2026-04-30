package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"

	"github.com/e11it/ra/helpers"
	RA "github.com/e11it/ra/internal/app/ra"
)

func main() {
	if runHealthcheckMode() {
		return
	}
	runServer()
}

func runHealthcheckMode() bool {
	healthcheckMode := flag.Bool("healthcheck", false, "run readiness probe and exit")
	healthcheckURL := flag.String("healthcheck-url", "", "readiness probe URL override")
	flag.Parse()
	if *healthcheckMode {
		targetURL := resolveHealthcheckURL(*healthcheckURL)
		if err := runHealthcheck(targetURL, 2*time.Second); err != nil {
			log.Error().Err(err).Str("url", targetURL).Msg("healthcheck failed")
			os.Exit(1)
		}
		os.Exit(0)
	}
	return false
}

func runServer() {
	ra, err := RA.NewRA(helpers.GetEnv("RA_CONFIG_FILE", "config.yml"))
	if err != nil {
		log.Err(err).Msg("config load error")
		os.Exit(1)
	}
	log.Info().Bool("compiled_with_company", RA.CompiledWithCompanyTag).Msg("build profile")
	metrics := RA.NewMetrics()
	router := buildRouter(ra, metrics)
	srv := &http.Server{
		Addr:              ra.GetServerAddr(),
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
		ErrorLog:          RA.HTTPServerErrorLog(),
	}
	startServer(srv, ra.GetServerAddr())
	waitForShutdownSignals(srv, ra)
}

func buildRouter(ra *RA.Ra, metrics *RA.Metrics) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metrics.GinMiddleware())
	router.Use(RA.ErrAbortHandlerSilenceMiddleware())
	router.Use(RA.AccessLogMiddleware(ra.AccessLogExcludePaths()))

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

	router.GET("/reload", buildReloadHandler(ra, metrics))
	router.GET("/metrics", gin.WrapH(metrics.Handler()))
	router.GET("/health", ra.HandleHealthGin)
	router.GET("/ready", ra.HandleReadyGin)
	router.GET("/swagger/ra.yaml", func(c *gin.Context) {
		c.File(resolveOpenAPIFile("ra.yaml"))
	})
	router.GET("/swagger", func(c *gin.Context) {
		c.File(resolveOpenAPIFile("index.html"))
	})
	return router
}

func buildReloadHandler(ra *RA.Ra, metrics *RA.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func startServer(srv *http.Server, addr string) {
	log.Info().Msgf("Starting server on: %s", addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Msgf("listen: %s\n", err)
		}
	}()
}

func waitForShutdownSignals(srv *http.Server, ra *RA.Ra) {
	quit := make(chan os.Signal, 1)
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

			timeout, err := time.ParseDuration(fmt.Sprintf("%ds", ra.GetShutdownTimeout()))
			if err != nil {
				timeout = 5 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			if err := srv.Shutdown(ctx); err != nil {
				log.Error().Msgf("server forced to shutdown: %s", err.Error())
			}
			cancel()

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

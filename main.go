package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
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
		log.Err(err)
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	// router.Use(helpers.DebugLogger())

	router.GET("/auth", ra.GetAuthMiddlerware(false), func(c *gin.Context) {
		c.String(http.StatusOK, "Auth")
	})

	if ra.ProxyEnabled() {
		router.Any("/topics/*proxyPath",
			RA.GetUUIDMiddlerware(),
			ra.GetAuthMiddlerware(true),
			ra.GetProxyHandler())
	}

	router.GET("/reload", func(c *gin.Context) {
		if err := ra.ReloadHandler(); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		log.Info().Msg("config reloaded")
		c.String(http.StatusOK, "Reload")
	})

	srv := &http.Server{
		Addr:              ra.GetServerAddr(),
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
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
			ra.ReloadHandler()
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info().Msg("shuting down server...")

			// The context is used to inform the server it has 5 seconds to finish
			// the request it is currently handling
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ra.GetShutdownTimeout())*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Error().Msgf("server forced to shutdown: %s", err.Error())
			}

			log.Info().Msg("server exiting")
			return
		}
	}
}

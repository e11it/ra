package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	RA "github.com/e11it/ra/internal/app/ra"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
)

func main() {
	ra, err := RA.NewRA(getEnv("RA_CONFIG_FILE", "config.yml"))
	if err != nil {
		log.Fatalln(err)
	}
	metrics := RA.NewMetrics()

	app := fiber.New(fiber.Config{
		ReadTimeout: 2 * time.Second,
		IdleTimeout: 30 * time.Second,
	})
	app.Use(metrics.FiberMiddleware())

	app.Post("/auth", ra.GetFiberAuthMiddlerware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/reload", func(c fiber.Ctx) error {
		reloaded, err := ra.ReloadHandler()
		if err != nil {
			metrics.ObserveReload("error")
			return RA.WriteJSONErrorFiber(
				c,
				fiber.StatusBadRequest,
				RA.ErrorCodeReloadFailed,
				"Ra: config reload failed",
				err.Error(),
				RA.DetailsWithReason(RA.FiberTraceID(c), err),
			)
		}
		if !reloaded {
			metrics.ObserveReload("not_changed")
			return c.SendStatus(fiber.StatusNotModified)
		}
		metrics.ObserveReload("changed")
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/metrics", adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})))
	app.Get("/api/openapi/ra.yaml", func(c fiber.Ctx) error {
		return c.SendFile(resolveOpenAPIFile("ra.yaml"))
	})
	app.Get("/api/openapi", func(c fiber.Ctx) error {
		return c.SendFile(resolveOpenAPIFile("index.html"))
	})
	go func() {
		if err := app.Listen(ra.GetServerAddr()); err != nil {
			log.Panic(err)
		}
	}()

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
				log.Printf("reload on SIGHUP failed: %v", err)
			}

			/* REMOVE
			updateConfig(Config, cs)
			auth_m.UpdateAuth(&Config.Auth)
			*/
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("shuting down server...")

			// The context is used to inform the server it has 5 seconds to finish
			// the request it is currently handling
			if err := app.Shutdown(); err != nil {
				log.Println("server forced to shutdown:", err)
			}

			log.Println("server exiting")
			return
		}
	}
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
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

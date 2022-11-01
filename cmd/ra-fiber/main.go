package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e11it/ra/internal/app/ra"
	"github.com/gofiber/fiber/v2"
)

func main() {
	ra, err := ra.NewRA(getEnv("RA_CONFIG_FILE", "config.yml"))
	if err != nil {
		log.Fatalln(err)
	}

	app := fiber.New(fiber.Config{
		ReadTimeout: 2 * time.Second,
		IdleTimeout: 30 * time.Second,
	})

	app.Post("/auth", ra.GetFiberAuthMiddlerware(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/reload", func(c *fiber.Ctx) error {
		if err := ra.ReloadHandler(); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.SendStatus(fiber.StatusOK)
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
			ra.ReloadHandler()

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

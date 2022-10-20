//go:build go1.8
// +build go1.8

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e11it/ra/auth"
	"github.com/e11it/ra/checksum"
	ginlogrus "github.com/e11it/ra/ginlogrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/configor"

	log "github.com/sirupsen/logrus"
)

const path string = "example/config.yml"

type config struct {
	APPName  string `default:"app name"`
	Addr     string `default:":8080"`
	LogLevel string `default:""`

	Auth auth.Config

	ShutdownTimeout uint `default:"5"`
}

type Authorizer interface {
	GetMiddleware() gin.HandlerFunc
}

type SumChecker interface {
	CompareCheckSum(newPath string) (ok bool)
}

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func createAuthRouter(auth_m Authorizer) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(ginlogrus.Logger(), gin.Recovery())
	// router.Use(helpers.DebugLogger())

	router.Use(auth_m.GetMiddleware())
	router.GET("/auth", func(c *gin.Context) {
		c.String(http.StatusOK, "Auth")
	})
	return router, nil
}

func loadConfig(cfg *config) {
	if err := configor.New(&configor.Config{Verbose: false}).Load(cfg, path); err != nil {
		log.WithError(err).Fatalln("Can't parse config")
	}
	log.Infoln("load config")
}

func updateConfig(cfg *config, cs SumChecker) {
	if cs.CompareCheckSum(path) {
		log.Warningln("Config is the same")
		return
	}
	loadConfig(cfg)
}

func main() {
	// Log as JSON instead of the default ASCII formatter.

	Config := new(config)

	os.Setenv("CONFIGOR_ENV_PREFIX", "RA")

	cs, err := checksum.NewChecksum(path)
	if err != nil {
		log.WithError(err).Fatalln("Can't get checksum module")
	}
	loadConfig(Config)

	auth_m, err := auth.NewAuth(&Config.Auth)
	if err != nil {
		log.WithError(err).Fatalln("Can't init auth module")
	}

	// method | path | user
	router, err := createAuthRouter(auth_m)
	if err != nil {
		log.Fatalf("Error create auth: %s\n", err.Error())
	}

	srv := &http.Server{
		Addr:    Config.Addr,
		Handler: router,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
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
			updateConfig(Config, cs)
			auth_m.UpdateAuth(&Config.Auth)
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("Shuting down server...")

			// The context is used to inform the server it has 5 seconds to finish
			// the request it is currently handling
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(Config.ShutdownTimeout)*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Println("Server forced to shutdown:", err)
			}

			log.Println("Server exiting")
			return
		}
	}

}

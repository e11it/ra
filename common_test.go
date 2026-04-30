package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/e11it/ra/internal/app/ra"
)

func createTestingAuthRouter(path string) *gin.Engine {
	newRa, err := ra.NewRA(path)
	if err != nil {
		return nil
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(newRa.GetAuthMiddlerware(false))
	router.GET("/auth", func(c *gin.Context) {
		c.String(http.StatusOK, "Auth")
	})
	router.POST("/auth", func(c *gin.Context) {
		c.String(http.StatusOK, "Auth")
	})
	return router
}

func testGetAuthServer() *gin.Engine {
	return createTestingAuthRouter("example/_test/auth_server_config.yml")
}

func testGetBodyValidationServer() *gin.Engine {
	return createTestingAuthRouter("example/_test/body_validation_config.yml")
}

func testGetSRServer() *gin.Engine {
	return createTestingAuthRouter("example/_test/sr_server_config.yml")
}

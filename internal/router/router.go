package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/venkxycodes/ab-experiments/internal/handler"
)

func New(experimentHandler *handler.ExperimentHandler) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/v1")
	{
		v1.POST("/experiments", experimentHandler.CreateExperiment)
		v1.GET("/experiments", experimentHandler.ListExperiments)
		v1.GET("/experiments/:key", experimentHandler.GetExperiment)
		v1.POST("/experiments/:key/resolve", experimentHandler.Resolve)
	}
	return engine
}

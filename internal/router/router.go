package router

import (
	"fmt"
	"service/internal/router/config"
	"service/internal/router/playground"
	"service/internal/router/unittest"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var Router *gin.Engine

func Run() {
	hostname := viper.GetString("hostname")
	port := viper.GetInt("port")
	Router.Run(fmt.Sprintf("%s:%v", hostname, port))
}

func setupRouter() {
	Router = gin.Default()
	Router.Use(cors.New(cors.Config{
		AllowOrigins: viper.GetStringSlice("allow-origins"),
	}))
}

func SetupConfigService() {
	setupRouter()

	c := Router.Group("/config")
	{
		c.POST("/:config_id", config.GetConfig)
	}
}

func SetupTestService() {
	setupRouter()

	t := Router.Group("/test")
	{
		t.GET("/:test_id", unittest.ExecuteTest)
	}
}

func SetupPlayground() {
	setupRouter()

	pg := Router.Group("/playground")
	{
		pg.POST("/starlark", playground.Starlark)
		pg.POST("/js", playground.JavaScript)
	}
}

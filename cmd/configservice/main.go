package main

import (
	"service/internal/engine/javascript"
	"service/internal/model"
	"service/internal/redis"
	"service/internal/router"

	"github.com/spf13/viper"
)

func main() {
	// initialize config
	viper.SetConfigFile("configfile/config.yml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	model.Run()
	redis.Setup()

	// init runners
	if err := javascript.Init(); err != nil {
		panic(err)
	}

	router.SetupConfigService()
	router.Run()
}

package main

import (
	"service/internal/model"
	"service/internal/router/push"

	"github.com/spf13/viper"
)

func main() {
	// initialize config
	viper.SetConfigFile("configfile/config.yml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	model.Run()

	model.DB.AutoMigrate(&model.ErrorReport{})

	constaintName := "fk_code_error_reports"
	if !model.DB.Migrator().HasConstraint(&model.Code{}, constaintName) {
		model.DB.Migrator().CreateConstraint(&model.Code{}, constaintName)
	}

	if !model.DB.Migrator().HasColumn(&model.Code{}, "is_broken") {
		model.DB.Migrator().AddColumn(&model.Code{}, "is_broken")
	}

	if !model.DB.Migrator().HasColumn(&model.Code{}, "err_count") {
		model.DB.Migrator().AddColumn(&model.Code{}, "err_count")
	}

	push.Setup()
	push.Run()
}

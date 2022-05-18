package model

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Run() {
	var err error
	config := viper.GetStringMap("mysql")
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local",
		config["username"],
		config["password"],
		config["hostname"],
		config["port"],
		config["database"],
	)
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Fail to connect to database: ", err)
	}

	// not running auto migrate as the config service only
	// reads the database managed by the management platform
}

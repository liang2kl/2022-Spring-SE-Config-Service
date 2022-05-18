package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"service/internal/engine"
	"service/internal/engine/javascript"
	"service/internal/engine/starlark"
	"service/internal/model"
	"service/internal/redis"
	"service/internal/router/resp"
	"service/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type GetConfigBody struct {
	Meta   model.ConfigMeta       `json:"meta"`
	Cached bool                   `json:"cached"`
	Params map[string]interface{} `json:"params"`
}

func GetConfig(c *gin.Context) {
	configId := c.Param("config_id")

	// get config from http body
	configBody := GetConfigBody{}
	err := c.ShouldBindJSON(&configBody)

	cached := configBody.Cached

	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid arguments")
		return
	}

	config, err := getConfig(configId, cached)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, fmt.Sprintf("fail to get record: %v", err))
		return
	}

	if !config.IsValid() {
		resp.Error(c, http.StatusForbidden, "the config is not active")
		return
	}

	// get code
	var code model.Code

	grayHit := config.Percentage > 0 &&
		utils.Hash(configBody.Meta.DeviceID+config.GrayReleaseCode)%100 < uint32(config.Percentage)

	if grayHit {
		// use gray release version
		code, err = getCode(config.GrayReleaseCode, cached)
		if err != nil {
			log.Printf("fail to find gray release code %s for config %s: %v",
				config.GrayReleaseCode, configId, err)
		}
	}

	// if not hit, or the gray release code cannot be used
	if !grayHit || err != nil || code.IsBroken {
		// use stable release version
		code, err = getCode(config.ReleasedCode, cached)
		if err != nil {
			log.Printf("fail to find code %s for config %s: %v",
				config.ReleasedCode, configId, err)
		}
	}

	if err != nil {
		resp.Error(c, http.StatusInternalServerError,
			fmt.Sprintf("fail to get retrive record: %v", err))
		return
	}

	if code.IsBroken {
		resp.Error(c, http.StatusForbidden, "the requested config is broken")
		return
	}

	// validate config and parameters
	if ok, err := code.ValidateRules(configBody.Meta); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	} else if !ok {
		resp.Error(c, http.StatusBadRequest, "request rejected by predefined rules")
		return
	}

	params, err := code.ValidateParams(configBody.Params)

	if err != nil {
		resp.Error(c, http.StatusBadRequest, "fail to validate: "+err.Error())
		return
	}

	data, err := json.Marshal(params)

	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "internal error: "+err.Error())
		return
	}

	var res engine.RunResult

	// use config id + code id as compiled code cached id
	cacheId := configId + code.CodeID
	if !configBody.Cached {
		cacheId = ""
	}

	if code.Lang == "starlark" {
		res = starlark.Run(cacheId, code.Content, string(data))
	} else if code.Lang == "javascript" {
		res = javascript.Run(cacheId, code.Content, string(data))
	} else {
		msg := "invalid lang " + code.Lang
		log.Print(msg)
		resp.Error(c, http.StatusInternalServerError, "internal error: "+msg)
		return
	}

	if res.Err != nil {
		resp.Error(c, http.StatusBadRequest, "execution failed: "+res.Err.Error())
		return
	}

	resp.Ok(c, http.StatusOK, map[string]interface{}{
		"result":  res.Val,
		"code_id": code.CodeID,
	})
}

func getConfigCacheKey(configId string) string {
	return "config/" + configId
}

func getCodeCacheKey(codeId string) string {
	return "code/" + codeId
}

func getConfig(configId string, cached bool) (model.Config, error) {
	cacheKey := getConfigCacheKey(configId)

	// check redis
	if cached {
		if config, err := redis.Get[model.Config](cacheKey); err == nil || err != redis.ErrGet {
			return *config, err
		}
	}

	// no-cached, cache miss or error occurs
	var config model.Config
	if err := model.DB.First(&config, "config_id = ?", configId).Error; err != nil {
		return config, err
	}

	// cache the config in redis
	expiration := viper.GetDuration("redis-expiration")
	if err := redis.Set(cacheKey, config, expiration); err != nil {
		log.Print(err)
	}

	return config, nil
}

func getCode(codeId string, cached bool) (model.Code, error) {
	cacheKey := getCodeCacheKey(codeId)

	// check redis
	if cached {
		if code, err := redis.Get[model.Code](cacheKey); err == nil || err != redis.ErrGet {
			return *code, err
		}
	}

	// no-cached, cache miss or error occurs
	var code model.Code
	if err := model.DB.First(&code, "code_id = ?", codeId).Error; err != nil {
		return code, err
	}

	// cache the config in redis
	expiration := viper.GetDuration("redis-expiration")
	if err := redis.Set(cacheKey, code, expiration); err != nil {
		log.Print(err)
	}

	return code, nil
}

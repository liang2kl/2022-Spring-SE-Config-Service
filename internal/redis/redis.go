package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var ctx = context.Background()
var Client redis.Client

var ErrGet error = errors.New("fail to retrive value from redis")

func Setup() {
	config := viper.GetStringMap("redis")
	Client = *redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%v:%v", config["hostname"], config["port"]),
	})
}

func Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return SetString(key, string(data), expiration)
}

func Get[T interface{}](key string) (*T, error) {
	str, err := GetString(key)
	if err != nil {
		return nil, ErrGet
	}

	var value T
	err = json.Unmarshal([]byte(str), &value)
	return &value, err
}

func SetString(key string, value string, expiration time.Duration) error {
	err := Client.Set(ctx, key, value, expiration).Err()
	return err
}

func GetString(key string) (string, error) {
	val, err := Client.Get(ctx, key).Result()
	return val, err
}

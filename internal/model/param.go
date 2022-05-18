package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ParamArray []Param

func (params *ParamArray) Scan(value interface{}) error {
	val, ok := value.([]uint8)
	if !ok {
		return errors.New("fail to retrive string value for 'config.param'")
	}
	err := json.Unmarshal(val, params)
	return err
}

func (params *ParamArray) Value() (driver.Value, error) {
	return json.Marshal(params)
}

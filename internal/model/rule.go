package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Rule struct {
	Field   string `json:"field"`
	Compare string `json:"compare"`
	Value   string `json:"value"`
}

type PlatformRule struct {
	Platform string   `json:"platform"`
	Rules    [][]Rule `json:"rules"`
}

type PlatformRuleArray []PlatformRule

func (rules *PlatformRuleArray) Scan(value interface{}) error {
	val, ok := value.([]uint8)
	if !ok {
		return errors.New("fail to retrive value for 'config.rules'")
	}

	err := json.Unmarshal(val, rules)
	return err
}

func (rules *PlatformRuleArray) Value() (driver.Value, error) {
	return json.Marshal(rules)
}

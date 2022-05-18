package model

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
)

type Code struct {
	ID      int               `gorm:"column:id;primaryKey;<-:false"`
	CodeID  string            `gorm:"column:code_id;<-:false"`
	Lang    string            `gorm:"column:lang;<-:false"`
	Rules   PlatformRuleArray `gorm:"column:rules;<-:false"`
	Params  ParamArray        `gorm:"column:params;<-:false"`
	Content string            `gorm:"column:code;<-:false"`

	// following fields are for error control
	IsBroken     bool          `gorm:"column:is_broken;default:false"`
	ErrorCount   int           `gorm:"column:err_count;default:0"`
	ErrorReports []ErrorReport `gorm:"foreignKey:CodeRef;references:ID"`
}

func (Code) TableName() string {
	return "code"
}

type ErrorReport struct {
	ID      uint   `gorm:"column:id;primaryKey;auto_increment;not_null"`
	Time    int    `gorm:"column:time"`
	Message string `gorm:"column:message"`
	CodeRef int    `gorm:"column:code_ref"`
}

func (ErrorReport) TableName() string {
	return "error_report"
}

func (code *Code) ValidateRules(meta ConfigMeta) (bool, error) {
	// validate rules
	valid := true

	for _, platformRule := range code.Rules {
		if platformRule.Platform != meta.Platform {
			continue
		}

		// OR between super rules
		valid = len(platformRule.Rules) == 0
		for _, supRules := range platformRule.Rules {
			if len(supRules) == 0 {
				continue
			}
			// AND within a sub rule
			subRuleValid := true
			for _, subRule := range supRules {
				compare := subRule.Compare

				if subRule.Field == "version" {
					fieldVal := meta.Version
					ruleVal, err := strconv.Atoi(subRule.Value)
					if err != nil {
						msg := "internal error: " + err.Error()
						log.Println(msg)
						return false, errors.New(msg)
					}

					var res bool

					switch compare {
					case "<":
						res = fieldVal < ruleVal
					case "<=":
						res = fieldVal <= ruleVal
					case "=":
						res = fieldVal == ruleVal
					case ">":
						res = fieldVal > ruleVal
					case ">=":
						res = fieldVal >= ruleVal
					default:
						msg := "unexpected comparer: " + compare
						log.Println(msg)
						return false, errors.New("internal error: " + msg)
					}

					subRuleValid = subRuleValid && res
				} else {
					msg := "unrecognized field: " + subRule.Field
					log.Println(msg)
					return false, errors.New("internal error: " + msg)
				}
			}

			valid = valid || subRuleValid
		}
	}
	return valid, nil
}

func (code *Code) ValidateParams(params map[string]interface{}) (map[string]interface{}, error) {
	ret := map[string]interface{}{}

	for _, param := range code.Params {
		val, exist := params[param.Name]
		if !exist {
			return ret, fmt.Errorf("missing key '%s' in parameters", param.Name)
		}

		var targetType reflect.Kind

		switch param.Type {
		case "string":
			targetType = reflect.String
		case "int":
			// json parser parses int into float
			targetType = reflect.Float64
		case "float":
			targetType = reflect.Float64
		case "bool":
			targetType = reflect.Bool
		case "array":
			targetType = reflect.Slice
		case "dictionary":
			targetType = reflect.Map
		default:
			return ret, fmt.Errorf("unexpedted type '%s' for param '%s'", param.Type, param.Name)
		}

		valType := reflect.TypeOf(val).Kind()

		if targetType != valType {
			return ret, fmt.Errorf(
				"mismatched type '%s' for parameter '%s': expected '%s'",
				valType.String(), param.Name, param.Type,
			)
		}

		if param.Type == "int" {
			val := val.(float64)
			ret[param.Name] = int(val)
		} else {
			ret[param.Name] = val
		}
	}

	return ret, nil
}

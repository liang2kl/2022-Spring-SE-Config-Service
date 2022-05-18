package config_test

import "service/internal/model"

func createCode(rules model.PlatformRuleArray) model.Code {
	return model.Code{
		CodeID: "100000",
		Rules:  rules,
		Lang:   "starlark",
	}
}

const HitDeviceID = "3"
const MissDeviceID = "1"

var ReleasedCodes = map[string]model.Code{
	"release": {
		CodeID:  "1",
		Content: "return 'release'",
		Lang:    "starlark",
	},
	"grayrelease": {
		CodeID:  "2",
		Content: "return 'grayrelease'",
		Lang:    "starlark",
	},
}

var Codes = map[string]model.Code{
	"empty": createCode(model.PlatformRuleArray{}),
	"single": createCode(model.PlatformRuleArray{
		model.PlatformRule{
			Platform: "iphone",
			Rules: [][]model.Rule{
				{{
					Field:   "version",
					Compare: ">=",
					Value:   "10",
				}},
			},
		},
	}),
	"single_and": createCode(model.PlatformRuleArray{
		model.PlatformRule{
			Platform: "iphone",
			Rules: [][]model.Rule{
				{
					{
						Field:   "version",
						Compare: ">=",
						Value:   "1",
					},
					{
						Field:   "version",
						Compare: "<=",
						Value:   "5",
					},
				},
			},
		},
	}),
	"multiple": createCode(model.PlatformRuleArray{
		model.PlatformRule{
			Platform: "iphone",
			Rules: [][]model.Rule{
				{{
					Field:   "version",
					Compare: ">=",
					Value:   "10",
				}},
				{{
					Field:   "version",
					Compare: "<=",
					Value:   "5",
				}},
			},
		},
	}),
	// 1~5 or 10~15
	"multiple_and": createCode(model.PlatformRuleArray{
		model.PlatformRule{
			Platform: "iphone",
			Rules: [][]model.Rule{
				{
					{
						Field:   "version",
						Compare: ">=",
						Value:   "1",
					},
					{
						Field:   "version",
						Compare: "<=",
						Value:   "5",
					},
				},
				{
					{
						Field:   "version",
						Compare: ">=",
						Value:   "10",
					},
					{
						Field:   "version",
						Compare: "<=",
						Value:   "15",
					},
				},
			},
		},
	}),
}

package model

type ConfigMeta struct {
	Version  int    `json:"version"`
	Platform string `json:"platform"`
	DeviceID string `json:"device_id"`
}

type Config struct {
	ConfigID        string `gorm:"column:config_id;<-:false"`
	ReleasedCode    string `gorm:"column:code_release;<-:false"`
	TestCode        string `gorm:"column:code_unittest;<-:false"`
	GrayReleaseCode string `gorm:"column:code_gray;<-:false"`
	Percentage      int    `gorm:"column:percentage;<-:false"`
	Status          string `gorm:"column:status;<-:false"`
	Secret          string `gorm:"column:secret;<-:false"`
}

func (Config) TableName() string {
	return "config"
}

func (config Config) IsValid() bool {
	return config.Status == "valid"
}

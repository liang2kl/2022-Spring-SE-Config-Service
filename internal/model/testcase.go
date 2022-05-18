package model

type TestInput map[string]string

type TestCase struct {
	TestID string `gorm:"column:test_id;<-:false"`
	Input  string `gorm:"column:input;<-:false"`
	Output string `gorm:"column:output;<-:false"`
	CodeID string `gorm:"column:code_id;<-:false"`
}

func (TestCase) TableName() string {
	return "unittest"
}

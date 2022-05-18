package javascript_test

import (
	"fmt"
	"os"
	"service/internal/engine/javascript"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	err := javascript.Init()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.SetDefault("timeout", 50)

	code := m.Run()
	os.Exit(code)
}

func TestParams(t *testing.T) {
	code := "return p.version_code;"
	params := `{"version_code": 1024}`
	res := javascript.Run("", code, params)

	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "1024", res.Val, "wrong returned result")
}

func TestMultiline(t *testing.T) {
	code := `
if (p.version_code > 100){
	return "valid"
} else {
	return "invalid"
}
`
	params := `{"version_code": 1024}`
	res := javascript.Run("", code, params)
	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "valid", res.Val, "wrong returned result")
}

func TestInlineFunc(t *testing.T) {
	code := `
function judge(version) {
	if (version > 100){
		return "valid"
	} else {
		return "invalid"
	}
}

return judge(p.version_code)
`
	params := `{"version_code": 1024}`
	res := javascript.Run("", code, params)
	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "valid", res.Val, "wrong returned result")
}

func TestTimeout(t *testing.T) {
	code := `
var i = 0;
while (i < 10000000000) {
	i += 1
}
`
	params := `{}`
	res := javascript.Run("", code, params)
	assert.Error(t, res.Err, "fail to exit on timeout")
}

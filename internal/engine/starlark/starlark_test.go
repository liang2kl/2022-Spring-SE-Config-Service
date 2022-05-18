package starlark_test

import (
	"fmt"
	"os"
	"service/internal/engine/starlark"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.SetDefault("timeout", 50)

	code := m.Run()
	os.Exit(code)
}

func TestParams(t *testing.T) {
	code := `return p["version_code"]`
	params := `{"version_code": 1024}`
	res := starlark.Run("", code, params)
	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "1024", res.Val, "wrong returned result")
}

func TestMultiline(t *testing.T) {
	code := `
if p["version_code"] > 100:
	return "valid"
else:
	return "invalid"
`
	params := `{"version_code": 1024}`
	res := starlark.Run("", code, params)
	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "\"valid\"", res.Val, "wrong returned result")
}

func TestInlineFunc(t *testing.T) {
	code := `
def judge(version):
	if version > 100:
		return True
	else:
		return False

return judge(p["version_code"])
`
	params := `{"version_code": 1024}`
	res := starlark.Run("", code, params)
	assert.NoError(t, res.Err, "runner returned an error")
	assert.Equal(t, "True", res.Val, "wrong returned result")
}

func TestTimeout(t *testing.T) {
	code := `
for i in range(100000000000):
	pass
`
	params := `{}`
	res := starlark.Run("", code, params)
	// assert.NoError(t, res.Err)
	assert.Error(t, res.Err, "fail to exit on timeout")
}

func TestHitCodeCache(t *testing.T) {
	for i := 0; i < 2; i++ {
		res := starlark.Run("id", "", "{}")
		assert.NoError(t, res.Err, "runner returned an error")
	}

	starlark.ClearCache("id")
}

func TestConcurrentCodeCacheAccess(t *testing.T) {
	const num = 1000
	ch := make(chan int, num)
	for i := 0; i < num; i++ {
		go func(i int) {
			id := fmt.Sprintln(i)
			starlark.Run(id, "", "{}")
			starlark.ClearCache(id)
			ch <- 1
		}(i)
	}

	resultNum := 0

	for range ch {
		resultNum++
		if resultNum == num {
			break
		}
	}
}

package config_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"service/internal/model"
	"service/internal/redis"
	"service/internal/router"
	"service/internal/router/config"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var mock sqlmock.Sqlmock
var redisMock redismock.ClientMock

func testRequest(method string, path string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	router.Router.ServeHTTP(w, req)

	return w
}

func createBody(meta model.ConfigMeta, params map[string]interface{}) []byte {
	data, _ := json.Marshal(config.GetConfigBody{
		Meta:   meta,
		Params: params,
		Cached: false,
	})
	return data
}

func configRow(config model.Config) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"config_id", "code_release", "code_unittest",
		"code_gray", "percentage", "secret", "status",
	}).AddRow(
		config.ConfigID, config.ReleasedCode,
		config.TestCode, config.GrayReleaseCode, config.Percentage, config.Secret,
		config.Status,
	)
}

func codeRow(code model.Code) *sqlmock.Rows {
	rules, _ := json.Marshal(code.Rules)
	params, _ := json.Marshal(code.Params)

	return sqlmock.NewRows([]string{"code_id", "code", "rules", "params", "lang"}).
		AddRow(code.CodeID, code.Content, rules, params, code.Lang)
}

func setConfigMockReturn(config model.Config) {
	cacheKey := "config/" + config.ConfigID
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM `+"`config`") + "(.+)").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows((configRow(config)))

	redisMock.
		Regexp().
		ExpectSet(cacheKey, `.*`, time.Duration(viper.GetInt("redis-expiration"))*time.Second)
}

func setCodeMockReturn(code model.Code) {
	cacheKey := "code/" + code.CodeID
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM `+"`code`") + "(.+)").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows((codeRow(code)))

	redisMock.
		Regexp().
		ExpectSet(cacheKey, `.*`, time.Duration(viper.GetInt("redis-expiration"))*time.Second)
}

func TestMain(m *testing.M) {
	// set default configuration values
	viper.SetDefault("timeout", 50)
	viper.SetDefault("allow-origins", []string{"*"})
	viper.SetDefault("redis-expiration", 60)

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	mock = sqlMock

	client, mock := redismock.NewClientMock()
	redis.Client = *client
	redisMock = mock

	model.DB, err = gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})

	if err != nil {
		os.Exit(1)
	}

	router.SetupConfigService()

	m.Run()
}

func TestNonExistingRecord(t *testing.T) {
	body := createBody(model.ConfigMeta{}, map[string]interface{}{})

	mock.ExpectQuery("SELECT(.+)").
		WithArgs(sqlmock.AnyArg())

	w := testRequest("POST", "/config/121213", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidJson(t *testing.T) {
	w := testRequest("POST", "/config/100000", []byte{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInactiveConfig(t *testing.T) {
	body := createBody(model.ConfigMeta{}, map[string]interface{}{})

	setConfigMockReturn(model.Config{
		Status: "invalid",
	})

	w := testRequest("POST", "/config/100000", body)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func testRule(codeName string, meta model.ConfigMeta) int {

	setConfigMockReturn(model.Config{
		ConfigID:     "100000",
		ReleasedCode: codeName,
		Status:       "valid",
	})

	// some requests fail before fetching code
	setCodeMockReturn(Codes[codeName])

	body := createBody(meta, map[string]interface{}{})

	w := testRequest("POST", "/config/100000", body)

	return w.Code
}

func TestEmptyRules(t *testing.T) {
	code := testRule("empty", model.ConfigMeta{
		Version: 9, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)
}

func TestOtherPlatform(t *testing.T) {
	code := testRule("single", model.ConfigMeta{
		Version: 9, Platform: "android",
	})
	assert.Equal(t, http.StatusOK, code)
}

func TestInvalidSingleRule(t *testing.T) {
	code := testRule("single", model.ConfigMeta{
		Version: 9, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestValidSingleRule(t *testing.T) {
	code := testRule("single", model.ConfigMeta{
		Version: 10, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)
}

func TestInvalidSingleAndRule(t *testing.T) {
	code := testRule("single_and", model.ConfigMeta{
		Version: 7, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)

	code = testRule("single_and", model.ConfigMeta{
		Version: 0, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)

}

func TestValidSingleAndRule(t *testing.T) {
	code := testRule("single_and", model.ConfigMeta{
		Version: 4, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)
}

func TestInvalidMultipleRules(t *testing.T) {
	code := testRule("multiple", model.ConfigMeta{
		Version: 9, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestValidMultipleRules(t *testing.T) {
	code := testRule("multiple", model.ConfigMeta{
		Version: 15, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)

	code = testRule("multiple", model.ConfigMeta{
		Version: 4, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)
}

func TestInvalidMultipleOrRules(t *testing.T) {
	code := testRule("multiple_and", model.ConfigMeta{
		Version: 0, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)

	code = testRule("multiple_and", model.ConfigMeta{
		Version: 6, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)

	code = testRule("multiple_and", model.ConfigMeta{
		Version: 16, Platform: "iphone",
	})
	assert.Equal(t, http.StatusBadRequest, code)

}

func TestValidMultipleOrRules(t *testing.T) {
	code := testRule("multiple", model.ConfigMeta{
		Version: 15, Platform: "iphone",
	})
	assert.Equal(t, http.StatusOK, code)
}

func testReleaseState(hit bool) (int, string, error) {
	setConfigMockReturn(model.Config{
		ConfigID:   "100000",
		Percentage: 50,
		Status:     "valid",
	})

	name := "release"
	if hit {
		name = "grayrelease"
	}

	setCodeMockReturn(ReleasedCodes[name])

	deviceID := MissDeviceID

	if hit {
		deviceID = HitDeviceID
	}

	body := createBody(model.ConfigMeta{
		Platform: "",
		DeviceID: deviceID,
	}, map[string]interface{}{})

	w := testRequest("POST", "/config/100000", body)

	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)

	if err != nil {
		return 0, "", err
	}

	return w.Code, res["data"].(map[string]interface{})["result"].(string), nil
}

func TestHitGrayRelease(t *testing.T) {
	code, res, err := testReleaseState(false)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code, "wrong return code")
	assert.Equal(t, "\"release\"", res, "wrong gray scale hit")
}

func TestMissGrayRelease(t *testing.T) {
	code, res, err := testReleaseState(true)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code, "wrong return code")
	assert.Equal(t, "\"grayrelease\"", res, "wrong gray scale hit")
}

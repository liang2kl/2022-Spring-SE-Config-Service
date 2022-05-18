package unittest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"service/internal/model"
	"service/internal/router"
	"service/internal/router/resp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const secret = "magid"

var mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	// set default configuration values
	viper.SetDefault("test-secret", secret)
	viper.SetDefault("timeout", 50)
	viper.SetDefault("allow-origins", []string{"*"})

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	mock = sqlMock

	model.DB, err = gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})

	if err != nil {
		os.Exit(1)
	}

	router.SetupTestService()

	m.Run()
}

const configID = "123"
const testID = "234"
const codeID = "345"
const testCode = `
return {
	"arr": [1, 2],
	"obj": {
		"obj": {
			"val": 3
		},
		"val": "string"
	}
}
`

const validOutput = `
{
	"obj": {
		"val": "string",
		"obj": {
			"val": 3
		}
	},
	"arr": [1, 2]
}
`

var invalidOutputs = []string{`
{
	"obj": {
		"val": "string",
		"obj": {
			"val": "3"
		}
	},
	"arr": [1, 2]
}
`,
	`{
	"obj": {
		"val": "string",
		"obj": {
			"val": 3
		}
	},
	"arr": []
}
`,
}

func testRequest(output string) (int, resp.Response) {
	setMockReturn(output)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/"+testID, bytes.NewReader([]byte{}))
	req.Header.Add("Secret", secret)
	router.Router.ServeHTTP(w, req)

	var response resp.Response
	json.Unmarshal(w.Body.Bytes(), &response)

	return w.Code, response
}

func testCaseRow(output string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"test_id", "input", "output", "config_id",
	}).AddRow(
		testID, []byte("{}"), output, configID,
	)
}

func codeRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"code_id", "code", "rules", "params", "lang",
	}).AddRow(
		codeID, testCode, []byte("[]"), []byte("[]"), "starlark",
	)
}

func setMockReturn(output string) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM `+"`unittest`") + "(.+)").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(testCaseRow(output))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM `+"`code`") + "(.+)").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(codeRow())
}

func getTestStatus(response resp.Response) bool {
	res := response.Data.(map[string]interface{})
	return res["data"].(map[string]interface{})["succeed"].(bool)
}

func TestSuccessfulTest(t *testing.T) {
	code, response := testRequest(validOutput)
	assert.Equal(t, http.StatusOK, code)
	succeed := getTestStatus(response)
	assert.Equal(t, true, succeed)
}

func TestFailedTest(t *testing.T) {
	for _, output := range invalidOutputs {
		code, response := testRequest(output)
		assert.Equal(t, http.StatusOK, code)
		succeed := getTestStatus(response)
		assert.Equal(t, false, succeed)
	}
}

package unittest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"service/internal/engine"
	"service/internal/engine/javascript"
	"service/internal/engine/starlark"
	"service/internal/model"
	"service/internal/router/resp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type TestResult struct {
	Data    TestResultData `json:"data"`
	Message string         `json:"message"`
}

type TestResultData struct {
	Duration int64 `json:"duration"`
	Succeed  bool  `json:"succeed"`
}

func ExecuteTest(c *gin.Context) {
	// verify the identity of the request initiator
	secret := c.GetHeader("Secret")

	if secret == "" {
		resp.Error(c, http.StatusBadRequest, "missing secret")
		return
	}

	// verify the secret
	localSecret := viper.GetString("test-secret")

	if localSecret == "" {
		resp.Error(c, http.StatusInternalServerError,
			"secret is not set properly in the server")
		return
	}

	if secret != localSecret {
		resp.Error(c, http.StatusForbidden, "wrong access secret")
		return
	}

	testId := c.Param("test_id")

	var testCase model.TestCase
	if err := model.DB.First(&testCase, "test_id = ?", testId).Error; err != nil {
		resp.Error(c, http.StatusBadRequest, "test record does not exist")
		return
	}

	var testCode model.Code
	if err := model.DB.First(&testCode, "code_id = ?", testCase.CodeID).Error; err != nil {
		log.Printf("cannot find test case %s for code %s: %v",
			testCase.TestID, testCase.CodeID, err)
		resp.Error(c, http.StatusInternalServerError, "cannot find test case: "+err.Error())
		return
	}

	// validate input
	var inputMap map[string]interface{}
	if err := json.Unmarshal([]byte(testCase.Input), &inputMap); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid JSON input: "+err.Error())
		return
	}

	inputMap, err := testCode.ValidateParams(inputMap)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid input params: "+err.Error())
		return
	}

	inputData, err := json.Marshal(inputMap)

	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "internal error: "+err.Error())
		return
	}

	var res engine.RunResult
	startTime := time.Now()

	if testCode.Lang == "starlark" {
		res = starlark.Run("", testCode.Content, string(inputData))
	} else if testCode.Lang == "javascript" {
		res = javascript.Run("", testCode.Content, string(inputData))
	} else {
		msg := "invalid lang " + testCode.Lang
		log.Print(msg)
		resp.Error(c, http.StatusInternalServerError, "internal error: "+msg)
		return
	}

	if res.Err != nil {
		resp.Error(c, http.StatusBadRequest, res.Err.Error())
		return
	}

	duration := int64(time.Since(startTime) / time.Microsecond)

	var testResult map[string]interface{}
	if err := json.Unmarshal([]byte(res.Val), &testResult); err != nil {
		resp.Error(c, http.StatusBadRequest,
			"fail to parse JSON from actural output:"+err.Error())
		return
	}

	var expectedOutput map[string]interface{}
	if err := json.Unmarshal([]byte(testCase.Output), &expectedOutput); err != nil {
		resp.Error(c, http.StatusBadRequest,
			"fail to parse JSON from expected output:"+err.Error())
		return
	}

	message := "success"
	matched := reflect.DeepEqual(testResult, expectedOutput)

	if !matched {
		message = fmt.Sprintf("wrong output:\n%s", res.Val)
	}

	resp.Ok(c, http.StatusOK, TestResult{
		Data: TestResultData{
			Duration: duration,
			Succeed:  matched,
		},
		Message: message,
	})
}

package playground

import (
	"encoding/json"
	"net/http"
	"service/internal/engine"
	"service/internal/engine/javascript"
	"service/internal/engine/starlark"
	"service/internal/router/resp"

	"github.com/gin-gonic/gin"
)

type PlaygroundRequestBody struct {
	Code   string                 `json:"code"`
	Params map[string]interface{} `json:"params"`
}

func run(c *gin.Context, runner func(string, string, string) engine.RunResult) {
	var body PlaygroundRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	data, err := json.Marshal(body.Params)

	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid params: "+err.Error())
		return
	}

	res := runner("", body.Code, string(data))

	if res.Err != nil {
		resp.Error(c, http.StatusBadRequest, res.Err.Error())
		return
	}

	resp.Ok(c, http.StatusOK, res.Val)
}

func Starlark(c *gin.Context) {
	run(c, starlark.Run)
}

func JavaScript(c *gin.Context) {
	run(c, javascript.Run)
}

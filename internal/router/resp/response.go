package resp

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Data interface{} `json:"data"`
	Msg  string      `json:"message"`
}

func Ok(c *gin.Context, code int, data interface{}) {
	c.JSON(code, Response{
		Data: data,
		Msg:  "success",
	})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, Response{
		Msg: msg,
	})
}

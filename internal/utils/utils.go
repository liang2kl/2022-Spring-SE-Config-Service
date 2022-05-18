package utils

import (
	"bytes"
	"hash/fnv"

	"github.com/gin-gonic/gin"
)

func GetBody(c *gin.Context) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(c.Request.Body)
	return buf.String()
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func Find[T comparable](slice []T, value T) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

func Remove[T comparable](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

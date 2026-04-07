package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RespSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  data,
	})
}

func RespError(c *gin.Context, code int, info any) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  info,
	})
}

package response

import (
	"errors"
	"net/http"

	"github.com/herj1025/kumquat/pkg/e"
	"github.com/herj1025/kumquat/pkg/i18n"

	"github.com/gin-gonic/gin"
)

// Response 标准响应结构
type Response[T any] struct {
	Code    int    `json:"code"`    // 业务码，0 表示成功，非 0 表示错误
	Message string `json:"message"` // 提示信息
	Data    T      `json:"data"`    // 数据载荷
}

// Success 输出成功响应
func GinSuccess[T any](c *gin.Context, data T) {
	lang := i18n.GinGetLang(c)
	c.JSON(http.StatusOK, Response[T]{
		Code:    0,
		Message: i18n.Translate(lang, i18n.KeySuccess),
		Data:    data,
	})
}

// Error 输出错误响应
// 支持传递 e.Error 接口，或者普通的 error (自动包装为 500)
func GinError(c *gin.Context, err error) {
	lang := i18n.GinGetLang(c)
	var (
		code = 50001
		msg  = i18n.Translate(lang, i18n.KeyInternalServerError)
	)

	// 类型断言检查是否为业务错误
	var serverE e.Error
	if errors.As(err, &serverE) {
		code = serverE.Code()
		msg = serverE.Msg()
	}

	c.JSON(http.StatusInternalServerError, Response[any]{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// Success 输出成功响应
func GinCustomize[T any](c *gin.Context, code int, message string, data T) {
	c.JSON(http.StatusOK, Response[T]{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

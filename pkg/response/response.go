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

// GinError 输出错误响应。
// 支持传递 e.Error 接口（自动映射 HTTP 状态码），或者普通的 error（默认 500）。
func GinError(c *gin.Context, err error) {
	lang := i18n.GinGetLang(c)
	var (
		code       = 50001
		msg        = i18n.Translate(lang, i18n.KeyInternalServerError)
		httpStatus = http.StatusInternalServerError
	)

	var serverE e.Error
	if errors.As(err, &serverE) {
		code = serverE.Code()
		msg = serverE.Msg()
		httpStatus = httpStatusFromCode(code)
	}

	c.JSON(httpStatus, Response[any]{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// httpStatusFromCode 根据业务错误码范围映射 HTTP 状态码
func httpStatusFromCode(code int) int {
	switch {
	case code == 0:
		return http.StatusOK
	case code >= 4000 && code < 4010:
		return http.StatusBadRequest
	case code >= 4010 && code < 4020:
		return http.StatusUnauthorized
	case code >= 4030 && code < 4040:
		return http.StatusForbidden
	case code >= 4040 && code < 4050:
		return http.StatusNotFound
	case code >= 4090 && code < 4100:
		return http.StatusConflict
	case code >= 4290 && code < 4300:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// GinCustomize 输出自定义格式的响应
func GinCustomize[T any](c *gin.Context, code int, message string, data T) {
	c.JSON(http.StatusOK, Response[T]{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

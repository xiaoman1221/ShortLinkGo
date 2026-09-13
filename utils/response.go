// Package utils 提供通用工具：统一响应、密码哈希、JWT 等。
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一 JSON 响应结构：code 为 0 表示成功；失败时 code 与 HTTP 状态码一致。
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// OK 成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "succ", Data: data})
}

// OKMsg 成功响应（自定义提示语）。
func OKMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: msg, Data: data})
}

// Fail 失败响应。
func Fail(c *gin.Context, status int, msg string) {
	c.JSON(status, Response{Code: status, Msg: msg})
}

// BadRequest 参数错误。
func BadRequest(c *gin.Context, msg string) { Fail(c, http.StatusBadRequest, msg) }

// Unauthorized 未认证。
func Unauthorized(c *gin.Context, msg string) { Fail(c, http.StatusUnauthorized, msg) }

// Forbidden 无权限。
func Forbidden(c *gin.Context, msg string) { Fail(c, http.StatusForbidden, msg) }

// NotFound 资源不存在。
func NotFound(c *gin.Context, msg string) { Fail(c, http.StatusNotFound, msg) }

// ServerError 服务内部错误。
func ServerError(c *gin.Context, msg string) { Fail(c, http.StatusInternalServerError, msg) }

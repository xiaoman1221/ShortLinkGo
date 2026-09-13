// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// TokenHandler API 令牌接口。
type TokenHandler struct {
	Svc *services.TokenService
}

// List GET /api/tokens
func (h *TokenHandler) List(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	list, err := h.Svc.List(uid)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, list)
}

// Create POST /api/tokens  body: {"name":"xxx"}
func (h *TokenHandler) Create(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	plain, t, err := h.Svc.Create(uid, req.Name)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "令牌已创建，请立即保存（只显示一次）", gin.H{"token": plain, "item": t})
}

// Delete DELETE /api/tokens/:id
func (h *TokenHandler) Delete(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的令牌 ID")
		return
	}
	if err := h.Svc.Delete(uid, uint(id)); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "令牌已删除", nil)
}

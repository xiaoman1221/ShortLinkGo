// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// UserAdminHandler 用户管理（管理员查看，超级管理员修改角色/封禁）。
type UserAdminHandler struct {
	Svc *services.AuthService
}

// List GET /api/admin/users?page=&page_size=&keyword=
func (h *UserAdminHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.Svc.ListUsers(page, pageSize, c.Query("keyword"))
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// SetRole PUT /api/admin/users/:id/role  body: {"role":"admin"}
func (h *UserAdminHandler) SetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户 ID")
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.Svc.SetUserRole(uint(id), req.Role); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "角色已更新", nil)
}

// SetStatus PUT /api/admin/users/:id/status  body: {"status":0}
func (h *UserAdminHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户 ID")
		return
	}
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.Svc.SetUserStatus(uint(id), req.Status); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	msg := "已解封"
	if req.Status == services.StatusBanned {
		msg = "已封禁该用户"
	}
	utils.OKMsg(c, msg, nil)
}

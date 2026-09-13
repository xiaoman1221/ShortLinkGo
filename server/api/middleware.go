// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// Auth 统一鉴权：接受 JWT（Authorization: Bearer <jwt>）或 API Token（Bearer <token>）。
// 通过后在上下文中写入 userID/username/role。
func Auth(jwtKey string, tokens *services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		bearer := strings.TrimPrefix(h, "Bearer ")
		if bearer == "" || bearer == h {
			utils.Unauthorized(c, "未登录或缺少令牌")
			c.Abort()
			return
		}
		// 先尝试 JWT；角色/封禁状态以数据库实时为准
		if claims, err := utils.ParseToken(bearer, jwtKey); err == nil {
			u, err := tokens.UserByID(claims.UserID)
			if err != nil {
				utils.Unauthorized(c, err.Error())
				c.Abort()
				return
			}
			if u == nil {
				utils.Unauthorized(c, "账号不存在")
				c.Abort()
				return
			}
			c.Set("userID", u.ID)
			c.Set("username", u.Username)
			c.Set("role", u.Role)
			c.Next()
			return
		}
		// 再尝试 API Token
		u, err := tokens.AuthByToken(bearer)
		if err != nil {
			utils.Unauthorized(c, err.Error())
			c.Abort()
			return
		}
		if u == nil {
			utils.Unauthorized(c, "登录已过期或令牌无效")
			c.Abort()
			return
		}
		c.Set("userID", u.ID)
		c.Set("username", u.Username)
		c.Set("role", u.Role)
		c.Next()
	}
}

// AdminOnly 要求管理员或超级管理员。
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		s, _ := role.(string)
		if !services.IsStaff(s) {
			utils.Forbidden(c, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SuperOnly 要求超级管理员（UID=1）。
func SuperOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		s, _ := role.(string)
		if s != services.RoleSuper {
			utils.Forbidden(c, "需要超级管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

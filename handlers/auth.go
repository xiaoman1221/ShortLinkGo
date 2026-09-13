// Package handlers 提供 HTTP 请求处理器。
package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/services"
	"ShortLinkGo/utils"
)

// AuthHandler 认证相关接口。
type AuthHandler struct {
	Svc      *services.AuthService
	Settings *services.SettingService
	Debug    bool   // 开发模式：SMTP 未配置时返回重置链接便于本地联调
	BaseHost string // 站点对外地址（config.Host），空时用请求 Host
}

// Register POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Email    string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	u, err := h.Svc.Register(req.Username, req.Password, req.Nickname, req.Email)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "注册成功，请登录", u)
}

// Login POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	token, user, err := h.Svc.Login(req.Username, req.Password)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "登录成功", gin.H{"token": token, "user": user})
}

// Profile GET /api/auth/profile（需登录）
func (h *AuthHandler) Profile(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	u, err := h.Svc.GetByID(uid)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, u)
}

// UpdateProfile PUT /api/auth/profile（需登录）
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	u, err := h.Svc.UpdateProfile(uid, req.Nickname, req.Email, req.Phone, req.Avatar)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "资料已更新", u)
}

// ChangePassword PUT /api/auth/password（需登录）
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.Svc.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "密码已修改，请重新登录", nil)
}

// Forgot POST /api/auth/forgot
func (h *AuthHandler) Forgot(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	u, err := h.Svc.FindByEmail(req.Email)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	if u == nil {
		// 不泄露邮箱是否存在
		utils.OKMsg(c, "如果该邮箱已注册，重置邮件已发送", nil)
		return
	}
	token, err := h.Svc.CreateResetToken(u.ID)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	resetURL := h.resetURL(c, token)
	smtpCfg, err := h.Settings.SMTP()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	if !smtpCfg.Enabled() {
		if h.Debug {
			utils.OKMsg(c, "开发模式：未配置 SMTP，重置链接见下方（仅开发环境）", gin.H{"reset_url": resetURL})
			return
		}
		utils.BadRequest(c, "邮件服务未配置，请联系管理员重置密码")
		return
	}
	body := "你好 " + u.Username + "：\n\n" +
		"请点击以下链接重置你的 ShortLinkGo 密码（30 分钟内有效）：\n\n" +
		resetURL + "\n\n如果不是你本人操作，请忽略本邮件。"
	if err := utils.SendMail(smtpCfg, u.Email, "ShortLinkGo 重置密码", body); err != nil {
		utils.ServerError(c, "邮件发送失败: "+err.Error())
		return
	}
	utils.OKMsg(c, "重置邮件已发送，请查收", nil)
}

// Reset POST /api/auth/reset
func (h *AuthHandler) Reset(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.Svc.ResetPassword(req.Token, req.Password); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "密码已重置，请使用新密码登录", nil)
}

// Avatar POST /api/auth/avatar（multipart file 字段）
func (h *AuthHandler) Avatar(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	url, err := saveImage(c, "avatars")
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	u, err := h.Svc.SetAvatar(uid, url)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OKMsg(c, "头像已更新", u)
}

// QQAuthorize GET /api/auth/qq —— 跳转 QQ 授权页。
func (h *AuthHandler) QQAuthorize(c *gin.Context) {
	cfg, err := h.Settings.QQ()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	if !cfg.Enabled() {
		utils.BadRequest(c, "QQ 登录未配置，请联系管理员在系统管理中配置")
		return
	}
	qq := services.NewQQOAuth()
	uri := h.callbackBase(c) + "/api/auth/qq/callback"
	c.Redirect(http.StatusFound, qq.AuthorizeURL(cfg, uri, services.NewQQState()))
}

// QQCallback GET /api/auth/qq/callback —— QQ 授权回调。
func (h *AuthHandler) QQCallback(c *gin.Context) {
	base := h.frontendBase(c)
	fail := func(msg string) {
		c.Redirect(http.StatusFound, base+"/#/login?oauth_error="+url.QueryEscape(msg))
	}
	state := c.Query("state")
	code := c.Query("code")
	if !services.ValidQQState(state) {
		fail("QQ 授权状态校验失败，请重新登录")
		return
	}
	if code == "" {
		fail("QQ 未返回授权码，请重试")
		return
	}
	cfg, err := h.Settings.QQ()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	if !cfg.Enabled() {
		fail("QQ 登录未配置")
		return
	}
	qq := services.NewQQOAuth()
	uri := h.callbackBase(c) + "/api/auth/qq/callback"

	accessToken, err := qq.Exchange(cfg, code, uri)
	if err != nil {
		fail(err.Error())
		return
	}
	openid, err := qq.OpenID(accessToken)
	if err != nil {
		fail(err.Error())
		return
	}
	info, err := qq.UserInfo(cfg, accessToken, openid)
	if err != nil {
		fail(err.Error())
		return
	}
	u, err := h.Svc.FindOrCreateByQQ(info.OpenID, info.Nickname, info.Avatar)
	if err != nil {
		fail(err.Error())
		return
	}
	token, err := utils.GenerateToken(u.ID, u.Username, u.Role, h.Svc.JWTKey, h.Svc.TokenTTL)
	if err != nil {
		fail("签发登录凭证失败")
		return
	}
	c.Redirect(http.StatusFound, base+"/#/oauth?token="+url.QueryEscape(token))
}

// frontendBase 计算前端站点地址（不含路径）。
func (h *AuthHandler) frontendBase(c *gin.Context) string {
	base := strings.TrimRight(h.BaseHost, "/")
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + c.Request.Host
	}
	return base
}

// callbackBase 计算回调地址（OAuth redirect_uri 使用）。
func (h *AuthHandler) callbackBase(c *gin.Context) string {
	return h.frontendBase(c)
}

func (h *AuthHandler) resetURL(c *gin.Context, token string) string {
	base := strings.TrimRight(h.BaseHost, "/")
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + c.Request.Host
	}
	return base + "/#/reset?token=" + token
}

// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ShortLinkGo/server/app"
	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// New 组装 Gin 引擎与全部路由。
func New(cfg *app.Config, db *gorm.DB) *gin.Engine {
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	r := gin.Default()

	// 真实访客 IP 由 api.resolveClientIP 按网页配置的模式探测转发头获得，
	// 这里禁用 Gin 自身的代理信任逻辑，避免两套行为（误用 c.ClientIP 时始终取直连地址）。
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("[router] 关闭内置代理信任失败: %v", err)
	}

	authSvc := services.NewAuthService(db, cfg.JWTKey, time.Duration(cfg.JWTExpire)*time.Second)
	linkSvc := services.NewLinkService(db)
	settingSvc := services.NewSettingService(db)
	tokenSvc := services.NewTokenService(db)

	authH := &AuthHandler{
		Svc:      authSvc,
		Settings: settingSvc,
		Debug:    cfg.GinMode == "debug",
	}
	linkH := &LinkHandler{Svc: linkSvc, Settings: settingSvc}
	settingH := &SettingsHandler{Svc: settingSvc}
	tokenH := &TokenHandler{Svc: tokenSvc}
	userAdminH := &UserAdminHandler{Svc: authSvc}

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "msg": "ok",
			"data": gin.H{"time": time.Now().Format(time.RFC3339)},
		})
	})

	// 站点公开信息（名称/Logo/简介）
	r.GET("/api/site", func(c *gin.Context) {
		utils.OK(c, settingSvc.Site())
	})

	// 认证（公开：注册/登录/忘记密码/重置密码）
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/forgot", authH.Forgot)
		auth.POST("/reset", authH.Reset)
		auth.GET("/qq", authH.QQAuthorize)
		auth.GET("/qq/callback", authH.QQCallback)

		authed := auth.Group("", Auth(cfg.JWTKey, tokenSvc))
		authed.GET("/profile", authH.Profile)
		authed.PUT("/profile", authH.UpdateProfile)
		authed.PUT("/password", authH.ChangePassword)
		authed.POST("/avatar", authH.Avatar)
	}

	// 需登录：链接 / 统计 / 令牌
	apiGroup := r.Group("/api", Auth(cfg.JWTKey, tokenSvc))
	{
		links := apiGroup.Group("/links")
		{
			links.GET("", linkH.List)
			links.POST("", linkH.Create)
			links.GET("/:id", linkH.Get)
			links.PUT("/:id", linkH.Update)
			links.POST("/:id/review", linkH.Review)
			links.DELETE("/:id", linkH.Delete)
		}
		stats := apiGroup.Group("/stats")
		{
			stats.GET("/summary", linkH.Summary)
			stats.GET("/trend", linkH.Trend)
			stats.GET("/top", linkH.Top)
			stats.GET("/geo", linkH.Geo)
		}
		tokens := apiGroup.Group("/tokens")
		{
			tokens.GET("", tokenH.List)
			tokens.POST("", tokenH.Create)
			tokens.DELETE("/:id", tokenH.Delete)
		}
	}

	// 管理：用户管理（管理员可见，超级管理员可改）
	admin := r.Group("/api/admin", Auth(cfg.JWTKey, tokenSvc), AdminOnly())
	{
		admin.GET("/users", userAdminH.List)
		super := admin.Group("", SuperOnly())
		super.PUT("/users/:id/role", userAdminH.SetRole)
		super.PUT("/users/:id/status", userAdminH.SetStatus)
	}

	// 系统设置（管理员/超级管理员）
	settings := r.Group("/api/settings", Auth(cfg.JWTKey, tokenSvc), AdminOnly())
	{
		settings.GET("", settingH.List)
		settings.PUT("", settingH.Update)
		settings.POST("/smtp/test", settingH.TestSMTP)
		settings.POST("/logo", settingH.UploadLogo)
		settings.GET("/site-url/detect", settingH.DetectSiteURL)
		settings.GET("/geoip/status", settingH.GeoIPStatus)
		settings.POST("/geoip/update", settingH.GeoIPUpdate)
		settings.POST("/geoip/backfill", settingH.GeoIPBackfill)
	}

	// 上传文件（头像/Logo，存储于 data/uploads，容器内为数据卷 /app/data/uploads）
	r.Static("/uploads", "./data/uploads")

	// 接口文档
	r.Static("/docs", "./docs")

	// 前端静态资源（web/dist，若存在）
	if _, err := os.Stat("./web/dist"); err == nil {
		r.Static("/assets", "./web/dist/assets")
		r.GET("/", serveIndex(http.Dir("./web/dist")))
	} else {
		r.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "ShortLinkGo 服务运行中。请先构建前端，或访问 /docs 查看接口文档。")
		})
	}

	// 未匹配路径：GET 单段短码走短链接跳转；其余返回 404。
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/docs") ||
			strings.HasPrefix(path, "/assets") || strings.HasPrefix(path, "/uploads") {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		code := strings.TrimPrefix(path, "/")
		if code == "" || strings.Contains(code, "/") {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		linkH.HandleRedirect(c, code)
	})

	return r
}

// serveIndex 返回一个处理函数，用于输出 web/dist/index.html（每次请求重新打开文件）。
func serveIndex(fs http.FileSystem) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := fs.Open("index.html")
		if err != nil {
			c.String(http.StatusNotFound, "index.html not found")
			return
		}
		defer file.Close()
		if stat, err := file.Stat(); err == nil {
			http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), file)
			return
		}
		c.String(http.StatusInternalServerError, "serve index failed")
	}
}

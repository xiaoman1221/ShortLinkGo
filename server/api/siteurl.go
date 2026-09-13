// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// resolveSiteBase 计算站点对外基地址（不含尾部斜杠）：
// 优先使用网页配置的 site_url（存数据库），未配置时回退到当前请求的
// X-Forwarded-Proto/Host（反代场景）或 TLS + Host。
func resolveSiteBase(c *gin.Context, svc *services.SettingService) string {
	if u := svc.SiteURL(); u != "" {
		return u
	}
	proto := c.GetHeader("X-Forwarded-Proto")
	if proto != "https" && proto != "http" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return proto + "://" + host
}

// DetectSiteURL GET /api/settings/site-url/detect —— 探测当前请求的站点地址。
func (h *SettingsHandler) DetectSiteURL(c *gin.Context) {
	utils.OK(c, gin.H{
		"detected": resolveSiteBase(c, h.Svc),
		"saved":    h.Svc.SiteURL(),
	})
}

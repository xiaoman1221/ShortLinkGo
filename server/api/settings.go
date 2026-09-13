// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/services"
	"ShortLinkGo/server/utils"
)

// SettingsHandler 系统设置接口（仅管理员）。
type SettingsHandler struct {
	Svc *services.SettingService
}

// List GET /api/settings
func (h *SettingsHandler) List(c *gin.Context) {
	all, err := h.Svc.All()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	_, passSet := all["smtp_pass"]
	all["smtp_pass"] = "" // 不回传密码明文
	all["smtp_pass_set"] = ""
	if passSet {
		all["smtp_pass_set"] = "1"
	}
	_, qqSet := all["qq_app_key"]
	all["qq_app_key"] = "" // 不回传 QQ AppKey 明文
	all["qq_app_key_set"] = ""
	if qqSet {
		all["qq_app_key_set"] = "1"
	}
	_, geoKeySet := all["geoip_license_key"]
	all["geoip_license_key"] = "" // 不回传 License Key 明文
	all["geoip_license_key_set"] = ""
	if geoKeySet {
		all["geoip_license_key_set"] = "1"
	}
	utils.OK(c, all)
}

// Update PUT /api/settings
func (h *SettingsHandler) Update(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// smtp_pass / geoip_license_key 留空表示不修改
	for k, v := range body {
		if (k == "smtp_pass" || k == "geoip_license_key") && v == "" {
			continue
		}
		if err := h.Svc.Set(k, v); err != nil {
			utils.BadRequest(c, err.Error())
			return
		}
	}
	utils.OKMsg(c, "设置已保存", nil)
}

// GeoIPStatus GET /api/settings/geoip/status —— GeoIP 数据库下载器状态。
func (h *SettingsHandler) GeoIPStatus(c *gin.Context) {
	utils.OK(c, services.GetGeoIPStatus(h.Svc.DB))
}

// GeoIPBackfill POST /api/settings/geoip/backfill —— 立即触发一次未识别 IP 回填（异步执行）。
func (h *SettingsHandler) GeoIPBackfill(c *gin.Context) {
	if err := services.TriggerGeoIPBackfill(h.Svc.DB); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "已开始后台回填，稍后可刷新状态查看结果", nil)
}

// GeoIPUpdate POST /api/settings/geoip/update —— 立即触发一次检查/下载（异步执行）。
func (h *SettingsHandler) GeoIPUpdate(c *gin.Context) {
	if err := services.TriggerGeoIPUpdate(h.Svc.DB); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "已开始后台更新，稍后可刷新状态查看进度", nil)
}

// UploadLogo POST /api/settings/logo（multipart file 字段）
func (h *SettingsHandler) UploadLogo(c *gin.Context) {
	url, err := saveImage(c, "site")
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if err := h.Svc.Set("site_logo", url); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "Logo 已更新", gin.H{"site_logo": url})
}

// TestSMTP POST /api/settings/smtp/test  body: {"to":"xxx@example.com"}
func (h *SettingsHandler) TestSMTP(c *gin.Context) {
	var req struct {
		To string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	cfg, err := h.Svc.SMTP()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	if err := utils.SendMail(cfg, req.To, "ShortLinkGo SMTP 测试", "这是一封来自 ShortLinkGo 的测试邮件。如果收到本邮件，说明 SMTP 配置正确。"); err != nil {
		utils.BadRequest(c, "发送失败: "+err.Error())
		return
	}
	utils.OKMsg(c, "测试邮件已发送", nil)
}

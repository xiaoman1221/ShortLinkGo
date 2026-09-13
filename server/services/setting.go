package services

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// allowedSettingKeys 允许通过接口修改的系统设置键。
var allowedSettingKeys = map[string]bool{
	"smtp_host":                 true,
	"smtp_port":                 true,
	"smtp_user":                 true,
	"smtp_pass":                 true,
	"smtp_from":                 true,
	"site_name":                 true,
	"site_logo":                 true,
	"site_desc":                 true,
	"site_url":                  true,
	"qq_app_id":                 true,
	"qq_app_key":                true,
	"geoip_enabled":             true,
	"geoip_url":                 true,
	"geoip_license_key":         true,
	"client_ip_mode":            true,
	"client_ip_trusted_proxies": true,
	"geoip_api_enabled":         true,
	"geoip_api_provider":        true,
	"geoip_api_key":             true,
	"geoip_api_url":             true,
}

// GeoIPAPIProviders 公共 IP 查询提供商（网页可切换）。
var GeoIPAPIProviders = map[string]bool{
	"pconline": true, // 太平洋电脑网，国内老牌，中文，免费无 key（默认）
	"ip-api":   true, // 免费无 key，批量查询，免费版仅 http
	"ipwhois":  true, // https 免费无 key
	"ipinfo":   true, // https，无 key 有限额
	"custom":   true, // 自定义 URL 模板（{ip} 占位，ip-api 兼容响应格式）
}

// ClientIPModes 真实 IP 识别模式（网页可切换）。
var ClientIPModes = map[string]bool{
	"smart":  true, // 默认：直连为私网/回环（反代后）才探测转发头
	"always": true, // 始终探测转发头（Cloudflare 等回源 IP 为公网的场景）
	"direct": true, // 始终使用直连地址
	"cidr":   true, // 直连命中 client_ip_trusted_proxies 代理段才探测转发头
}

// defaultSiteName 默认站点名称。
const defaultSiteName = "ShortLinkGo"

// SettingService 系统设置服务。
type SettingService struct {
	DB *gorm.DB
}

// NewSettingService 构造 SettingService。
func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{DB: db}
}

// All 返回全部设置（用于管理端展示；密码类单独处理）。
func (s *SettingService) All() (map[string]string, error) {
	var rows []Setting
	if err := s.DB.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

// Set 写入单个设置。
func (s *SettingService) Set(key, value string) error {
	if !allowedSettingKeys[key] {
		return errors.New("不允许修改该设置项: " + key)
	}
	if key == "smtp_port" && value != "" {
		if n, err := strconv.Atoi(value); err != nil || n < 1 || n > 65535 {
			return errors.New("SMTP 端口无效")
		}
	}
	if key == "site_url" {
		value = strings.TrimRight(strings.TrimSpace(value), "/")
		if value != "" {
			u, err := url.Parse(value)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return errors.New("站点地址无效，应为 http(s)://域名 形式")
			}
			value = u.Scheme + "://" + u.Host
		}
	}
	if key == "geoip_api_provider" {
		if !GeoIPAPIProviders[value] {
			return errors.New("无效的提供商，可选 pconline/ip-api/ipwhois/ipinfo/custom")
		}
	}
	if key == "geoip_api_url" && strings.TrimSpace(value) != "" {
		if !strings.Contains(value, "{ip}") {
			return errors.New("自定义地址必须包含 {ip} 占位符")
		}
	}
	if key == "client_ip_mode" {
		if !ClientIPModes[value] {
			return errors.New("无效的 IP 识别模式，可选 smart/always/direct/cidr")
		}
	}
	if key == "client_ip_trusted_proxies" && strings.TrimSpace(value) != "" {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			cidr := item
			if !strings.Contains(cidr, "/") {
				cidr += "/32"
			}
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return errors.New("代理段格式无效: " + item + "（应为 CIDR 如 173.245.48.0/20 或单 IP）")
			}
		}
	}
	var count int64
	if err := s.DB.Model(&Setting{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return s.DB.Create(&Setting{Key: key, Value: strings.TrimSpace(value)}).Error
	}
	return s.DB.Model(&Setting{}).Where("key = ?", key).UpdateColumn("value", value).Error
}

// Site 返回对前端公开的站点信息（名称/Logo/简介）。
func (s *SettingService) Site() map[string]string {
	all, err := s.All()
	if err != nil {
		all = map[string]string{}
	}
	if all["site_name"] == "" {
		all["site_name"] = defaultSiteName
	}
	qqCfg := QQConfig{AppID: all["qq_app_id"], AppKey: all["qq_app_key"]}
	out := map[string]string{
		"site_name": all["site_name"],
		"site_logo": all["site_logo"],
		"site_desc": all["site_desc"],
	}
	if qqCfg.Enabled() {
		out["qq_enabled"] = "1"
	}
	return out
}

// QQ 读取 QQ 互联配置。
func (s *SettingService) QQ() (QQConfig, error) {
	all, err := s.All()
	if err != nil {
		return QQConfig{}, err
	}
	return QQConfig{AppID: all["qq_app_id"], AppKey: all["qq_app_key"]}, nil
}

// SMTP 读取 SMTP 配置。
func (s *SettingService) SMTP() (utils.SMTPConfig, error) {
	all, err := s.All()
	if err != nil {
		return utils.SMTPConfig{}, err
	}
	return utils.SMTPConfig{
		Host: all["smtp_host"],
		Port: all["smtp_port"],
		User: all["smtp_user"],
		Pass: all["smtp_pass"],
		From: all["smtp_from"],
	}, nil
}

// ClientIPSettings 读取真实 IP 识别配置：返回模式与可信代理段。
func (s *SettingService) ClientIPSettings() (string, string) {
	all, err := s.All()
	if err != nil {
		return "smart", ""
	}
	mode := all["client_ip_mode"]
	if !ClientIPModes[mode] {
		mode = "smart"
	}
	return mode, strings.TrimSpace(all["client_ip_trusted_proxies"])
}

// SiteURL 数据库中保存的站点对外地址（空 = 未配置，调用方回退到请求 Host）。
func (s *SettingService) SiteURL() string {
	all, err := s.All()
	if err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(all["site_url"]), "/")
}

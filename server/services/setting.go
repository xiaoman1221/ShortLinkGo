package services

import (
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// allowedSettingKeys 允许通过接口修改的系统设置键。
var allowedSettingKeys = map[string]bool{
	"smtp_host":  true,
	"smtp_port":  true,
	"smtp_user":  true,
	"smtp_pass":  true,
	"smtp_from":  true,
	"site_name":  true,
	"site_logo":  true,
	"site_desc":  true,
	"qq_app_id":  true,
	"qq_app_key": true,
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

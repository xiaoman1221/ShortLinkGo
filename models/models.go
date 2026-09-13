// Package models 定义数据模型。
package models

import "time"

// User 平台用户。
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Nickname     string    `gorm:"size:64" json:"nickname"`
	Avatar       string    `gorm:"size:255" json:"avatar"`
	Email        string    `gorm:"size:128;index" json:"email"`
	Phone        string    `gorm:"size:32" json:"phone"`
	QQOpenID     string    `gorm:"size:64" json:"qq_openid,omitempty"` // QQ 互联 openid；唯一性由 database 中的部分唯一索引保证（排除空值）
	Role         string    `gorm:"size:32;default:user" json:"role"`   // super/admin/user/vip
	Status       int       `gorm:"default:1" json:"status"`            // 1 正常 0 禁用
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Link 短链接。
type Link struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Code       string     `gorm:"size:32;uniqueIndex;not null" json:"code"` // 短码
	URL        string     `gorm:"size:2048;not null" json:"url"`            // 原始长链接
	Remark     string     `gorm:"size:255" json:"remark"`                   // 备注
	ExpireAt   *time.Time `json:"expire_at"`                                // 过期时间（空 = 永不过期）
	Status     int        `gorm:"default:1" json:"status"`                  // 1 启用 0 禁用
	VisitCount int64      `gorm:"default:0" json:"visit_count"`             // 访问次数
	UserID     uint       `gorm:"index;not null" json:"user_id"`            // 创建人
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// VisitLog 短链接访问日志。
type VisitLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	LinkID    uint      `gorm:"index;not null" json:"link_id"`
	IP        string    `gorm:"size:64;index" json:"ip"`
	Country   string    `gorm:"size:64" json:"country"`
	Region    string    `gorm:"size:64" json:"region"`
	City      string    `gorm:"size:64" json:"city"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	Referer   string    `gorm:"size:512" json:"referer"`
	CreatedAt time.Time `json:"created_at"`
}

// Setting 系统设置（键值对）。
type Setting struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value string `gorm:"size:2048" json:"value"`
}

// PasswordReset 找回密码令牌。
type PasswordReset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// ApiToken API 访问令牌（用于以令牌方式调用本人短链接接口）。
type ApiToken struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Name       string     `gorm:"size:64;not null" json:"name"`
	TokenHash  string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

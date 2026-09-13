// Package app 负责应用装配：配置加载与数据库初始化。
package app

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ShortLinkGo/server/services"
)

// Init 打开数据库并执行自动迁移、初始化默认数据。
func Init(path string) (*gorm.DB, error) {
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true, // 把驱动错误翻译为 gorm.ErrDuplicatedKey 等语义化错误
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&services.User{},
		&services.Link{},
		&services.VisitLog{},
		&services.Setting{},
		&services.PasswordReset{},
		&services.ApiToken{},
	); err != nil {
		return nil, err
	}
	if err := ensurePartialUniqueIndexes(db); err != nil {
		return nil, err
	}
	if err := ensureSuperAdmin(db); err != nil {
		return nil, err
	}
	if err := seedSettingsFromEnv(db); err != nil {
		return nil, err
	}
	return db, nil
}

// ensurePartialUniqueIndexes 为 email / qq_open_id 建立排除空值的部分唯一索引。
// GORM 标签无法声明部分索引；两者都存在大量空值（未绑定邮箱/QQ 的账号），
// 全列唯一索引会让多个空值互相冲突，导致第二个注册用户必然失败。
// 以下语句均为固定 DDL 字面量，不含任何用户输入。
func ensurePartialUniqueIndexes(db *gorm.DB) error {
	if err := db.Exec("DROP INDEX IF EXISTS idx_users_qq_open_id").Error; err != nil {
		log.Printf("[database] 清理旧索引失败: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email) WHERE email <> ''").Error; err != nil {
		// 旧库可能已存在重复数据导致建索引失败：不阻塞启动，但必须提示人工处理
		log.Printf("[database] 创建 users(email) 唯一索引失败（可能存在重复数据，请清理后重启）: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_qq_open_id_unique ON users(qq_open_id) WHERE qq_open_id <> ''").Error; err != nil {
		log.Printf("[database] 创建 users(qq_open_id) 唯一索引失败（可能存在重复数据，请清理后重启）: %v", err)
	}
	return nil
}

// ensureSuperAdmin 兼容旧库：若存在 ID=1 的用户则将其提升为超级管理员（super）。
// 新库不创建任何默认账号，第一个注册的用户自动成为超级管理员。
func ensureSuperAdmin(db *gorm.DB) error {
	var u services.User
	if err := db.First(&u, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if u.Role != "super" {
		return db.Model(&services.User{}).Where("id = ?", 1).UpdateColumn("role", "super").Error
	}
	return nil
}

// seedSettingsFromEnv 若环境变量提供了 SMTP 配置，则写入系统设置（便于 Docker/手动部署）。
func seedSettingsFromEnv(db *gorm.DB) error {
	env := map[string]string{
		"smtp_host": os.Getenv("SMTP_HOST"),
		"smtp_port": os.Getenv("SMTP_PORT"),
		"smtp_user": os.Getenv("SMTP_USER"),
		"smtp_pass": os.Getenv("SMTP_PASS"),
		"smtp_from": os.Getenv("SMTP_FROM"),
		"site_name": os.Getenv("SITE_NAME"),
		"site_logo": os.Getenv("SITE_LOGO"),
		"site_desc": os.Getenv("SITE_DESC"),
	}
	for k, v := range env {
		if v == "" {
			continue
		}
		var count int64
		if err := db.Model(&services.Setting{}).Where("key = ?", k).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Create(&services.Setting{Key: k, Value: v}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

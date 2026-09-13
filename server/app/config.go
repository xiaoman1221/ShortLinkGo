// Package app 负责应用装配：配置加载与数据库初始化。
package app

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
)

// Config 应用配置。
type Config struct {
	Port      string // 服务监听端口
	GinMode   string // Gin 运行模式 debug/release/test
	DBPath    string // SQLite 数据库文件路径
	JWTKey    string // JWT 签名密钥
	Host      string // 站点对外地址，用于拼装短链接，如 https://s.example.com
	JWTExpire int64  // JWT 有效期（秒）
}

// Load 从环境变量加载配置，未设置时使用默认值。
// 仅保留数据库/密钥等关键配置；其余运行时配置一律在网页「系统管理」完成（存数据库）。
func Load() *Config {
	cfg := &Config{
		Port:      getEnv("PORT", "8080"),
		GinMode:   getEnv("GIN_MODE", "release"),
		DBPath:    getEnv("DB_PATH", "data.db"),
		JWTKey:    os.Getenv("JWT_KEY"),
		Host:      getEnv("HOST", ""),
		JWTExpire: getEnvInt64("JWT_EXPIRE", 24*3600),
	}
	if cfg.JWTKey == "" {
		cfg.JWTKey = randomHex(32)
		log.Println("[config] JWT_KEY 未设置，已生成临时随机密钥（重启后失效）。生产环境请通过环境变量设置固定 JWT_KEY")
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
		log.Printf("[config] %s 不是有效的正整数，使用默认值 %d", key, def)
	}
	return def
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

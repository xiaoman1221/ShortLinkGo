package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// RandomHex 生成 n 字节的随机十六进制字符串（2n 个字符）。
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// HashToken 对重置令牌做 SHA-256，返回十六进制摘要，用于落库存储。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

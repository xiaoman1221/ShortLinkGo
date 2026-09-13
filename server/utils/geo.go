// Package utils 提供通用工具：统一响应、密码哈希、JWT、邮件发送、IP 地理位置解析。
// 地理解析数据源：可选的 MaxMind GeoLite2-City.mmdb（可通过 GEO_DB_PATH 指定，
// 或放在 ./data/GeoLite2-City.mmdb）。未配置数据文件时，仅区分内网/公网。
package utils

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoResult 地理位置结果。
type GeoResult struct {
	Country string  `json:"country"`
	Region  string  `json:"region"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

var (
	geoOnce sync.Once
	geoDB   *geoip2.Reader
)

func geoDBPath() string {
	if v := os.Getenv("GEO_DB_PATH"); v != "" {
		return v
	}
	for _, c := range []string{"./data/GeoLite2-City.mmdb", "./GeoLite2-City.mmdb"} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func openGeoDB() {
	path := geoDBPath()
	if path == "" {
		return
	}
	r, err := geoip2.Open(path)
	if err != nil {
		// 数据文件损坏/版本不兼容时不阻塞服务，但要留下线索
		log.Printf("[geo] 打开 GeoIP 数据库失败 %s: %v", path, err)
		return
	}
	geoDB = r
}

func isPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// Lookup 解析 IP 对应的地理位置。geoip2.Reader 并发安全，无需额外加锁。
func Lookup(ipStr string) GeoResult {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return GeoResult{Country: "未知"}
	}
	if isPrivate(ip) {
		return GeoResult{Country: "内网"}
	}
	geoOnce.Do(openGeoDB)
	if geoDB == nil {
		return GeoResult{Country: "未知"}
	}
	rec, err := geoDB.City(ip)
	if err != nil {
		return GeoResult{Country: "未知"}
	}
	r := GeoResult{
		Country: firstNonEmpty(rec.Country.Names["zh-CN"], rec.Country.Names["en"], rec.Country.IsoCode),
		Region:  firstNonEmpty(subName(rec, 0)),
		City:    firstNonEmpty(subName(rec, 1), rec.City.Names["zh-CN"], rec.City.Names["en"]),
		Lat:     rec.Location.Latitude,
		Lon:     rec.Location.Longitude,
	}
	return r
}

func subName(rec *geoip2.City, i int) string {
	if i < len(rec.Subdivisions) {
		if v := rec.Subdivisions[i].Names["zh-CN"]; v != "" {
			return v
		}
		if v := rec.Subdivisions[i].Names["en"]; v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

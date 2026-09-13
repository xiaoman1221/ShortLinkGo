// Package geo 提供基于 IP 的访问地理位置解析。
// 数据源：可选的 MaxMind GeoLite2-City.mmdb（可通过 GEO_DB_PATH 指定，
// 或放在 ./data/GeoLite2-City.mmdb）。未配置数据文件时，仅区分内网/公网。
package geo

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// Result 地理位置结果。
type Result struct {
	Country string  `json:"country"`
	Region  string  `json:"region"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

var (
	once sync.Once
	db   *geoip2.Reader
)

func dbPath() string {
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

func openDB() {
	path := dbPath()
	if path == "" {
		return
	}
	r, err := geoip2.Open(path)
	if err != nil {
		// 数据文件损坏/版本不兼容时不阻塞服务，但要留下线索
		log.Printf("[geo] 打开 GeoIP 数据库失败 %s: %v", path, err)
		return
	}
	db = r
}

func isPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// Lookup 解析 IP 对应的地理位置。geoip2.Reader 并发安全，无需额外加锁。
func Lookup(ipStr string) Result {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return Result{Country: "未知"}
	}
	if isPrivate(ip) {
		return Result{Country: "内网"}
	}
	once.Do(openDB)
	if db == nil {
		return Result{Country: "未知"}
	}
	rec, err := db.City(ip)
	if err != nil {
		return Result{Country: "未知"}
	}
	r := Result{
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

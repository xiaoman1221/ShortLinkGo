// Package utils 提供通用工具：统一响应、密码哈希、JWT、邮件发送、IP 地理位置解析。
// 地理解析数据源：MaxMind GeoLite2-City.mmdb，固定存放于 GeoDBPath，
// 由 services 包的 GeoIP 更新器负责下载与每小时更新，全部配置在网页「系统管理」完成。
package utils

import (
	"net"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoDBPath GeoIP 数据库文件路径（相对工作目录；Docker 中位于 /app/data 数据卷）。
const GeoDBPath = "data/GeoLite2-City.mmdb"

// GeoResult 地理位置结果。
type GeoResult struct {
	Country string  `json:"country"`
	Region  string  `json:"region"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

var (
	geoMu sync.RWMutex
	geoDB *geoip2.Reader
)

// OpenGeoDB 打开（或热重载）指定路径的 mmdb 数据库。
// 成功后旧 Reader 被关闭替换，进行中的解析请求不受影响。
func OpenGeoDB(path string) error {
	r, err := geoip2.Open(path)
	if err != nil {
		return err
	}
	geoMu.Lock()
	old := geoDB
	geoDB = r
	geoMu.Unlock()
	if old != nil {
		old.Close()
	}
	return nil
}

// GeoDBLoaded mmdb 是否已加载可用。
func GeoDBLoaded() bool {
	geoMu.RLock()
	defer geoMu.RUnlock()
	return geoDB != nil
}

func isPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// PublicIP 是否公网地址（排除环回/私有/链路本地/组播/未指定/CGNAT）。
func PublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] < 128 {
		return false // 100.64.0.0/10 CGNAT 保留段
	}
	return true
}

// Lookup 分层解析 IP 地理位置：
//  1. 中国 IP 走中国库（ip2region，城市/运营商级精度）
//  2. 其他 IP 走世界库（GeoLite2-City）
//  3. 世界库不可用或未命中时，回退中国库的国外结果，最后返回「未知」
//
// geoip2.Reader 并发安全，读指针加锁避免热重载竞态。
func Lookup(ipStr string) GeoResult {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return GeoResult{Country: "未知"}
	}
	if isPrivate(ip) {
		return GeoResult{Country: "内网"}
	}

	// 中国库：仅采用「中国」结果，国外交由世界库
	cnCountry, cnRegion, cnCity, cnOK := SearchIP2Region(ipStr)
	if cnOK {
		return GeoResult{Country: cnCountry, Region: cnRegion, City: cnCity}
	}

	geoMu.RLock()
	db := geoDB
	geoMu.RUnlock()
	if db != nil {
		if rec, err := db.City(ip); err == nil {
			r := GeoResult{
				Country: firstNonEmpty(rec.Country.Names["zh-CN"], rec.Country.Names["en"], rec.Country.IsoCode),
				Region:  firstNonEmpty(subName(rec, 0)),
				City:    firstNonEmpty(subName(rec, 1), rec.City.Names["zh-CN"], rec.City.Names["en"]),
				Lat:     rec.Location.Latitude,
				Lon:     rec.Location.Longitude,
			}
			if r.Country != "" {
				return r
			}
		}
	}

	// 世界库不可用时回退中国库的国外结果（粗粒度但优于「未知」）
	if cnOK {
		return GeoResult{Country: cnCountry, Region: cnRegion, City: cnCity}
	}
	return GeoResult{Country: "未知"}
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

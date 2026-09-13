// Package utils 提供通用工具：统一响应、密码哈希、JWT、邮件发送、IP 地理位置解析。
package utils

import (
	"os"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// IP2RegionPath 中国 IP 库（ip2region xdb）文件路径，中国城市/运营商级精度；
// 与世界库 GeoLite2-City 分开：Lookup 先查中国库，非中国 IP 走世界库。
const IP2RegionPath = "data/ip2region.xdb"

var (
	ip2rMu       sync.RWMutex
	ip2rSearcher *xdb.Searcher
)

// OpenIP2Region 打开（或热重载）指定路径的 xdb 中国库，全量载入内存（约 11MB）。
func OpenIP2Region(path string) error {
	cBuff, err := xdb.LoadContentFromFile(path)
	if err != nil {
		return err
	}
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	defer handle.Close()
	header, err := xdb.LoadHeader(handle)
	if err != nil {
		return err
	}
	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		return err
	}
	s, err := xdb.NewWithBuffer(version, cBuff)
	if err != nil {
		return err
	}
	ip2rMu.Lock()
	ip2rSearcher = s
	ip2rMu.Unlock()
	return nil
}

// IP2RegionLoaded 中国库是否已加载可用。
func IP2RegionLoaded() bool {
	ip2rMu.RLock()
	defer ip2rMu.RUnlock()
	return ip2rSearcher != nil
}

// SearchIP2Region 查询中国库，返回 country/region/city。
// xdb 内容格式为「国家|区域|省份|城市|ISP」；ok=false 表示未命中或非中国 IP。
func SearchIP2Region(ipStr string) (country, region, city string, ok bool) {
	ip2rMu.RLock()
	s := ip2rSearcher
	ip2rMu.RUnlock()
	if s == nil {
		return "", "", "", false
	}
	raw, err := s.Search(strings.TrimSpace(ipStr))
	if err != nil {
		return "", "", "", false
	}
	parts := strings.Split(raw, "|")
	country = cleanRegionPart(parts, 0)
	region = cleanRegionPart(parts, 2)
	city = cleanRegionPart(parts, 3)
	if country == "" || country == "0" {
		return "", "", "", false
	}
	// 仅中国库命中「中国」时作为主结果；国外粗数据交由世界库处理
	if country != "中国" {
		return country, region, city, false
	}
	if region == "" {
		region = cleanRegionPart(parts, 1)
	}
	return country, region, city, true
}

func cleanRegionPart(parts []string, i int) string {
	if i >= len(parts) {
		return ""
	}
	v := strings.TrimSpace(parts[i])
	if v == "0" {
		return ""
	}
	return v
}

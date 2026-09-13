// Package services 封装业务逻辑与数据模型。
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/text/encoding/simplifiedchinese"

	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// 未识别 IP 的公共 API 兜底查询。
// 跳转只负责记录（解析不出的 IP 标记「未知」）；后台任务定期扫描未识别的访问
// 日志，去重后调用公共 API 批量查询，结果写入 ip_geo_cache 缓存（一次查询永久
// 复用）并回填访问日志的 country/region/city/lat/lon。
// 配置全部在网页「系统管理 → 访问地图 → 公共 API 兜底」完成（存数据库）：
//
//	geoip_api_enabled   "1" 启用
//	geoip_api_provider  ip-api / ipinfo / ipwhois / custom
//	geoip_api_key       部分提供商的密钥（可选）
//	geoip_api_url       custom 模式的地址模板，必须含 {ip} 占位符

const (
	backfillScanInterval = 10 * time.Minute
	backfillBatchSize    = 200                     // 每轮最多处理的未识别 IP 数
	backfillQueryDelay   = 1500 * time.Millisecond // 单条查询间隔（免费额度限速保护）
)

type geoIPBackfillState struct {
	mu          sync.Mutex
	running     bool
	lastRun     time.Time
	lastSuccess time.Time
	lastQueried int
	lastUpdated int
	lastError   string
}

var backfillState geoIPBackfillState

// GeoIPBackfillStatus 兜底回填状态（管理端展示）。
type GeoIPBackfillStatus struct {
	Enabled     bool      `json:"enabled"`
	Provider    string    `json:"provider"`
	LastRun     time.Time `json:"last_run"`
	LastSuccess time.Time `json:"last_success"`
	LastQueried int       `json:"last_queried"`
	LastUpdated int       `json:"last_updated"`
	LastError   string    `json:"last_error"`
	CacheCount  int64     `json:"cache_count"`
}

// geoAPIResult 统一的查询结果。
type geoAPIResult struct {
	Country string
	Region  string
	City    string
	Lat     float64
	Lon     float64
}

// StartGeoIPBackfill 启动兜底回填协程：1 分钟后首扫，此后每 10 分钟扫描一次。
func StartGeoIPBackfill(db *gorm.DB) {
	go func() {
		time.Sleep(time.Minute)
		runGeoIPBackfill(db)
		ticker := time.NewTicker(backfillScanInterval)
		defer ticker.Stop()
		for range ticker.C {
			runGeoIPBackfill(db)
		}
	}()
}

// TriggerGeoIPBackfill 管理端「立即回填」入口。
func TriggerGeoIPBackfill(db *gorm.DB) error {
	backfillState.mu.Lock()
	running := backfillState.running
	backfillState.mu.Unlock()
	if running {
		return errors.New("已有回填任务在进行中")
	}
	go runGeoIPBackfill(db)
	return nil
}

// GetGeoIPBackfillStatus 返回回填状态。
func GetGeoIPBackfillStatus(db *gorm.DB) GeoIPBackfillStatus {
	all := allSettings(db)
	backfillState.mu.Lock()
	st := GeoIPBackfillStatus{
		Enabled:     all["geoip_api_enabled"] == "1",
		Provider:    all["geoip_api_provider"],
		LastRun:     backfillState.lastRun,
		LastSuccess: backfillState.lastSuccess,
		LastQueried: backfillState.lastQueried,
		LastUpdated: backfillState.lastUpdated,
		LastError:   backfillState.lastError,
	}
	backfillState.mu.Unlock()
	if !GeoIPAPIProviders[st.Provider] {
		st.Provider = "ip-api"
	}
	db.Model(&IpGeoCache{}).Count(&st.CacheCount)
	return st
}

func runGeoIPBackfill(db *gorm.DB) {
	backfillState.mu.Lock()
	if backfillState.running {
		backfillState.mu.Unlock()
		return
	}
	backfillState.running = true
	backfillState.lastRun = time.Now()
	backfillState.mu.Unlock()

	defer func() {
		backfillState.mu.Lock()
		backfillState.running = false
		backfillState.mu.Unlock()
	}()

	all := allSettings(db)
	if all["geoip_api_enabled"] != "1" {
		return
	}
	provider := all["geoip_api_provider"]
	if !GeoIPAPIProviders[provider] {
		provider = "pconline"
	}
	key := strings.TrimSpace(all["geoip_api_key"])
	tpl := strings.TrimSpace(all["geoip_api_url"])

	// 未识别的访问日志 IP（去重，每轮限量）
	var ips []string
	if err := db.Model(&VisitLog{}).
		Where("country IN ?", []string{"未知", ""}).
		Distinct().Limit(backfillBatchSize).Pluck("ip", &ips).Error; err != nil {
		setBackfillError(db, err.Error())
		return
	}
	if len(ips) == 0 {
		clearBackfillError()
		return
	}

	var toQuery []string
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		parsed := net.ParseIP(ip)
		if parsed == nil {
			// 非法字符串（多为伪造头），标记后不再重复扫描
			markVisitLogs(db, ip, "无效", "", 0, 0)
			continue
		}
		if !utils.PublicIP(parsed) {
			markVisitLogs(db, ip, "内网", "", 0, 0)
			continue
		}
		// 缓存命中直接回填，不消耗配额
		var c IpGeoCache
		if err := db.First(&c, "ip = ?", ip).Error; err == nil {
			if c.Country != "" {
				applyVisitLogGeo(db, ip, c.Country, c.Region, c.City, c.Lat, c.Lon)
				continue
			}
			// country 为空的缓存 = 上次查询失败的记录，允许重试
		}
		toQuery = append(toQuery, ip)
	}

	queried, updated := 0, 0
	if provider == "ip-api" && len(toQuery) > 1 {
		// ip-api 批量接口：一次最多 100 个，消耗 1 次请求配额
		for start := 0; start < len(toQuery); start += 100 {
			end := min(start+100, len(toQuery))
			results, err := queryIPAPIBatch(toQuery[start:end])
			if err != nil {
				setBackfillError(db, "批量查询失败: "+err.Error())
				return
			}
			for _, ip := range toQuery[start:end] {
				if r, ok := results[ip]; ok {
					saveGeoIPResult(db, ip, r, provider)
					queried++
					updated++
				}
			}
		}
	} else {
		for i, ip := range toQuery {
			r, err := queryGeoIPAPI(provider, key, tpl, ip)
			if err != nil {
				log.Printf("[geoip-api] 查询 %s 失败: %v", ip, err)
				continue
			}
			saveGeoIPResult(db, ip, *r, provider)
			queried++
			updated++
			if i < len(toQuery)-1 {
				time.Sleep(backfillQueryDelay)
			}
		}
	}

	backfillState.mu.Lock()
	backfillState.lastQueried = queried
	backfillState.lastUpdated = updated
	if queried > 0 {
		backfillState.lastSuccess = time.Now()
	}
	if len(toQuery) > 0 && queried == 0 {
		backfillState.lastError = fmt.Sprintf("本轮 %d 个 IP 查询均失败（详见日志）", len(toQuery))
	} else {
		backfillState.lastError = ""
	}
	backfillState.mu.Unlock()
	log.Printf("[geoip-api] 回填完成：候选 %d，查询 %d，回填 %d 行", len(toQuery), queried, updated)
}

func setBackfillError(db *gorm.DB, msg string) {
	backfillState.mu.Lock()
	backfillState.lastError = msg
	backfillState.mu.Unlock()
	log.Printf("[geoip-api] %s", msg)
}

func clearBackfillError() {
	backfillState.mu.Lock()
	backfillState.lastError = ""
	backfillState.mu.Unlock()
}

// saveGeoIPResult 写缓存并回填访问日志；查询失败（country 为空）只记缓存供下轮重试。
func saveGeoIPResult(db *gorm.DB, ip string, r geoAPIResult, source string) {
	db.Save(&IpGeoCache{
		IP: ip, Country: r.Country, Region: r.Region, City: r.City,
		Lat: r.Lat, Lon: r.Lon, Source: source, CreatedAt: time.Now(),
	})
	if r.Country != "" {
		applyVisitLogGeo(db, ip, r.Country, r.Region, r.City, r.Lat, r.Lon)
	}
}

// applyVisitLogGeo 回填该 IP 全部未识别访问记录的地理信息。
func applyVisitLogGeo(db *gorm.DB, ip, country, region, city string, lat, lon float64) {
	db.Model(&VisitLog{}).
		Where("ip = ? AND country IN ?", ip, []string{"未知", ""}).
		Updates(map[string]interface{}{
			"country": country, "region": region, "city": city, "lat": lat, "lon": lon,
		})
}

// markVisitLogs 将无法通过公共 API 查询的 IP（非法串/私网）标记为固定分类。
func markVisitLogs(db *gorm.DB, ip, country, region string, lat, lon float64) {
	db.Model(&VisitLog{}).
		Where("ip = ? AND country IN ?", ip, []string{"未知", ""}).
		Updates(map[string]interface{}{"country": country, "region": region, "lat": lat, "lon": lon})
}

var geoHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("重定向次数过多")
		}
		return validateDownloadURL(req.URL.String())
	},
}

// queryGeoIPAPI 单条查询，按提供商分发。
func queryGeoIPAPI(provider, key, tpl, ip string) (*geoAPIResult, error) {
	switch provider {
	case "pconline":
		return queryPConline(ip)
	case "baidu":
		return queryBaidu(ip)
	case "ipinfo":
		return queryIPInfo(ip, key)
	case "ipwhois":
		return queryIPWhoIs(ip)
	case "custom":
		return queryCustom(tpl, ip)
	default: // ip-api
		return queryIPAPISingle(ip)
	}
}

func fetchJSON(u string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP")
	resp, err := geoHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("响应非 JSON")
	}
	return m, nil
}

func str(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func num(m map[string]interface{}, keys ...string) float64 {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

// queryIPAPISingle http://ip-api.com（免费版 http；lang=zh-CN 返回中文）。
func queryIPAPISingle(ip string) (*geoAPIResult, error) {
	m, err := fetchJSON("http://ip-api.com/json/" + url.PathEscape(ip) + "?lang=zh-CN&fields=status,message,country,regionName,city,lat,lon")
	if err != nil {
		return nil, err
	}
	if str(m, "status") == "fail" {
		return nil, fmt.Errorf("ip-api: %s", str(m, "message"))
	}
	return &geoAPIResult{
		Country: str(m, "country"), Region: str(m, "regionName"), City: str(m, "city"),
		Lat: num(m, "lat"), Lon: num(m, "lon"),
	}, nil
}

// queryIPAPIBatch http://ip-api.com 批量接口：单次最多 100 个 IP。
func queryIPAPIBatch(ips []string) (map[string]geoAPIResult, error) {
	payload, _ := json.Marshal(ips)
	req, err := http.NewRequest(http.MethodPost,
		"http://ip-api.com/batch?fields=status,query,country,regionName,city,lat,lon", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP")
	resp, err := geoHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, fmt.Errorf("响应非 JSON")
	}
	out := make(map[string]geoAPIResult, len(arr))
	for _, m := range arr {
		ip := str(m, "query")
		if ip == "" || str(m, "status") == "fail" {
			continue
		}
		out[ip] = geoAPIResult{
			Country: str(m, "country"), Region: str(m, "regionName"), City: str(m, "city"),
			Lat: num(m, "lat"), Lon: num(m, "lon"),
		}
	}
	return out, nil
}

// countryNamesZH ipinfo 返回 ISO 代码，转常用中文（未收录保留代码）。
var countryNamesZH = map[string]string{
	"CN": "中国", "US": "美国", "JP": "日本", "KR": "韩国", "SG": "新加坡",
	"DE": "德国", "GB": "英国", "FR": "法国", "RU": "俄罗斯", "CA": "加拿大",
	"AU": "澳大利亚", "IN": "印度", "BR": "巴西", "NL": "荷兰", "IT": "意大利",
	"ES": "西班牙", "TH": "泰国", "MY": "马来西亚", "ID": "印度尼西亚",
	"VN": "越南", "PH": "菲律宾", "TR": "土耳其", "AE": "阿联酋", "MX": "墨西哥",
	"HK": "香港", "TW": "台湾", "MO": "澳门", "SE": "瑞典", "CH": "瑞士", "UA": "乌克兰",
}

// queryIPInfo https://ipinfo.io（无 token 有限额；country 为 ISO 代码）。
func queryIPInfo(ip, key string) (*geoAPIResult, error) {
	u := "https://ipinfo.io/" + ip + "/json"
	if key != "" {
		u += "?token=" + key
	}
	m, err := fetchJSON(u)
	if err != nil {
		return nil, err
	}
	if msg := str(m, "error"); msg != "" {
		return nil, fmt.Errorf("ipinfo: %s", msg)
	}
	country := str(m, "country")
	if zh, ok := countryNamesZH[strings.ToUpper(country)]; ok {
		country = zh
	}
	lat, lon := 0.0, 0.0
	if loc := str(m, "loc"); loc != "" {
		parts := strings.SplitN(loc, ",", 2)
		if len(parts) == 2 {
			lat, _ = strconv.ParseFloat(parts[0], 64)
			lon, _ = strconv.ParseFloat(parts[1], 64)
		}
	}
	return &geoAPIResult{
		Country: country, Region: str(m, "region"), City: str(m, "city"), Lat: lat, Lon: lon,
	}, nil
}

// queryIPWhoIs https://ipwho.is（免费 https，lang=zh 返回中文）。
func queryIPWhoIs(ip string) (*geoAPIResult, error) {
	m, err := fetchJSON("https://ipwho.is/" + ip + "?lang=zh")
	if err != nil {
		return nil, err
	}
	if success, ok := m["success"].(bool); ok && !success {
		return nil, fmt.Errorf("ipwho.is: %s", str(m, "message"))
	}
	return &geoAPIResult{
		Country: str(m, "country"), Region: str(m, "region"), City: str(m, "city"),
		Lat: num(m, "latitude"), Lon: num(m, "longitude"),
	}, nil
}

// queryPConline 太平洋电脑网 ipJson 接口（国内老牌，免费无 key，GBK 编码）。
// 国外 IP 返回 addr=国家；国内 IP 返回 pro/city=省市级、addr=省市+运营商。
func queryPConline(ip string) (*geoAPIResult, error) {
	u := "https://whois.pconline.com.cn/ipJson.jsp?ip=" + url.PathEscape(ip) + "&json=true"
	if err := validateDownloadURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP")
	resp, err := geoHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	// 响应为 GBK 编码（前有若干空行），转 UTF-8 后提取 JSON
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(body)
	if err != nil {
		return nil, fmt.Errorf("GBK 解码失败: %v", err)
	}
	start := strings.Index(string(decoded), "{")
	if start < 0 {
		return nil, errors.New("响应不含 JSON")
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(string(decoded)[start:]), &m); err != nil {
		return nil, fmt.Errorf("响应非 JSON")
	}
	addr := strings.TrimSpace(str(m, "addr"))
	pro := strings.TrimSpace(str(m, "pro"))
	city := strings.TrimSpace(str(m, "city"))
	if addr == "" {
		return nil, errors.New("无归属地信息")
	}
	// 国内地址形如「北京市 」「广东省深圳市 电信」，拆为国家/省/市
	if pro != "" || strings.Contains(addr, "省") || strings.Contains(addr, "市") {
		country := "中国"
		if pro == "" {
			pro = strings.TrimSpace(addr)
		}
		return &geoAPIResult{Country: country, Region: pro, City: city}, nil
	}
	return &geoAPIResult{Country: addr}, nil
}

// queryBaidu 百度开放数据 IP 查询（国内可达，location 为中文混合描述）。
func queryBaidu(ip string) (*geoAPIResult, error) {
	m, err := fetchJSON("https://opendata.baidu.com/api.php?query=" + url.PathEscape(ip) + "&resource_id=6006&oe=utf8")
	if err != nil {
		return nil, err
	}
	arr, ok := m["data"].([]interface{})
	if !ok || len(arr) == 0 {
		return nil, errors.New("无数据")
	}
	first, _ := arr[0].(map[string]interface{})
	loc := str(first, "location")
	if loc == "" {
		return nil, errors.New("无归属地信息")
	}
	// location 形如「美国」「北京市北京市 运营商」：含省/市视为国内，剩余为国家
	cleaned := strings.TrimSpace(loc)
	for _, suffix := range []string{"CNNIC", "电信", "联通", "移动", "铁通", "鹏博士", "教育网"} {
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, suffix))
	}
	if strings.Contains(cleaned, "省") || strings.Contains(cleaned, "市") {
		return &geoAPIResult{Country: "中国", Region: cleaned}, nil
	}
	return &geoAPIResult{Country: cleaned}, nil
}

// queryCustom 自定义 URL 模板（{ip} 占位），宽松解析常见字段名（ip-api 兼容格式）。
func queryCustom(tpl, ip string) (*geoAPIResult, error) {
	if tpl == "" || !strings.Contains(tpl, "{ip}") {
		return nil, errors.New("自定义地址未配置或缺少 {ip} 占位符")
	}
	u := strings.ReplaceAll(tpl, "{ip}", url.PathEscape(ip))
	if err := validateDownloadURL(u); err != nil {
		return nil, err
	}
	m, err := fetchJSON(u)
	if err != nil {
		return nil, err
	}
	if str(m, "status") == "fail" {
		return nil, fmt.Errorf("%s", str(m, "message"))
	}
	return &geoAPIResult{
		Country: str(m, "country", "country_name", "countryCode"),
		Region:  str(m, "regionName", "region"),
		City:    str(m, "city"),
		Lat:     num(m, "lat", "latitude"),
		Lon:     num(m, "lon", "longitude"),
	}, nil
}

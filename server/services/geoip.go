// Package services 封装业务逻辑与数据模型。
package services

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// GeoIP 数据库自动下载与每小时更新。
// 配置全部存于系统设置（网页「系统管理 → 访问地图」配置，持久化在数据库）：
//
//	geoip_enabled      "1" 启用自动下载/更新
//	geoip_url          手动指定下载地址（高级，留空 = 自动在候选源间探测选择）
//	geoip_license_key  MaxMind License Key（官方源需要；社区镜像留空）
//
// 下载源选择：留空时按内置候选源顺序探测（秒级超时），用第一个可用源下载；
// 某个源下载中断自动回退下一个；全部失败时在状态页汇总各源错误。
// 数据库文件固定写入 utils.GeoDBPath，成功后立即热重载，解析无需重启。

// geoIPSource 内置候选下载源。
type geoIPSource struct {
	Name    string
	URL     string
	Type    string // ""=直链文件；"npmmirror"=npmmirror 最新版 tarball（两步：latest JSON → tarball）
	NeedKey bool   // 需要 MaxMind License Key
}

// npmmirrorTarballURL 从 registry.npmmirror.com 的 latest JSON 解析最新 tarball 直链。
func npmmirrorTarballURL(latestURL string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var meta struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("latest 元数据非 JSON")
	}
	if meta.Dist.Tarball == "" {
		return "", errors.New("latest 元数据缺少 tarball")
	}
	return meta.Dist.Tarball, nil
}

// ip2region xdb（中国库）下载源（国内优先）与更新窗口。
var ip2rSources = []struct{ name, url string }{
	{"jsDelivr", "https://fastly.jsdelivr.net/gh/lionsoul2014/ip2region@master/data/ip2region_v4.xdb"},
	{"GitHub", "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb"},
}

const ip2rRefreshInterval = 7 * 24 * time.Hour

// ensureIP2RegionDB 检查并下载/更新中国库（多源回退，7 天更新窗口）。
func ensureIP2RegionDB() {
	fi, err := os.Stat(utils.IP2RegionPath)
	if err == nil {
		if time.Since(fi.ModTime()) < ip2rRefreshInterval {
			if !utils.IP2RegionLoaded() {
				utils.OpenIP2Region(utils.IP2RegionPath)
			}
			return // 新鲜且已加载
		}
		// 陈旧：尝试更新，失败继续用旧库
	}
	for _, s := range ip2rSources {
		if err := downloadIP2RegionDB(s.url); err != nil {
			log.Printf("[geoip] 中国库下载失败（%s）: %v", s.name, err)
			continue
		}
		log.Printf("[geoip] 中国库已就绪（source=%s）", s.name)
		return
	}
	if fi != nil {
		log.Printf("[geoip] 中国库更新失败，继续使用本地旧库")
	}
}

// downloadIP2RegionDB 下载 xdb 到临时文件，加载校验通过后原子替换。
func downloadIP2RegionDB(dlURL string) error {
	if err := validateDownloadURL(dlURL); err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			return validateDownloadURL(req.URL.String())
		},
	}
	req, err := http.NewRequest(http.MethodGet, dlURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmp := utils.IP2RegionPath + ".tmp"
	if err := saveLimited(resp.Body, tmp, 100<<20); err != nil {
		os.Remove(tmp)
		return err
	}
	// 加载校验通过才替换（与 mmdb 相同模式：先开新、关旧、再改名）
	if err := utils.OpenIP2Region(tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("xdb 校验失败: %v", err)
	}
	if err := os.Remove(utils.IP2RegionPath); err != nil && !os.IsNotExist(err) {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, utils.IP2RegionPath); err != nil {
		os.Remove(tmp)
		return err
	}
	return utils.OpenIP2Region(utils.IP2RegionPath)
}

// geoIPSources 内置候选源列表（按国内可达性优先排序）。
var geoIPSources = []geoIPSource{
	// npmmirror（阿里云国内 CDN）分发的 @geo-mmd/geolite2-city：标准 GeoLite2-City
	// 库（约 60MB，geoip2 完全兼容，weekly 更新），tgz 压缩包自动解包。
	// 下载时先从 latest 元数据解析当日 tarball 直链（两步）
	{Name: "npmmirror 镜像", URL: "https://registry.npmmirror.com/@geo-mmd/geolite2-city/latest", Type: "npmmirror"},
	// 社区自动构建的 GeoLite2-City.mmdb（IPv4+IPv6 合并，每日构建；GitHub 直连，
	// 国内不稳定时自动回退下一源）
	{Name: "社区镜像", URL: "https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-City.mmdb"},
	// MaxMind 官方（需 License Key，{key} 会被替换；返回 tar.gz 压缩包）
	{Name: "MaxMind 官方", URL: "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key={key}&suffix=tar.gz", NeedKey: true},
}

const geoIPUpdateInterval = time.Hour

type geoIPState struct {
	mu          sync.Mutex
	lastCheck   time.Time
	lastSuccess time.Time
	lastSource  string
	lastError   string
	downloading bool
}

var geoipState geoIPState

// GeoIPStatus 下载器运行状态（管理端展示）。
type GeoIPStatus struct {
	Enabled     bool                `json:"enabled"`
	AutoSource  bool                `json:"auto_source"`
	URL         string              `json:"url"`
	Source      string              `json:"source"`
	Loaded      bool                `json:"loaded"`
	FileExists  bool                `json:"file_exists"`
	FileSize    int64               `json:"file_size"`
	FileModTime time.Time           `json:"file_mod_time"`
	LastCheck   time.Time           `json:"last_check"`
	LastSuccess time.Time           `json:"last_success"`
	LastError   string              `json:"last_error"`
	Downloading bool                `json:"downloading"`
	Backfill    GeoIPBackfillStatus `json:"backfill"`
	CNDB        GeoIPCNDBStatus     `json:"cn_db"`
}

// GeoIPCNDBStatus 中国库（ip2region xdb）状态。
type GeoIPCNDBStatus struct {
	Loaded   bool  `json:"loaded"`
	FileSize int64 `json:"file_size"`
}

// StartGeoIPUpdater 启动 GeoIP 更新协程：启动时立即检查一次，此后每小时检查一次。
func StartGeoIPUpdater(db *gorm.DB) {
	// 已有数据文件则先加载，重启后立即可用
	if err := utils.OpenGeoDB(utils.GeoDBPath); err != nil {
		log.Printf("[geoip] 未找到本地数据库（%v），等待自动下载", err)
	}
	go func() {
		checkGeoIPUpdate(db)
		ticker := time.NewTicker(geoIPUpdateInterval)
		defer ticker.Stop()
		for range ticker.C {
			checkGeoIPUpdate(db)
		}
	}()
}

// TriggerGeoIPUpdate 管理端「立即更新」入口。
func TriggerGeoIPUpdate(db *gorm.DB) error {
	geoipState.mu.Lock()
	downloading := geoipState.downloading
	geoipState.mu.Unlock()
	if downloading {
		return errors.New("已有更新任务在进行中")
	}
	go checkGeoIPUpdate(db)
	return nil
}

// GetGeoIPStatus 返回当前状态与配置。
func GetGeoIPStatus(db *gorm.DB) GeoIPStatus {
	all := allSettings(db)
	manual := strings.TrimSpace(all["geoip_url"])
	geoipState.mu.Lock()
	st := GeoIPStatus{
		Enabled:     all["geoip_enabled"] == "1",
		AutoSource:  manual == "",
		URL:         manual,
		Source:      geoipState.lastSource,
		LastCheck:   geoipState.lastCheck,
		LastSuccess: geoipState.lastSuccess,
		LastError:   geoipState.lastError,
		Downloading: geoipState.downloading,
	}
	geoipState.mu.Unlock()
	if st.FileSize == 0 {
		if fi, err := os.Stat(utils.GeoDBPath); err == nil {
			st.FileExists = true
			st.FileSize = fi.Size()
			st.FileModTime = fi.ModTime()
		}
	}
	st.Loaded = utils.GeoDBLoaded()
	if fi, err := os.Stat(utils.IP2RegionPath); err == nil {
		st.CNDB.FileSize = fi.Size()
	}
	st.CNDB.Loaded = utils.IP2RegionLoaded() || st.CNDB.FileSize > 0
	st.Backfill = GetGeoIPBackfillStatus(db)
	return st
}

func checkGeoIPUpdate(db *gorm.DB) {
	geoipState.mu.Lock()
	if geoipState.downloading {
		geoipState.mu.Unlock()
		return
	}
	geoipState.downloading = true
	geoipState.lastCheck = time.Now()
	geoipState.mu.Unlock()

	defer func() {
		geoipState.mu.Lock()
		geoipState.downloading = false
		geoipState.mu.Unlock()
	}()

	all := allSettings(db)
	if all["geoip_enabled"] != "1" {
		return
	}

	// 顺带维护中国库（ip2region xdb）：不存在或超过 7 天则更新
	ensureIP2RegionDB()

	// 组装候选源：手动 URL 优先且唯一；否则按内置列表探测
	key := strings.TrimSpace(all["geoip_license_key"])
	manual := strings.TrimSpace(all["geoip_url"])
	type attempt struct {
		name, url string
	}
	var attempts []attempt
	if manual != "" {
		attempts = append(attempts, attempt{"手动指定", manual})
	} else {
		for _, s := range geoIPSources {
			if s.NeedKey && key == "" {
				continue // 官方源缺少 License Key，跳过
			}
			u := strings.ReplaceAll(s.URL, "{key}", url.QueryEscape(key))
			attempts = append(attempts, attempt{s.Name, u})
		}
	}
	if len(attempts) == 0 {
		setGeoIPError("没有可用下载源：MaxMind 官方源需要先在下方填写 License Key")
		return
	}

	var errs []string
	for _, a := range attempts {
		if err := validateDownloadURL(a.url); err != nil {
			errs = append(errs, a.name+" 地址不安全: "+err.Error())
			continue
		}
		if err := probeGeoIPSource(a.url); err != nil {
			errs = append(errs, a.name+" 不可达: "+err.Error())
			continue
		}
		changed, err := downloadGeoDB(db, all, a.name, a.url)
		if err != nil {
			errs = append(errs, a.name+" 下载失败: "+err.Error())
			continue
		}
		// 成功（或 304 无更新）：记录来源并结束
		geoipState.mu.Lock()
		geoipState.lastError = ""
		geoipState.lastSource = a.name
		if changed {
			geoipState.lastSuccess = time.Now()
		}
		geoipState.mu.Unlock()
		log.Printf("[geoip] 检查完成（source=%s changed=%v）", a.name, changed)
		return
	}
	setGeoIPError("所有下载源均不可达 — " + strings.Join(errs, "；"))
}

func setGeoIPError(msg string) {
	geoipState.mu.Lock()
	geoipState.lastError = msg
	geoipState.mu.Unlock()
	log.Printf("[geoip] 更新失败: %s", msg)
}

// probeGeoIPSource 轻量探测源可用性：Range 请求第一个字节，秒级失败。
func probeGeoIPSource(dlURL string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			return validateDownloadURL(req.URL.String())
		},
	}
	req, err := http.NewRequest(http.MethodGet, dlURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-0")
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 16))
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
		return nil
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}

// downloadGeoDB 下载并原子替换 mmdb。changed=false 表示远端无更新（304）。
// 官方源返回 tar.gz 压缩包（按 gzip 魔数识别），自动解包提取 .mmdb。
func downloadGeoDB(db *gorm.DB, all map[string]string, sourceName, dlURL string) (bool, error) {
	target := utils.GeoDBPath
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}

	// npmmirror 源：先从 latest 元数据解析当日 tarball 直链（手拼版本号不可靠）
	if strings.Contains(sourceName, "npmmirror") {
		tar, err := npmmirrorTarballURL(dlURL)
		if err != nil {
			return false, fmt.Errorf("解析 npmmirror 最新版本失败: %v", err)
		}
		dlURL = tar
	}

	client := &http.Client{
		Timeout: 15 * time.Minute,
		// 每一跳重定向都重新做 SSRF 校验，防止下载源 302 跳到内网地址
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			return validateDownloadURL(req.URL.String())
		},
	}

	req, err := http.NewRequest(http.MethodGet, dlURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "ShortLinkGo-GeoIP-Updater")
	if etag := all["geoip_etag"]; etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		log.Printf("[geoip] %s 远端无更新（304）", sourceName)
		return false, nil
	case resp.StatusCode != http.StatusOK:
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	raw := target + ".download"
	if err := saveLimited(resp.Body, raw, 512<<20); err != nil {
		os.Remove(raw)
		return false, err
	}

	tmp := target + ".tmp"
	if err := extractMMDB(raw, tmp); err != nil {
		os.Remove(raw)
		os.Remove(tmp)
		return false, err
	}
	os.Remove(raw)

	// 用 geoip2 打开临时文件验证完整性，成功才替换
	if err := utils.OpenGeoDB(tmp); err != nil {
		os.Remove(tmp)
		return false, fmt.Errorf("下载数据校验失败: %v", err)
	}
	// Windows 上 rename 不能覆盖已存在文件；OpenGeoDB(tmp) 已关闭旧文件句柄
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		os.Remove(tmp)
		return false, err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return false, err
	}

	if etag := resp.Header.Get("ETag"); etag != "" {
		upsertSetting(db, "geoip_etag", etag)
	}
	if err := utils.OpenGeoDB(target); err != nil {
		return true, fmt.Errorf("已下载但加载数据库失败: %v", err)
	}
	log.Printf("[geoip] 数据库已更新（source=%s size=%.1fMB）", sourceName, float64(fileSize(raw, tmp, target))/1024/1024)
	return true, nil
}

func fileSize(paths ...string) int64 {
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil {
			return fi.Size()
		}
	}
	return 0
}

// saveLimited 把响应体落盘，限制最大字节数，防止异常响应撑爆磁盘。
func saveLimited(r io.Reader, path string, maxBytes int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxBytes))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if n < 1024*1024 {
		return fmt.Errorf("下载数据异常（仅 %d 字节）", n)
	}
	return nil
}

// extractMMDB 从下载内容得到 .mmdb：gzip/tar 压缩包自动解包提取，原始 mmdb 直接改用。
func extractMMDB(raw, tmp string) error {
	f, err := os.Open(raw)
	if err != nil {
		return err
	}
	defer f.Close()

	head := make([]byte, 2)
	if _, err := io.ReadFull(f, head); err != nil {
		return fmt.Errorf("下载内容为空")
	}
	if head[0] != 0x1f || head[1] != 0x8b { // 非 gzip：视为原始 mmdb
		f.Close()
		if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Rename(raw, tmp)
	}

	// gzip（MaxMind 官方 tar.gz）：遍历提取包内 .mmdb
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip 解压失败: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return errors.New("压缩包内未找到 .mmdb 文件")
		}
		if err != nil {
			return fmt.Errorf("压缩包读取失败: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg && strings.HasSuffix(strings.ToLower(hdr.Name), ".mmdb") {
			out, err := os.Create(tmp)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("解压 .mmdb 失败: %v", err)
			}
			out.Close()
			return nil
		}
	}
}

// validateDownloadURL 限制仅 http/https 公网地址，拒绝内网/环回/保留地址（SSRF 防护）。
func validateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("地址无法解析")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("缺少主机名")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") ||
		strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") {
		return fmt.Errorf("拒绝内网主机 %s", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("拒绝非公网地址 %s", host)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("主机 %s 无法解析", host)
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("主机 %s 解析到非公网地址", host)
		}
	}
	return nil
}

// isPublicIP 是否公网地址（排除环回/私有/链路本地/组播/未指定/CGNAT）。
func isPublicIP(ip net.IP) bool {
	return utils.PublicIP(ip)
}

// upsertSetting 内部写入设置项（不经过管理接口白名单）。
func upsertSetting(db *gorm.DB, key, value string) {
	var count int64
	if err := db.Model(&Setting{}).Where("key = ?", key).Count(&count).Error; err != nil {
		log.Printf("[geoip] 写入设置 %s 失败: %v", key, err)
		return
	}
	if count == 0 {
		db.Create(&Setting{Key: key, Value: value})
		return
	}
	db.Model(&Setting{}).Where("key = ?", key).UpdateColumn("value", value)
}

// allSettings 读取全部设置（内部使用）。
func allSettings(db *gorm.DB) map[string]string {
	var rows []Setting
	out := make(map[string]string, 16)
	if err := db.Find(&rows).Error; err != nil {
		log.Printf("[geoip] 读取设置失败: %v", err)
		return out
	}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out
}

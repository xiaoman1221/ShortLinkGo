package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// codeAlphabet 短码字符集（去除易混淆字符 0/O/1/l/I）。
const codeAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// customCodePattern 自定义短码规则：2-32 位字母/数字/下划线/连字符。
var customCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{2,32}$`)

// reservedCodes 与系统路由冲突的保留短码。
var reservedCodes = map[string]bool{
	"api": true, "assets": true, "docs": true, "uploads": true, "login": true,
	"register": true, "forgot": true, "reset": true, "favicon.ico": true,
}

// 链接状态。
const (
	LinkStatusDisabled = 0 // 停用
	LinkStatusActive   = 1 // 已启用（公开可访问）
	LinkStatusPending  = 2 // 待审核
)

// LinkService 短链接服务。
type LinkService struct {
	DB *gorm.DB
}

// NewLinkService 构造 LinkService。
func NewLinkService(db *gorm.DB) *LinkService {
	return &LinkService{DB: db}
}

// List 分页查询短链接：管理员/超级管理员可见全部，普通用户仅本人。
func (s *LinkService) List(userID uint, role string, page, pageSize int) ([]Link, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := s.DB.Model(&Link{})
	if !IsStaff(role) {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Link
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// Get 查询一条短链接：管理员可查任意，普通用户仅本人。
func (s *LinkService) Get(id, userID uint, role string) (*Link, error) {
	var l Link
	q := s.DB.Where("id = ?", id)
	if !IsStaff(role) {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("短链接不存在")
		}
		return nil, err
	}
	return &l, nil
}

// GetByCode 查询某短码对应的短链接（任意用户，公开访问用）。
func (s *LinkService) GetByCode(code string) (*Link, error) {
	var l Link
	if err := s.DB.Where("code = ?", code).First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("短链接不存在")
		}
		return nil, err
	}
	return &l, nil
}

// Create 创建短链接；code 为空时自动生成唯一短码。
// 普通用户新建链接默认「待审核」，管理员/超级管理员直接启用。
func (s *LinkService) Create(userID uint, role, rawURL, remark, code string, expireAt *time.Time) (*Link, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("目标链接不能为空")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return nil, errors.New("目标链接必须以 http:// 或 https:// 开头")
	}
	if expireAt != nil && time.Now().After(*expireAt) {
		return nil, errors.New("过期时间不能早于当前时间")
	}

	status := LinkStatusPending
	if IsStaff(role) {
		status = LinkStatusActive
	}
	link := &Link{
		UserID:   userID,
		URL:      rawURL,
		Remark:   strings.TrimSpace(remark),
		ExpireAt: expireAt,
		Status:   status,
	}

	if code = strings.TrimSpace(code); code != "" {
		if !customCodePattern.MatchString(code) {
			return nil, errors.New("短码仅支持 2-32 位字母、数字、下划线或连字符")
		}
		if reservedCodes[strings.ToLower(code)] {
			return nil, errors.New("该短码为系统保留字，请换一个")
		}
		exists, err := s.codeExists(code)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("该短码已被使用，请换一个")
		}
		link.Code = code
	} else {
		for i := 0; i < 10; i++ {
			candidate := randomCode(6)
			exists, err := s.codeExists(candidate)
			if err != nil {
				return nil, err
			}
			if !exists {
				link.Code = candidate
				break
			}
		}
		if link.Code == "" {
			return nil, errors.New("生成短码失败，请重试")
		}
	}

	if err := s.DB.Create(link).Error; err != nil {
		return nil, err
	}
	return link, nil
}

// Review 审核短链接：设置状态（通过=1/停用=0/待审核=2）。
func (s *LinkService) Review(id, userID uint, role string, status int) (*Link, error) {
	if status != LinkStatusActive && status != LinkStatusDisabled && status != LinkStatusPending {
		return nil, errors.New("无效的状态")
	}
	if !IsStaff(role) {
		return nil, errors.New("需要管理员权限")
	}
	l, err := s.Get(id, userID, role)
	if err != nil {
		return nil, err
	}
	if err := s.DB.Model(&Link{}).Where("id = ?", l.ID).UpdateColumn("status", status).Error; err != nil {
		return nil, err
	}
	l.Status = status
	return l, nil
}

func (s *LinkService) codeExists(code string) (bool, error) {
	var count int64
	err := s.DB.Model(&Link{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}

// UpdateReq 更新短链接的请求字段（指针字段表示可空）。
type UpdateReq struct {
	URL         *string    `json:"url"`
	Remark      *string    `json:"remark"`
	Status      *int       `json:"status"`
	ExpireAt    *time.Time `json:"expire_at"`
	ClearExpire bool       `json:"clear_expire"`
}

// Update 更新短链接：管理员可更新任意，普通用户仅本人。
func (s *LinkService) Update(id, userID uint, role string, req UpdateReq) (*Link, error) {
	l, err := s.Get(id, userID, role)
	if err != nil {
		return nil, err
	}
	if req.URL != nil {
		url := strings.TrimSpace(*req.URL)
		if url == "" {
			return nil, errors.New("目标链接不能为空")
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return nil, errors.New("目标链接必须以 http:// 或 https:// 开头")
		}
		l.URL = url
	}
	if req.Remark != nil {
		l.Remark = strings.TrimSpace(*req.Remark)
	}
	if req.Status != nil {
		if *req.Status != LinkStatusDisabled && *req.Status != LinkStatusActive {
			return nil, errors.New("状态值无效")
		}
		if !IsStaff(role) && l.Status == LinkStatusPending && *req.Status == LinkStatusActive {
			return nil, errors.New("待审核链接不可自行启用，请等待管理员审核")
		}
		l.Status = *req.Status
	}
	if req.ClearExpire {
		l.ExpireAt = nil
	} else if req.ExpireAt != nil {
		if time.Now().After(*req.ExpireAt) {
			return nil, errors.New("过期时间不能早于当前时间")
		}
		l.ExpireAt = req.ExpireAt
	}
	if err := s.DB.Save(l).Error; err != nil {
		return nil, err
	}
	return l, nil
}

// Delete 删除短链接：管理员可删除任意，普通用户仅本人。链接与其访问日志同事务删除。
func (s *LinkService) Delete(id, userID uint, role string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		q := tx.Where("id = ?", id)
		if !IsStaff(role) {
			q = q.Where("user_id = ?", userID)
		}
		res := q.Delete(&Link{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("短链接不存在")
		}
		return tx.Where("link_id = ?", id).Delete(&VisitLog{}).Error
	})
}

// Resolve 根据短码解析目标链接，并累计访问次数、记录访问日志（含地理位置）。
func (s *LinkService) Resolve(code, ip, userAgent, referer string) (string, error) {
	var l Link
	err := s.DB.Where("code = ?", code).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("短链接不存在")
		}
		return "", err
	}
	switch l.Status {
	case LinkStatusPending:
		return "", errors.New("短链接审核中")
	case LinkStatusDisabled:
		return "", errors.New("短链接已被停用")
	}
	if l.ExpireAt != nil && time.Now().After(*l.ExpireAt) {
		return "", errors.New("短链接已过期")
	}

	if err := s.DB.Model(&Link{}).Where("id = ?", l.ID).
		UpdateColumn("visit_count", gorm.Expr("visit_count + 1")).Error; err != nil {
		return "", err
	}
	g := utils.Lookup(ip)
	if err := s.DB.Create(&VisitLog{
		LinkID:    l.ID,
		IP:        cut(ip, 64),
		Country:   cut(g.Country, 64),
		Region:    cut(g.Region, 64),
		City:      cut(g.City, 64),
		Lat:       g.Lat,
		Lon:       g.Lon,
		UserAgent: cut(userAgent, 512),
		Referer:   cut(referer, 512),
	}).Error; err != nil {
		// 日志失败不影响跳转，但要留下排查痕迹
		log.Printf("[link] 记录访问日志失败 link=%d: %v", l.ID, err)
	}
	return l.URL, nil
}

// Summary 统计概览。
func (s *LinkService) Summary(userID uint, role string) (map[string]int64, error) {
	q := s.DB.Model(&Link{})
	if !IsStaff(role) {
		q = q.Where("user_id = ?", userID)
	}
	var totalLinks, activeLinks, expiredLinks, pendingLinks int64
	if err := q.Count(&totalLinks).Error; err != nil {
		return nil, err
	}
	if err := q.Where("status = ?", LinkStatusActive).Count(&activeLinks).Error; err != nil {
		return nil, err
	}
	if err := q.Where("status = ?", LinkStatusPending).Count(&pendingLinks).Error; err != nil {
		return nil, err
	}
	if err := q.Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Count(&expiredLinks).Error; err != nil {
		return nil, err
	}
	var totalVisits int64
	vq := s.DB.Model(&Link{}).Select("COALESCE(SUM(visit_count), 0)")
	if !IsStaff(role) {
		vq = vq.Where("user_id = ?", userID)
	}
	if err := vq.Scan(&totalVisits).Error; err != nil {
		return nil, err
	}
	return map[string]int64{
		"total_links":   totalLinks,
		"active_links":  activeLinks,
		"pending_links": pendingLinks,
		"expired_links": expiredLinks,
		"total_visits":  totalVisits,
	}, nil
}

// Trend 返回最近 days 天每天访问量（含 0 的天）。
func (s *LinkService) Trend(userID uint, role string, days int) ([]map[string]interface{}, error) {
	if days < 1 || days > 90 {
		days = 14
	}
	from := time.Now().AddDate(0, 0, -(days - 1))
	type row struct {
		Day string
		Cnt int64
	}
	var rows []row
	// date(created_at, 'localtime')：与 Go 端本地日期对齐。SQLite 对带时区的
	// 时间串默认按 UTC 取日期，会让本地凌晨的访问被计入前一天。
	q := `SELECT date(created_at, 'localtime') AS day, COUNT(*) AS cnt FROM visit_logs
		WHERE created_at >= ?`
	args := []interface{}{from}
	if !IsStaff(role) {
		q += ` AND link_id IN (SELECT id FROM links WHERE user_id = ?)`
		args = append(args, userID)
	}
	q += ` GROUP BY date(created_at) ORDER BY date(created_at)`
	err := s.DB.Raw(q, args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		counts[r.Day] = r.Cnt
	}
	out := make([]map[string]interface{}, 0, days)
	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		out = append(out, map[string]interface{}{
			"date":  d.Format("01-02"),
			"count": counts[key],
		})
	}
	return out, nil
}

// Top 返回访问量最高的短链接。
func (s *LinkService) Top(userID uint, role string, limit int) ([]Link, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	q := s.DB.Model(&Link{})
	if !IsStaff(role) {
		q = q.Where("user_id = ?", userID)
	}
	var list []Link
	err := q.Order("visit_count DESC, id ASC").Limit(limit).Find(&list).Error
	return list, err
}

// GeoStats 按国家聚合访问地理分布。
type GeoStat struct {
	Country string  `json:"country"`
	Count   int64   `json:"count"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// chinaCountryNames 归入「中国地图」统计的国家/地区名（GeoLite2 zh-CN 名称及常见全称变体）。
var chinaCountryNames = []string{
	"中国", "中国大陆", "台湾", "中国台湾", "中华台北", "台湾省",
	"香港", "中国香港", "香港特别行政区",
	"澳门", "中国澳门", "澳门特别行政区",
}

// GeoDistribution 返回访问者地理分布（用于访问地图）。
// scope=world（默认）按国家聚合；scope=china 过滤中国及港澳台，按省级行政区聚合。
func (s *LinkService) GeoDistribution(userID uint, role string, days int, scope string) ([]GeoStat, error) {
	if days < 1 || days > 90 {
		days = 30
	}
	from := time.Now().AddDate(0, 0, -(days - 1))
	type row struct {
		Country string
		Cnt     int64
		Lat     float64
		Lon     float64
	}
	var rows []row
	q := `SELECT COALESCE(NULLIF(%s,''), '未知') AS country, COUNT(*) AS cnt,
		AVG(CASE WHEN lat <> 0 THEN lat END) AS lat,
		AVG(CASE WHEN lon <> 0 THEN lon END) AS lon
		FROM visit_logs WHERE created_at >= ?`
	args := []interface{}{from}
	if !IsStaff(role) {
		// 普通用户仅统计本人的链接访问
		q += ` AND link_id IN (SELECT id FROM links WHERE user_id = ?)`
		args = append(args, userID)
	}
	// 分组列：world 按国家、china 按省级行政区（GeoIP subdivision）。
	// 注意 GROUP BY 必须引用源表列名——SQLite 会把别名解析回源列，按别名分组会出错。
	groupCol := "country"
	if scope == "china" {
		groupCol = "region"
		q += ` AND country IN ?`
		args = append(args, chinaCountryNames)
	}
	q = fmt.Sprintf(q, groupCol) + ` GROUP BY ` + groupCol + ` ORDER BY cnt DESC`
	err := s.DB.Raw(q, args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]GeoStat, 0, len(rows))
	for _, r := range rows {
		if r.Country == "" {
			r.Country = "未知"
		}
		out = append(out, GeoStat{Country: r.Country, Count: r.Cnt, Lat: r.Lat, Lon: r.Lon})
	}
	return out, nil
}

func randomCode(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

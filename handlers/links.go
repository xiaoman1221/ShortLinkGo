package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/models"
	"ShortLinkGo/services"
	"ShortLinkGo/utils"
)

// LinkHandler 短链接相关接口。
type LinkHandler struct {
	Svc  *services.LinkService
	Host string
}

// linkVO 返回给前端的视图对象（附带 short_url 与创建者）。
type linkVO struct {
	models.Link
	ShortURL  string `json:"short_url"`
	OwnerName string `json:"owner_name,omitempty"`
}

func (h *LinkHandler) toVO(l *models.Link) linkVO {
	return linkVO{Link: *l, ShortURL: h.shortURL(l.Code)}
}

func (h *LinkHandler) shortURL(code string) string {
	host := strings.TrimRight(h.Host, "/")
	if host == "" {
		return "/" + code
	}
	return host + "/" + code
}

func roleOf(c *gin.Context) string {
	v, _ := c.Get("role")
	s, _ := v.(string)
	if s == "" {
		return services.RoleUser
	}
	return s
}

func userID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("userID")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// List GET /api/links?page=1&page_size=20
func (h *LinkHandler) List(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	role := roleOf(c)
	list, total, err := h.Svc.List(uid, role, page, pageSize)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	vos := make([]linkVO, 0, len(list))
	names := map[uint]string{}
	if services.IsStaff(role) {
		names = h.ownerNames(list)
	}
	for i := range list {
		vo := h.toVO(&list[i])
		if n, ok := names[list[i].UserID]; ok {
			vo.OwnerName = n
		}
		vos = append(vos, vo)
	}
	utils.OK(c, gin.H{"list": vos, "total": total, "page": page, "page_size": pageSize})
}

func (h *LinkHandler) ownerNames(links []models.Link) map[uint]string {
	ids := make([]uint, 0, len(links))
	seen := map[uint]bool{}
	for _, l := range links {
		if !seen[l.UserID] {
			seen[l.UserID] = true
			ids = append(ids, l.UserID)
		}
	}
	if len(ids) == 0 {
		return map[uint]string{}
	}
	var users []models.User
	if err := h.Svc.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return map[uint]string{}
	}
	out := map[uint]string{}
	for _, u := range users {
		out[u.ID] = u.Username
	}
	return out
}

// Create POST /api/links
func (h *LinkHandler) Create(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var req struct {
		URL      string `json:"url" binding:"required"`
		Code     string `json:"code"`
		Remark   string `json:"remark"`
		ExpireAt string `json:"expire_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	var expireAt *time.Time
	if strings.TrimSpace(req.ExpireAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpireAt))
		if err != nil {
			utils.BadRequest(c, "expire_at 格式错误，应为 RFC3339")
			return
		}
		expireAt = &t
	}
	l, err := h.Svc.Create(uid, roleOf(c), req.URL, req.Remark, req.Code, expireAt)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	msg := "创建成功"
	if l.Status == services.LinkStatusPending {
		msg = "创建成功，待管理员审核"
	}
	utils.OKMsg(c, msg, h.toVO(l))
}

// Get GET /api/links/:id
func (h *LinkHandler) Get(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的链接 ID")
		return
	}
	l, err := h.Svc.Get(uint(id), uid, roleOf(c))
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}
	utils.OK(c, h.toVO(l))
}

// Update PUT /api/links/:id
func (h *LinkHandler) Update(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的链接 ID")
		return
	}
	var req services.UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	l, err := h.Svc.Update(uint(id), uid, roleOf(c), req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "更新成功", h.toVO(l))
}

// Review POST /api/links/:id/review  body: {"status":1}（管理员审核）
func (h *LinkHandler) Review(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的链接 ID")
		return
	}
	var req struct {
		Status *int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 不能用 binding:"required"：它会把合法的 0（停用）当作缺参拦截
	if req.Status == nil {
		utils.BadRequest(c, "缺少 status 字段")
		return
	}
	l, err := h.Svc.Review(uint(id), uid, roleOf(c), *req.Status)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	switch l.Status {
	case services.LinkStatusActive:
		utils.OKMsg(c, "已通过审核", h.toVO(l))
	case services.LinkStatusPending:
		utils.OKMsg(c, "已设为待审核", h.toVO(l))
	default:
		utils.OKMsg(c, "已停用", h.toVO(l))
	}
}

// Delete DELETE /api/links/:id
func (h *LinkHandler) Delete(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的链接 ID")
		return
	}
	if err := h.Svc.Delete(uint(id), uid, roleOf(c)); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.OKMsg(c, "删除成功", nil)
}

// Summary GET /api/stats/summary
func (h *LinkHandler) Summary(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	s, err := h.Svc.Summary(uid, roleOf(c))
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, s)
}

// Trend GET /api/stats/trend?days=14
func (h *LinkHandler) Trend(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	list, err := h.Svc.Trend(uid, roleOf(c), days)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, list)
}

// Top GET /api/stats/top?limit=5
func (h *LinkHandler) Top(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	list, err := h.Svc.Top(uid, roleOf(c), limit)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	vos := make([]linkVO, 0, len(list))
	for i := range list {
		vos = append(vos, h.toVO(&list[i]))
	}
	utils.OK(c, vos)
}

// Geo GET /api/stats/geo?days=30&scope=world|china 访问者地理分布
func (h *LinkHandler) Geo(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	scope := c.DefaultQuery("scope", "world")
	if scope != "world" && scope != "china" {
		scope = "world"
	}
	list, err := h.Svc.GeoDistribution(uid, roleOf(c), days, scope)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.OK(c, list)
}

// HandleRedirect 处理短码跳转（由路由 NoRoute 兜底调用）。
func (h *LinkHandler) HandleRedirect(c *gin.Context, code string) {
	url, err := h.Svc.Resolve(code, c.ClientIP(), c.GetHeader("User-Agent"), c.GetHeader("Referer"))
	if err != nil {
		c.String(http.StatusNotFound, "短链接不存在或已失效")
		return
	}
	c.Redirect(http.StatusFound, url)
}

// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/utils"
)

// 真实访客 IP 探测。
//
// 四种模式（网页「系统管理 → 访问地图」配置，存于数据库）：
//
//	smart  （默认）直连地址为私网/回环时视为部署在反代/容器之后，探测 header；
//	         直连为公网（裸机直接暴露）时不信任任何 header，防伪造
//	always 始终探测 header（Cloudflare 等回源 IP 为公网的场景；header 可被伪造）
//	direct 始终使用直连地址，完全不读 header
//	cidr   仅当直连地址命中 client_ip_trusted_proxies 配置的代理段时才探测 header
//
// header 优先级（高→低）：CF-Connecting-IP、X-Real-IP、X-Forwarded-For（从右往左
// 取首个公网地址，抗伪造插入）、True-Client-IP、X-Client-IP、Forwarded（RFC 7239）。
// 全部为私网时取最左侧地址（代理链中的原始客户端，地理解析显示「内网」）。

// forwardedHeaders 按可信度排序的转发头。
var forwardedHeaders = []string{
	"CF-Connecting-IP",
	"X-Real-IP",
	"X-Forwarded-For",
	"True-Client-IP",
	"X-Client-IP",
	"X-Forwarded",
	"Forwarded-For",
	"Forwarded",
}

// resolveClientIP 按模式解析真实访客 IP。
func resolveClientIP(c *gin.Context, mode string, trustedCIDRs []*net.IPNet) string {
	remote := net.ParseIP(c.RemoteIP()) // gin 1.12 返回 string
	ip := remote

	switch mode {
	case "direct":
		// 保持 ip = remote
	case "cidr":
		if matchCIDR(remote, trustedCIDRs) {
			ip = firstHeaderIP(c, remote)
		}
	case "always":
		ip = firstHeaderIP(c, remote)
	default: // smart
		if isPrivateOrLoopback(remote) {
			ip = firstHeaderIP(c, remote)
		}
	}

	if ip == nil {
		ip = remote
	}
	return ip.String()
}

// firstHeaderIP 按优先级探测转发头，返回提取到的 IP；全部无公网地址时取最左侧原始客户端。
func firstHeaderIP(c *gin.Context, fallback net.IP) net.IP {
	var leftmost net.IP
	for _, name := range forwardedHeaders {
		raw := c.GetHeader(name)
		if raw == "" {
			continue
		}
		if name == "Forwarded" {
			raw = forwardedForOf(raw)
			if raw == "" {
				continue
			}
		}
		parts := strings.Split(raw, ",")
		var firstSeen net.IP
		// 从右往左取首个公网地址：XFF 伪造者只能插入左侧，代理追加的临近地址更可信
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				continue
			}
			if firstSeen == nil {
				firstSeen = ip
			}
			if utils.PublicIP(ip) {
				return ip
			}
		}
		if leftmost == nil && firstSeen != nil {
			leftmost = firstSeen
		}
	}
	if leftmost != nil {
		return leftmost
	}
	return fallback
}

// forwardedForOf 从 RFC 7239 Forwarded 头提取 for= 的地址。
func forwardedForOf(v string) string {
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(strings.ToLower(part), "for=") {
			continue
		}
		val := strings.Trim(strings.TrimPrefix(part[len("for="):], "\""), "[]")
		return val
	}
	return ""
}

// matchCIDR 直连地址是否命中任一可信代理段。
func matchCIDR(ip net.IP, cidrs []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range cidrs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// isPrivateOrLoopback 私网/回环判断（判断部署是否处于反代之后）。
func isPrivateOrLoopback(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// parseCIDRList 解析逗号分隔的 CIDR/IP 列表，非法项忽略。
func parseCIDRList(list string) []*net.IPNet {
	var out []*net.IPNet
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !strings.Contains(item, "/") {
			item += "/32"
			if strings.Contains(item, ":") {
				item = strings.Replace(item, "/32", "/128", 1)
			}
		}
		if _, n, err := net.ParseCIDR(item); err == nil {
			out = append(out, n)
		}
	}
	return out
}

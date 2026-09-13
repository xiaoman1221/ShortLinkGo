// Package services 提供 QQ 互联（官方 OAuth2）登录能力。
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"ShortLinkGo/server/utils"
)

// QQConfig QQ 互联应用配置。
type QQConfig struct {
	AppID  string
	AppKey string
}

// Enabled QQ 登录是否已配置。
func (c QQConfig) Enabled() bool {
	return c.AppID != "" && c.AppKey != ""
}

// QQOAuth QQ 互联客户端。
type QQOAuth struct {
	HTTP *http.Client
}

// NewQQOAuth 构造 QQOAuth。
func NewQQOAuth() *QQOAuth {
	return &QQOAuth{HTTP: &http.Client{Timeout: 10 * time.Second}}
}

const qqAuthorizeURL = "https://graph.qq.com/oauth2.0/authorize"
const qqTokenURL = "https://graph.qq.com/oauth2.0/token"
const qqMeURL = "https://graph.qq.com/oauth2.0/me"
const qqUserInfoURL = "https://graph.qq.com/user/get_user_info"

// QQUser QQ 用户信息。
type QQUser struct {
	OpenID   string
	Nickname string
	Avatar   string // 100x100
	Gender   string
}

// AuthorizeURL 生成 QQ 授权地址。
func (q *QQOAuth) AuthorizeURL(cfg QQConfig, redirectURI, state string) string {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", cfg.AppID)
	v.Set("redirect_uri", redirectURI)
	v.Set("state", state)
	v.Set("scope", "get_user_info")
	return qqAuthorizeURL + "?" + v.Encode()
}

// Exchange 用授权码换取 access_token。
func (q *QQOAuth) Exchange(cfg QQConfig, code, redirectURI string) (string, error) {
	v := url.Values{}
	v.Set("grant_type", "authorization_code")
	v.Set("client_id", cfg.AppID)
	v.Set("client_secret", cfg.AppKey)
	v.Set("code", code)
	v.Set("redirect_uri", redirectURI)
	v.Set("fmt", "json")

	resp, err := q.HTTP.Get(qqTokenURL + "?" + v.Encode())
	if err != nil {
		return "", fmt.Errorf("获取 QQ access_token 失败: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       int    `json:"error"`
		ErrorMsg    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		// QQ 部分场景返回 application/x-www-form-urlencoded
		vals, perr := url.ParseQuery(string(body))
		if perr != nil {
			return "", errors.New("QQ 返回格式无法解析")
		}
		if tok := vals.Get("access_token"); tok != "" {
			return tok, nil
		}
		return "", errors.New("QQ 授权失败: " + vals.Get("error_description"))
	}
	if tokenResp.AccessToken == "" || tokenResp.Error != 0 {
		return "", errors.New("QQ 授权失败: " + tokenResp.ErrorMsg)
	}
	return tokenResp.AccessToken, nil
}

// OpenID 通过 access_token 获取 openid。
func (q *QQOAuth) OpenID(accessToken string) (string, error) {
	v := url.Values{}
	v.Set("access_token", accessToken)
	v.Set("fmt", "json")
	resp, err := q.HTTP.Get(qqMeURL + "?" + v.Encode())
	if err != nil {
		return "", fmt.Errorf("获取 QQ openid 失败: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// QQ 返回 callback( {...} ); 提取其中的 JSON
	re := regexp.MustCompile(`callback\s*\((.*)\)\s*;?`)
	m := re.FindSubmatch(body)
	if len(m) == 2 {
		body = m[1]
	}
	var me struct {
		OpenID string `json:"openid"`
		Error  int    `json:"error"`
		Msg    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		return "", errors.New("QQ openid 解析失败")
	}
	if me.OpenID == "" || me.Error != 0 {
		return "", errors.New("QQ openid 获取失败: " + me.Msg)
	}
	return me.OpenID, nil
}

// UserInfo 获取 QQ 用户资料。
func (q *QQOAuth) UserInfo(cfg QQConfig, accessToken, openid string) (QQUser, error) {
	v := url.Values{}
	v.Set("access_token", accessToken)
	v.Set("oauth_consumer_key", cfg.AppID)
	v.Set("openid", openid)
	resp, err := q.HTTP.Get(qqUserInfoURL + "?" + v.Encode())
	if err != nil {
		return QQUser{}, fmt.Errorf("获取 QQ 用户信息失败: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return QQUser{}, err
	}
	var info struct {
		Ret        int    `json:"ret"`
		Msg        string `json:"msg"`
		Nickname   string `json:"nickname"`
		Gender     string `json:"gender"`
		FigureURL2 string `json:"figureurl_qq_2"` // 100x100
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return QQUser{}, errors.New("QQ 用户信息解析失败")
	}
	if info.Ret != 0 {
		return QQUser{}, errors.New("QQ 用户信息获取失败: " + info.Msg)
	}
	return QQUser{
		OpenID:   openid,
		Nickname: info.Nickname,
		Avatar:   info.FigureURL2,
		Gender:   info.Gender,
	}, nil
}

// ---------- OAuth state 存储（单实例内存，10 分钟有效） ----------

var qqStateMu sync.Mutex
var qqStates = map[string]time.Time{}

// NewQQState 生成并登记一个 state，顺带清理已过期的旧条目。
func NewQQState() string {
	s := utils.RandomHex(16)
	now := time.Now()
	qqStateMu.Lock()
	for k, exp := range qqStates {
		if now.After(exp) {
			delete(qqStates, k)
		}
	}
	qqStates[s] = now.Add(10 * time.Minute)
	qqStateMu.Unlock()
	return s
}

// ValidQQState 校验并消费 state。
func ValidQQState(s string) bool {
	if s == "" {
		return false
	}
	qqStateMu.Lock()
	defer qqStateMu.Unlock()
	exp, ok := qqStates[s]
	if !ok {
		return false
	}
	delete(qqStates, s)
	return time.Now().Before(exp)
}

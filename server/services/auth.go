// Package services 封装业务逻辑。
package services

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// 角色与状态常量。
const (
	RoleSuper    = "super" // 超级管理员（UID=1）
	RoleAdmin    = "admin" // 管理员
	RoleUser     = "user"  // 普通用户
	RoleVIP      = "vip"   // VIP
	StatusOK     = 1       // 正常
	StatusBanned = 0       // 封禁
)

// emailPattern 邮箱格式：非空白本地部分 @ 带点的域名。
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// IsStaff 是否为管理员/超级管理员。
func IsStaff(role string) bool {
	return role == RoleAdmin || role == RoleSuper
}

// AuthService 用户认证服务。
type AuthService struct {
	DB       *gorm.DB
	JWTKey   string
	TokenTTL time.Duration
}

// NewAuthService 构造 AuthService。
func NewAuthService(db *gorm.DB, jwtKey string, ttl time.Duration) *AuthService {
	return &AuthService{DB: db, JWTKey: jwtKey, TokenTTL: ttl}
}

// Register 注册新用户。首个注册用户自动成为超级管理员（UID=1）。
// 查重与创建放在同一事务内，配合 users(email) 部分唯一索引防止并发下重复注册。
func (s *AuthService) Register(username, password, nickname, email string) (*User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" || password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}
	if len([]rune(username)) > 64 {
		return nil, errors.New("用户名过长（最多 64 字）")
	}
	if len(password) < 6 {
		return nil, errors.New("密码长度不能少于 6 位")
	}
	if !emailPattern.MatchString(email) {
		return nil, errors.New("请填写有效的邮箱地址")
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &User{
		Username:     username,
		PasswordHash: hash,
		Nickname:     strings.TrimSpace(nickname),
		Email:        email,
		Status:       StatusOK,
	}
	if u.Nickname == "" {
		u.Nickname = username
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("用户名已存在")
		}
		if err := tx.Model(&User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("该邮箱已被注册")
		}
		// 首个用户为超级管理员
		var total int64
		if err := tx.Model(&User{}).Count(&total).Error; err != nil {
			return err
		}
		u.Role = RoleUser
		if total == 0 {
			u.Role = RoleSuper
		}
		if err := tx.Create(u).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) ||
				strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
				return errors.New("用户名或邮箱已被注册")
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Login 登录校验，成功返回 JWT 与用户信息。
func (s *AuthService) Login(username, password string) (string, *User, error) {
	var u User
	err := s.DB.Where("username = ?", strings.TrimSpace(username)).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户名或密码错误")
		}
		return "", nil, err
	}
	if u.Status != StatusOK {
		return "", nil, errors.New("账号已被封禁，请联系管理员")
	}
	if !utils.CheckPassword(u.PasswordHash, password) {
		return "", nil, errors.New("用户名或密码错误")
	}
	token, err := utils.GenerateToken(u.ID, u.Username, u.Role, s.JWTKey, s.TokenTTL)
	if err != nil {
		return "", nil, err
	}
	return token, &u, nil
}

// GetByID 按 ID 查询用户。
func (s *AuthService) GetByID(id uint) (*User, error) {
	var u User
	if err := s.DB.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &u, nil
}

// FindByEmail 按邮箱查找用户；未找到时返回 (nil, nil)。
func (s *AuthService) FindByEmail(email string) (*User, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, nil
	}
	var u User
	err := s.DB.Where("email = ?", email).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// UpdateProfile 更新个人资料（昵称/邮箱/手机号/头像），用户名不可修改。
func (s *AuthService) UpdateProfile(id uint, nickname, email, phone, avatar string) (*User, error) {
	u, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	email = strings.TrimSpace(email)
	if email != "" {
		if !strings.Contains(email, "@") {
			return nil, errors.New("邮箱格式不正确")
		}
		var count int64
		if err := s.DB.Model(&User{}).
			Where("email = ? AND id <> ?", email, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("该邮箱已被其他账号使用")
		}
		u.Email = email
	}
	if nickname != "" {
		u.Nickname = nickname
	}
	if phone != "" {
		u.Phone = phone
	}
	if avatar != "" {
		u.Avatar = avatar
	}
	if err := s.DB.Save(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// ChangePassword 修改密码。
func (s *AuthService) ChangePassword(id uint, oldPwd, newPwd string) error {
	if len(newPwd) < 6 {
		return errors.New("新密码长度不能少于 6 位")
	}
	u, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if u.PasswordHash != "" && !utils.CheckPassword(u.PasswordHash, oldPwd) {
		return errors.New("原密码不正确")
	}
	hash, err := utils.HashPassword(newPwd)
	if err != nil {
		return err
	}
	return s.DB.Model(&User{}).Where("id = ?", id).UpdateColumn("password_hash", hash).Error
}

// SetAvatar 更新头像。
func (s *AuthService) SetAvatar(id uint, avatar string) (*User, error) {
	if err := s.DB.Model(&User{}).Where("id = ?", id).UpdateColumn("avatar", avatar).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

// FindOrCreateByQQ 通过 QQ openid 查找或创建本地账号。
func (s *AuthService) FindOrCreateByQQ(openid, nickname, avatar string) (*User, error) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, errors.New("QQ openid 为空")
	}
	var u User
	err := s.DB.Where("qq_openid = ?", openid).First(&u).Error
	if err == nil {
		if u.Status != StatusOK {
			return nil, errors.New("账号已被封禁")
		}
		if avatar != "" && u.Avatar == "" {
			s.DB.Model(&User{}).Where("id = ?", u.ID).UpdateColumn("avatar", avatar)
		}
		return &u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var total int64
	if err := s.DB.Model(&User{}).Count(&total).Error; err != nil {
		return nil, err
	}
	role := RoleUser
	if total == 0 {
		role = RoleSuper // 系统首个账号
	}
	if strings.TrimSpace(nickname) == "" {
		// openid 长度不保证 ≥ 8，安全截断避免切片越界
		nickname = "QQ用户" + openid[:min(8, len(openid))]
	}
	u = User{
		Username: "qq_" + openid,
		Nickname: cut(nickname, 32),
		Avatar:   avatar,
		QQOpenID: openid,
		Role:     role,
		Status:   StatusOK,
	}
	if err := s.DB.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateResetToken 为指定用户生成找回密码令牌（30 分钟有效），返回明文令牌。
func (s *AuthService) CreateResetToken(userID uint) (string, error) {
	token := utils.RandomHex(24)
	hash := utils.HashToken(token)
	now := time.Now()
	if err := s.DB.Model(&PasswordReset{}).
		Where("user_id = ? AND used = ?", userID, false).
		Updates(map[string]interface{}{"used": true}).Error; err != nil {
		return "", err
	}
	rec := &PasswordReset{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: now.Add(30 * time.Minute),
		Used:      false,
	}
	if err := s.DB.Create(rec).Error; err != nil {
		return "", err
	}
	return token, nil
}

// ResetPassword 校验令牌并重置密码。
func (s *AuthService) ResetPassword(token, newPwd string) error {
	if len(newPwd) < 6 {
		return errors.New("新密码长度不能少于 6 位")
	}
	if token == "" {
		return errors.New("重置令牌无效")
	}
	var rec PasswordReset
	err := s.DB.Where("token_hash = ? AND used = ?", utils.HashToken(token), false).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("重置链接无效或已使用")
		}
		return err
	}
	if time.Now().After(rec.ExpiresAt) {
		return errors.New("重置链接已过期，请重新申请")
	}
	hash, err := utils.HashPassword(newPwd)
	if err != nil {
		return err
	}
	if err := s.DB.Model(&User{}).Where("id = ?", rec.UserID).
		UpdateColumn("password_hash", hash).Error; err != nil {
		return err
	}
	return s.DB.Model(&PasswordReset{}).Where("id = ?", rec.ID).UpdateColumn("used", true).Error
}

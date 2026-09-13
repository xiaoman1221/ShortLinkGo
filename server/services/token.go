package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"ShortLinkGo/server/utils"
)

// TokenService API 令牌服务。
type TokenService struct {
	DB *gorm.DB
}

// NewTokenService 构造 TokenService。
func NewTokenService(db *gorm.DB) *TokenService {
	return &TokenService{DB: db}
}

// Create 为用户创建一枚 API 令牌，返回明文（仅此一次）。
func (s *TokenService) Create(userID uint, name string) (string, *ApiToken, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, errors.New("令牌名称不能为空")
	}
	if len(name) > 60 {
		return "", nil, errors.New("令牌名称过长")
	}
	plain := "slg_" + utils.RandomHex(20)
	t := &ApiToken{
		UserID:    userID,
		Name:      name,
		TokenHash: utils.HashToken(plain),
	}
	if err := s.DB.Create(t).Error; err != nil {
		return "", nil, err
	}
	return plain, t, nil
}

// List 列出用户的令牌（不返回哈希）。
func (s *TokenService) List(userID uint) ([]ApiToken, error) {
	var list []ApiToken
	err := s.DB.Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// Delete 删除令牌。
func (s *TokenService) Delete(userID, id uint) error {
	res := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&ApiToken{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("令牌不存在")
	}
	return nil
}

// UserByID 以 ID 获取用户（校验封禁状态），用于 JWT 请求时实时读取角色/状态。
func (s *TokenService) UserByID(id uint) (*User, error) {
	var u User
	if err := s.DB.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if u.Status != StatusOK {
		return nil, errors.New("账号已被封禁")
	}
	return &u, nil
}

// AuthByToken 用明文令牌换取用户；失败返回 nil。
func (s *TokenService) AuthByToken(plain string) (*User, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nil, nil
	}
	var t ApiToken
	err := s.DB.Where("token_hash = ?", utils.HashToken(plain)).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var u User
	if err := s.DB.First(&u, t.UserID).Error; err != nil {
		return nil, nil
	}
	if u.Status != StatusOK {
		return nil, errors.New("账号已被封禁")
	}
	now := time.Now()
	s.DB.Model(&ApiToken{}).Where("id = ?", t.ID).UpdateColumn("last_used_at", &now)
	return &u, nil
}

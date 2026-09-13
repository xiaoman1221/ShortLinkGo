package services

import (
	"errors"
	"strings"
)

// ListUsers 分页列出用户（管理员/超级管理员）。
func (s *AuthService) ListUsers(page, pageSize int, keyword string) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := s.DB.Model(&User{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []User
	err := q.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// SetUserRole 设置用户角色（仅超级管理员；超级管理员自身不可被修改）。
func (s *AuthService) SetUserRole(id uint, role string) error {
	switch role {
	case RoleSuper, RoleAdmin, RoleUser, RoleVIP:
	default:
		return errors.New("无效的角色")
	}
	if id == 1 {
		return errors.New("不能修改超级管理员的角色")
	}
	var u User
	if err := s.DB.First(&u, id).Error; err != nil {
		return errors.New("用户不存在")
	}
	return s.DB.Model(&User{}).Where("id = ?", id).UpdateColumn("role", role).Error
}

// SetUserStatus 设置用户状态（封禁/解封；仅超级管理员；超级管理员不可被封禁）。
func (s *AuthService) SetUserStatus(id uint, status int) error {
	if status != 0 && status != 1 {
		return errors.New("状态仅支持 0（封禁）/ 1（正常）")
	}
	if id == 1 {
		return errors.New("不能封禁超级管理员")
	}
	var u User
	if err := s.DB.First(&u, id).Error; err != nil {
		return errors.New("用户不存在")
	}
	return s.DB.Model(&User{}).Where("id = ?", id).UpdateColumn("status", status).Error
}

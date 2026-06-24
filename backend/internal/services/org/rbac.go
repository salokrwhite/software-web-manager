package org

import (
	"errors"
	"strings"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/rbac"

	"gorm.io/gorm"
)

// IsAssignableOrgRole reports whether a role may be assigned to a member: the
// built-in non-owner roles, or an active custom org role.
func (s *Service) IsAssignableOrgRole(orgID string, role string) bool {
	key := rbac.NormalizeOrgRoleKey(role)
	if key == "" || key == "owner" {
		return false
	}
	if rbac.IsReservedOrgRoleKey(key) {
		return true
	}
	var item models.OrgRole
	if err := s.DB.Where("org_id = ? AND role_name = ? AND status = ?", orgID, key, "active").First(&item).Error; err != nil {
		return false
	}
	return true
}

// ResolveEffectiveOrgRole maps a (possibly custom) role to its effective role
// name, defaulting to viewer when the custom role is missing/inactive.
func (s *Service) ResolveEffectiveOrgRole(orgID string, role string) string {
	key := rbac.NormalizeOrgRoleKey(role)
	if key == "" {
		return "viewer"
	}
	if key == "owner" || key == "admin" || key == "dev" || key == "viewer" {
		return key
	}
	var item models.OrgRole
	if err := s.DB.Where("org_id = ? AND role_name = ? AND status = ?", orgID, key, "active").First(&item).Error; err != nil {
		return "viewer"
	}
	return item.RoleName
}

// LoadOrgPermissionSet resolves the effective permission set for an org member's
// role, honoring custom role bindings and falling back to built-in defaults.
func (s *Service) LoadOrgPermissionSet(orgID string, role string) (map[string]struct{}, error) {
	roleName := rbac.NormalizeOrgRoleKey(role)
	if roleName == "" {
		codes, _ := rbac.DefaultRoleCodes("viewer")
		return rbac.ToPermissionSet(codes), nil
	}
	if roleName == "owner" {
		return map[string]struct{}{"*": {}}, nil
	}

	_, isBuiltIn := rbac.DefaultRoleCodes(roleName)
	var roleItem models.OrgRole
	err := s.DB.Where("org_id = ? AND role_name = ?", orgID, roleName).First(&roleItem).Error
	if err == nil && strings.ToLower(strings.TrimSpace(roleItem.Status)) != "active" {
		return map[string]struct{}{}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil && !isBuiltIn {
		return map[string]struct{}{}, nil
	}

	var bindings []models.OrgRolePermission
	if err := s.DB.Where("org_id = ? AND role_name = ?", orgID, roleName).Find(&bindings).Error; err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		codes, _ := rbac.DefaultRoleCodes(roleName)
		return rbac.ToPermissionSet(codes), nil
	}
	codes := make([]string, 0, len(bindings))
	for _, b := range bindings {
		codes = append(codes, b.PermissionCode)
	}
	return rbac.ToPermissionSet(codes), nil
}

// LoadAppPermissionSet returns the permission set for an app member's role.
func (s *Service) LoadAppPermissionSet(role string) map[string]struct{} {
	roleName := rbac.NormalizeOrgRoleKey(role)
	if roleName == "owner" {
		return map[string]struct{}{"*": {}}
	}
	codes, _ := rbac.DefaultRoleCodes(roleName)
	return rbac.ToPermissionSet(codes)
}

// HasAppPermission reports whether a user holds an app-scoped permission via
// their app membership role.
func (s *Service) HasAppPermission(userID string, appID string, permissionCode string) bool {
	if userID == "" || appID == "" {
		return false
	}
	var member models.AppMember
	if err := s.DB.Where("scope_id = ? AND user_id = ?", appID, userID).First(&member).Error; err != nil {
		return false
	}
	set := s.LoadAppPermissionSet(member.Role)
	return rbac.PermissionSetAllows(set, strings.ToLower(strings.TrimSpace(permissionCode)))
}

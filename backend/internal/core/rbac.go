package core

import (
	"errors"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/rbac"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The RBAC model (catalog, role permissions, set operations, role-key helpers)
// lives in internal/rbac. The aliases below preserve the existing
// core.PermissionRoleXxx / core.NormalizeOrgRoleKey call sites; the
// methods are thin org/app-scoped wrappers that add DB and gin context.

const (
	ContextPermissions = rbac.ContextPermissions

	PermissionRoleViewer = rbac.PermissionRoleViewer
	PermissionRoleDev    = rbac.PermissionRoleDev
	PermissionRoleAdmin  = rbac.PermissionRoleAdmin
	PermissionRoleOwner  = rbac.PermissionRoleOwner
)

// WithRequiredTiers re-exports rbac.WithRequiredTiers.
func WithRequiredTiers(codes []string) []string { return rbac.WithRequiredTiers(codes) }

// ListPermissionCatalog re-exports rbac.ListPermissionCatalog.
func ListPermissionCatalog() []rbac.Permission { return rbac.ListPermissionCatalog() }

// NormalizeOrgRoleKey re-exports rbac.NormalizeOrgRoleKey.
func NormalizeOrgRoleKey(role string) string { return rbac.NormalizeOrgRoleKey(role) }

// IsValidRole re-exports rbac.IsValidRole.
func IsValidRole(role string) bool { return rbac.IsValidRole(role) }

// IsReservedOrgRoleKey re-exports rbac.IsReservedOrgRoleKey.
func IsReservedOrgRoleKey(role string) bool { return rbac.IsReservedOrgRoleKey(role) }

func (h *Handler) IsAssignableOrgRole(orgID string, role string) bool {
	key := rbac.NormalizeOrgRoleKey(role)
	if key == "" || key == "owner" {
		return false
	}
	if rbac.IsReservedOrgRoleKey(key) {
		return true
	}
	var item models.OrgRole
	if err := h.DB.Where("org_id = ? AND role_name = ? AND status = ?", orgID, key, "active").First(&item).Error; err != nil {
		return false
	}
	return true
}

func (h *Handler) ResolveEffectiveOrgRole(orgID string, role string) string {
	key := rbac.NormalizeOrgRoleKey(role)
	if key == "" {
		return "viewer"
	}
	if key == "owner" || key == "admin" || key == "dev" || key == "viewer" {
		return key
	}
	var item models.OrgRole
	if err := h.DB.Where("org_id = ? AND role_name = ? AND status = ?", orgID, key, "active").First(&item).Error; err != nil {
		return "viewer"
	}
	return item.RoleName
}

func (h *Handler) LoadOrgPermissionSet(orgID string, role string) (map[string]struct{}, error) {
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
	err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleName).First(&roleItem).Error
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
	if err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleName).Find(&bindings).Error; err != nil {
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

func (h *Handler) HasPermission(c *gin.Context, code string) bool {
	key := strings.ToLower(strings.TrimSpace(code))
	if key == "" {
		return false
	}
	// Platform system admins (including while impersonating an org) are not org
	// members and therefore have no ContextPermissions set by the middleware.
	// They act with full authority over org resources — consistent with the
	// system_admin bypasses elsewhere (e.g. online stream, release ops) — so
	// they pass every permission check. Without this, the org-scoped read gates
	// would 403 during impersonation.
	if strings.EqualFold(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)), "system_admin") {
		return true
	}
	permissions, _ := c.Get(ContextPermissions)
	switch v := permissions.(type) {
	case map[string]struct{}:
		return rbac.PermissionSetAllows(v, key)
	case []string:
		return rbac.PermissionSetAllows(rbac.ToPermissionSet(v), key)
	}
	return false
}

func (h *Handler) RequirePermission(c *gin.Context, code string) bool {
	if h.HasPermission(c, code) {
		return true
	}
	c.JSON(403, gin.H{"error": "insufficient role"})
	return false
}

func (h *Handler) LoadAppPermissionSet(role string) map[string]struct{} {
	roleName := rbac.NormalizeOrgRoleKey(role)
	if roleName == "owner" {
		return map[string]struct{}{"*": {}}
	}
	codes, _ := rbac.DefaultRoleCodes(roleName)
	return rbac.ToPermissionSet(codes)
}

func (h *Handler) HasAppPermission(userID string, appID string, permissionCode string) bool {
	if userID == "" || appID == "" {
		return false
	}
	var member models.AppMember
	if err := h.DB.Where("scope_id = ? AND user_id = ?", appID, userID).First(&member).Error; err != nil {
		return false
	}
	set := h.LoadAppPermissionSet(member.Role)
	return rbac.PermissionSetAllows(set, strings.ToLower(strings.TrimSpace(permissionCode)))
}

func GetRequestOrgID(c *gin.Context) string {
	return strings.TrimSpace(c.GetString(middleware.ContextOrgID))
}

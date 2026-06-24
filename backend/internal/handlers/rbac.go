package handlers

import (
	"errors"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ContextPermissions = "permissions"

const (
	PermissionRoleViewer = "role.viewer"
	PermissionRoleDev    = "role.dev"
	PermissionRoleAdmin  = "role.admin"
	PermissionRoleOwner  = "role.owner"
)

type permissionMeta struct {
	Code        string `json:"permission_code"`
	Module      string `json:"module"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var permissionCatalog = []permissionMeta{
	{Code: PermissionRoleViewer, Module: "role", Name: "查看权限", Description: "基础查看权限"},
	{Code: PermissionRoleDev, Module: "role", Name: "开发权限", Description: "应用与发布操作权限"},
	{Code: PermissionRoleAdmin, Module: "role", Name: "管理权限", Description: "组织管理权限"},
	{Code: PermissionRoleOwner, Module: "role", Name: "所有者权限", Description: "组织所有者权限"},
	{Code: "org_management.view", Module: "org_management", Name: "查看组织信息", Description: "查看组织信息"},
	{Code: "org_management.update", Module: "org_management", Name: "更新组织信息", Description: "更新组织基础设置"},
	{Code: "org_management.transfer_owner", Module: "org_management", Name: "转移所有者", Description: "转移组织所有者"},
	{Code: "org_management.delete", Module: "org_management", Name: "删除组织", Description: "删除当前组织"},
	{Code: "member_manage.view", Module: "member_manage", Name: "查看成员", Description: "查看组织成员列表"},
	{Code: "member_manage.create", Module: "member_manage", Name: "新增成员", Description: "创建组织成员"},
	{Code: "member_manage.update", Module: "member_manage", Name: "编辑成员", Description: "编辑成员角色与状态"},
	{Code: "member_manage.delete", Module: "member_manage", Name: "删除成员", Description: "移除组织成员"},
	{Code: "member_invite.manage", Module: "member_invite", Name: "邀请管理", Description: "管理成员邀请"},
	{Code: "org_join_request.review", Module: "org_join_request", Name: "审批加入申请", Description: "审批用户加入组织申请"},
	{Code: "org_join_request.manage_own", Module: "org_join_request", Name: "管理我的申请", Description: "查看与撤回我的加入申请"},
	{Code: "role_manage.view", Module: "role_manage", Name: "查看角色", Description: "查看权限类型"},
	{Code: "role_manage.edit", Module: "role_manage", Name: "管理角色", Description: "管理权限类型与绑定权限点"},
	{Code: "app.manage", Module: "app", Name: "应用管理", Description: "创建与编辑应用"},
	{Code: "release.manage", Module: "release", Name: "发布管理", Description: "创建与发布版本"},
	{Code: "ticket.manage", Module: "ticket", Name: "工单管理", Description: "处理工单"},
	{Code: "audit_log.view", Module: "audit_log", Name: "查看审计日志", Description: "查看审计日志"},
}

var defaultRolePermissions = map[string][]string{
	"viewer": {
		PermissionRoleViewer,
		"org_management.view",
		"member_manage.view",
		"org_join_request.manage_own",
		"role_manage.view",
		"audit_log.view",
	},
	"dev": {
		PermissionRoleViewer,
		PermissionRoleDev,
		"org_management.view",
		"member_manage.view",
		"org_join_request.manage_own",
		"role_manage.view",
		"audit_log.view",
		"app.manage",
		"release.manage",
		"ticket.manage",
	},
	"admin": {
		PermissionRoleViewer,
		PermissionRoleDev,
		PermissionRoleAdmin,
		"org_management.view",
		"org_management.update",
		"member_manage.view",
		"member_manage.create",
		"member_manage.update",
		"member_manage.delete",
		"member_invite.manage",
		"org_join_request.review",
		"org_join_request.manage_own",
		"role_manage.view",
		"role_manage.edit",
		"audit_log.view",
		"app.manage",
		"release.manage",
		"ticket.manage",
	},
}

// permissionTier maps each fine-grained permission to the role-tier marker it
// requires. Holding the fine-grained permission alone is not sufficient — the
// caller's role must also carry the corresponding tier marker. The owner
// wildcard "*" satisfies every tier. This makes the role.* markers real,
// layered gates on top of the existing fine-grained checks.
var permissionTier = map[string]string{
	"org_management.view":         PermissionRoleViewer,
	"member_manage.view":          PermissionRoleViewer,
	"org_join_request.manage_own": PermissionRoleViewer,
	"role_manage.view":            PermissionRoleViewer,
	"audit_log.view":              PermissionRoleViewer,

	"app.manage":     PermissionRoleDev,
	"release.manage": PermissionRoleDev,
	"ticket.manage":  PermissionRoleDev,

	"org_management.update":   PermissionRoleAdmin,
	"member_manage.create":    PermissionRoleAdmin,
	"member_manage.update":    PermissionRoleAdmin,
	"member_manage.delete":    PermissionRoleAdmin,
	"member_invite.manage":    PermissionRoleAdmin,
	"org_join_request.review": PermissionRoleAdmin,
	"role_manage.edit":        PermissionRoleAdmin,

	"org_management.transfer_owner": PermissionRoleOwner,
	"org_management.delete":         PermissionRoleOwner,
}

// permissionSetAllows reports whether a normalized permission set grants the
// given code, including the layered role-tier requirement from permissionTier.
func permissionSetAllows(set map[string]struct{}, code string) bool {
	if _, ok := set["*"]; ok {
		return true
	}
	if _, ok := set[code]; !ok {
		return false
	}
	if tier := permissionTier[code]; tier != "" {
		if _, ok := set[tier]; !ok {
			return false
		}
	}
	return true
}

// WithRequiredTiers returns the given permission codes plus any role-tier
// markers required by them (per permissionTier) that are not already present.
// This keeps saved role bindings consistent with the layered enforcement model:
// an admin can grant a fine-grained permission (e.g. app.manage) without having
// to also remember its tier marker (role.dev) — the tier is attached on save so
// the permission never ends up silently inert.
func WithRequiredTiers(codes []string) []string {
	present := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		present[code] = struct{}{}
	}
	result := make([]string, len(codes))
	copy(result, codes)
	for _, code := range codes {
		tier := permissionTier[code]
		if tier == "" {
			continue
		}
		if _, ok := present[tier]; ok {
			continue
		}
		present[tier] = struct{}{}
		result = append(result, tier)
	}
	return result
}

func toPermissionSet(codes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		c := strings.ToLower(strings.TrimSpace(code))
		if c == "" {
			continue
		}
		set[c] = struct{}{}
	}
	return set
}

func ListPermissionCatalog() []permissionMeta {
	return permissionCatalog
}

func NormalizeOrgRoleKey(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func IsValidRole(role string) bool {
	key := NormalizeOrgRoleKey(role)
	return key == "viewer" || key == "dev" || key == "admin" || key == "owner"
}

func IsReservedOrgRoleKey(role string) bool {
	v := NormalizeOrgRoleKey(role)
	return v == "owner" || v == "admin" || v == "dev" || v == "viewer"
}

func (h *Handler) IsAssignableOrgRole(orgID string, role string) bool {
	key := NormalizeOrgRoleKey(role)
	if key == "" || key == "owner" {
		return false
	}
	if IsReservedOrgRoleKey(key) {
		return true
	}
	var item models.OrgRole
	if err := h.DB.Where("org_id = ? AND role_name = ? AND status = ?", orgID, key, "active").First(&item).Error; err != nil {
		return false
	}
	return true
}

func (h *Handler) ResolveEffectiveOrgRole(orgID string, role string) string {
	key := NormalizeOrgRoleKey(role)
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
	roleName := NormalizeOrgRoleKey(role)
	if roleName == "" {
		return toPermissionSet(defaultRolePermissions["viewer"]), nil
	}
	if roleName == "owner" {
		return map[string]struct{}{"*": {}}, nil
	}

	_, isBuiltIn := defaultRolePermissions[roleName]
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
		return toPermissionSet(defaultRolePermissions[roleName]), nil
	}
	codes := make([]string, 0, len(bindings))
	for _, b := range bindings {
		codes = append(codes, b.PermissionCode)
	}
	return toPermissionSet(codes), nil
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
		return permissionSetAllows(v, key)
	case []string:
		return permissionSetAllows(toPermissionSet(v), key)
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
	roleName := NormalizeOrgRoleKey(role)
	if roleName == "owner" {
		return map[string]struct{}{"*": {}}
	}
	return toPermissionSet(defaultRolePermissions[roleName])
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
	return permissionSetAllows(set, strings.ToLower(strings.TrimSpace(permissionCode)))
}

func GetRequestOrgID(c *gin.Context) string {
	return strings.TrimSpace(c.GetString(middleware.ContextOrgID))
}



package rbac

import "strings"

// ContextPermissions is the gin context key under which the caller's resolved
// permission set is stored by the auth middleware.
const ContextPermissions = "permissions"

const (
	PermissionRoleViewer = "role.viewer"
	PermissionRoleDev    = "role.dev"
	PermissionRoleAdmin  = "role.admin"
	PermissionRoleOwner  = "role.owner"
)

// Permission is a catalog entry describing a single fine-grained permission.
type Permission struct {
	Code        string `json:"permission_code"`
	Module      string `json:"module"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var permissionCatalog = []Permission{
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

// PermissionSetAllows reports whether a normalized permission set grants the
// given code, including the layered role-tier requirement from permissionTier.
func PermissionSetAllows(set map[string]struct{}, code string) bool {
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

// ToPermissionSet builds a normalized (lower-cased, trimmed) set from codes.
func ToPermissionSet(codes []string) map[string]struct{} {
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

// ListPermissionCatalog returns the full permission catalog.
func ListPermissionCatalog() []Permission {
	return permissionCatalog
}

// DefaultRoleCodes returns the built-in permission codes for a role and whether
// the role is a recognized built-in.
func DefaultRoleCodes(role string) ([]string, bool) {
	codes, ok := defaultRolePermissions[role]
	return codes, ok
}

// NormalizeOrgRoleKey lower-cases and trims an org role name.
func NormalizeOrgRoleKey(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

// IsValidRole reports whether role is one of the four built-in roles.
func IsValidRole(role string) bool {
	key := NormalizeOrgRoleKey(role)
	return key == "viewer" || key == "dev" || key == "admin" || key == "owner"
}

// IsReservedOrgRoleKey reports whether role is a reserved built-in role key.
func IsReservedOrgRoleKey(role string) bool {
	v := NormalizeOrgRoleKey(role)
	return v == "owner" || v == "admin" || v == "dev" || v == "viewer"
}

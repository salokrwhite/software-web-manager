package handlers

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOrgRoleRequest struct {
	RoleName    string  `json:"role_name" binding:"required"`
	Description *string `json:"description"`
}

type updateOrgRoleRequest struct {
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type updateRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required"`
}

func normalizeRoleDescription(v *string) *string {
	if v == nil {
		return nil
	}
	text := strings.TrimSpace(*v)
	if text == "" {
		return nil
	}
	return &text
}

func ensureOrgScope(c *gin.Context) bool {
	return strings.TrimSpace(c.Param("id")) == getRequestOrgID(c)
}

func (h *Handler) ListOrgRoles(c *gin.Context) {
	if !h.hasPermission(c, "role_manage.view") && !h.hasPermission(c, "role_manage.edit") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	var items []models.OrgRole
	if err := h.DB.Where("org_id = ?", orgID).Order("created_at asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list org roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateOrgRole(c *gin.Context) {
	if !h.requirePermission(c, "role_manage.edit") {
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	var req createOrgRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key := normalizeOrgRoleKey(req.RoleName)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_name required"})
		return
	}
	if len(key) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_name too long"})
		return
	}
	if isReservedOrgRoleKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_name reserved"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	var createdBy *uuid.UUID
	if parsed, err := uuid.Parse(userID); err == nil {
		createdBy = &parsed
	}
	item := models.OrgRole{
		OrgID:       orgUUID,
		RoleName:    key,
		IsBuiltin:   false,
		Description: normalizeRoleDescription(req.Description),
		Status:      "active",
		CreatedBy:   createdBy,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role already exists"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": item})
}

func (h *Handler) UpdateOrgRole(c *gin.Context) {
	if !h.requirePermission(c, "role_manage.edit") {
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	roleKey := normalizeOrgRoleKey(c.Param("role_name"))
	if roleKey == "" || roleKey == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_name"})
		return
	}
	var req updateOrgRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.OrgRole
	if err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleKey).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if req.Description != nil {
		updates["description"] = normalizeRoleDescription(req.Description)
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status != "active" && status != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		if item.IsBuiltin && status != "active" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "builtin role cannot be disabled"})
			return
		}
		updates["status"] = status
	}
	if len(updates) == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	res := h.DB.Model(&models.OrgRole{}).
		Where("org_id = ? AND role_name = ?", orgID, roleKey).
		Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update org role"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleKey).First(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": item})
}

func (h *Handler) DeleteOrgRole(c *gin.Context) {
	if !h.requirePermission(c, "role_manage.edit") {
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	roleKey := normalizeOrgRoleKey(c.Param("role_name"))
	if roleKey == "" || isReservedOrgRoleKey(roleKey) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_name"})
		return
	}
	var usedCount int64
	if err := h.DB.Model(&models.OrgMember{}).Where("scope_id = ? AND role = ?", orgID, roleKey).Count(&usedCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check role usage"})
		return
	}
	if usedCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role in use"})
		return
	}
	if err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleKey).Delete(&models.OrgRole{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete org role"})
		return
	}
	_ = h.DB.Where("org_id = ? AND role_name = ?", orgID, roleKey).Delete(&models.OrgRolePermission{}).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListOrgPermissions(c *gin.Context) {
	if !h.hasPermission(c, "role_manage.view") && !h.hasPermission(c, "role_manage.edit") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": listPermissionCatalog()})
}

func (h *Handler) GetRolePermissions(c *gin.Context) {
	if !h.hasPermission(c, "role_manage.view") && !h.hasPermission(c, "role_manage.edit") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	roleName := normalizeOrgRoleKey(c.Param("role_name"))
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_name"})
		return
	}

	var rows []models.OrgRolePermission
	if err := h.DB.Where("org_id = ? AND role_name = ?", orgID, roleName).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load role permissions"})
		return
	}
	items := make([]string, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.PermissionCode)
	}
	c.JSON(http.StatusOK, gin.H{"role_name": roleName, "permission_codes": items})
}

func (h *Handler) PutRolePermissions(c *gin.Context) {
	if !h.requirePermission(c, "role_manage.edit") {
		return
	}
	if !ensureOrgScope(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgID := c.Param("id")
	roleName := normalizeOrgRoleKey(c.Param("role_name"))
	if roleName == "" || roleName == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_name"})
		return
	}
	var req updateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed := map[string]struct{}{}
	for _, meta := range listPermissionCatalog() {
		allowed[meta.Code] = struct{}{}
	}
	seen := map[string]struct{}{}
	codes := make([]string, 0, len(req.PermissionCodes))
	for _, raw := range req.PermissionCodes {
		code := strings.ToLower(strings.TrimSpace(raw))
		if code == "" {
			continue
		}
		if _, ok := allowed[code]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission_code"})
			return
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("org_id = ? AND role_name = ?", orgID, roleName).Delete(&models.OrgRolePermission{}).Error; err != nil {
			return err
		}
		if len(codes) == 0 {
			return nil
		}
		orgUUID, err := uuid.Parse(orgID)
		if err != nil {
			return err
		}
		batch := make([]models.OrgRolePermission, 0, len(codes))
		for _, code := range codes {
			batch = append(batch, models.OrgRolePermission{
				OrgID:          orgUUID,
				RoleName:       roleName,
				PermissionCode: code,
			})
		}
		return tx.Create(&batch).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save role permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role_name": roleName, "permission_codes": codes})
}


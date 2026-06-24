package org

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	orgsvc "software-web-manager/backend/internal/services/org"
	"software-web-manager/backend/internal/services/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOrgUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required"`
}

type updateOrgMemberRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type updateOrgRequest struct {
	Name *string `json:"name"`
	Plan *string `json:"plan"`
}

type transferOwnerRequest struct {
	NewOwnerUserID string `json:"new_owner_user_id" binding:"required"`
}

func (h *Handler) CreateOrgUser(c *gin.Context) {
	if !h.RequirePermission(c, "member_manage.create") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req createOrgUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if !h.IsAssignableOrgRole(orgID, req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	if req.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use transfer owner to assign owner role"})
		return
	}
	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
			return
		}
		hash, err := crypto.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		user = models.User{Email: req.Email, PasswordHash: hash, Status: "active", SystemRole: "none"}
		if err := h.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	var existing models.OrgMember
	if err := h.DB.Where("scope_id = ? AND user_id = ?", orgUUID, user.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already in org"})
		return
	}
	member := models.OrgMember{OrgID: orgUUID, UserID: user.ID, Role: req.Role}
	if err := h.DB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}
	h.Audit(c, "org_member.add", "org_member", member.OrgID, nil, member)
	c.JSON(http.StatusOK, gin.H{
		"user":   gin.H{"id": user.ID, "email": user.Email},
		"member": member,
	})
}

func (h *Handler) UpdateOrgMember(c *gin.Context) {
	if !h.RequirePermission(c, "member_manage.update") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	userID := c.Param("user_id")
	var req updateOrgMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role == nil && req.Status == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	member, err := h.GetOrgMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}

	updates := map[string]any{}
	if req.Role != nil {
		nextRole := strings.ToLower(strings.TrimSpace(*req.Role))
		if !h.IsAssignableOrgRole(orgID, nextRole) && nextRole != "owner" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}
		if strings.ToLower(member.Role) == "owner" && nextRole != "owner" {
			owners, err := h.CountOrgOwners(orgID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check owners"})
				return
			}
			if owners <= 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot downgrade last owner"})
				return
			}
		}
		updates["role"] = nextRole
	}

	if req.Status != nil {
		nextStatus := strings.ToLower(strings.TrimSpace(*req.Status))
		if nextStatus != "active" && nextStatus != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		currentUserID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
		if currentUserID == userID && nextStatus == "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable current user"})
			return
		}
		if strings.ToLower(strings.TrimSpace(member.Role)) == "owner" && nextStatus == "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable org owner"})
			return
		}
		if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("status", nextStatus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update member status"})
			return
		}
	}

	before := member
	if len(updates) > 0 {
		if err := h.DB.Model(&models.OrgMember{}).
			Where("scope_id = ? AND user_id = ?", orgID, userID).
			Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update member"})
			return
		}
	}
	if err := h.DB.Where("scope_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err == nil {
		h.Audit(c, "org_member.update", "org_member", member.OrgID, before, member)
	}
	c.JSON(http.StatusOK, gin.H{"member": member})
}

func (h *Handler) DeleteOrgMember(c *gin.Context) {
	if !h.RequirePermission(c, "member_manage.delete") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	userID := c.Param("user_id")
	member, err := h.GetOrgMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if strings.ToLower(member.Role) == "owner" {
		owners, err := h.CountOrgOwners(orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check owners"})
			return
		}
		if owners <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove last owner"})
			return
		}
	}
	if err := h.DB.Where("scope_id = ? AND user_id = ?", orgID, userID).Delete(&models.OrgMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}
	h.Audit(c, "org_member.remove", "org_member", member.OrgID, member, nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateOrg(c *gin.Context) {
	if !h.RequirePermission(c, "org_management.update") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req updateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Plan != nil {
		plan := system.NormalizeOrgPlanValue(*req.Plan)
		planTypes, err := system.NewService(h.DB).OrgPlanTypes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org plan types"})
			return
		}
		if !system.IsAllowedOrgPlan(plan, planTypes) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org plan"})
			return
		}
		updates["plan"] = plan
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}
	before := org
	if err := h.DB.Model(&models.Org{}).Where("id = ?", orgID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update org"})
		return
	}
	if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
		h.Audit(c, "org.update", "org", org.ID, before, org)
	}
	c.JSON(http.StatusOK, gin.H{"org": org})
}

func (h *Handler) SwitchOrg(c *gin.Context) {
	systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
	if systemRole == "org_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "org switch disabled"})
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.Param("id")
	member, err := h.GetOrgMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	orgType := ""
	var org models.Org
	if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
		orgType = strings.TrimSpace(org.OrgType)
	}
	normalizedSystemRole := core.NormalizeSystemRole(c.GetString(middleware.ContextSystemRole))
	effectiveRole := h.ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, userID, member.OrgID.String(), effectiveRole, normalizedSystemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tokens":   tokens,
		"org_id":   member.OrgID,
		"role":     effectiveRole,
		"org_type": orgType,
	})
}

func (h *Handler) TransferOwner(c *gin.Context) {
	if !h.RequirePermission(c, "org_management.transfer_owner") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req transferOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newOwnerID := strings.TrimSpace(req.NewOwnerUserID)
	if newOwnerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new owner required"})
		return
	}
	currentUserID := c.GetString(middleware.ContextUserID)
	if newOwnerID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new owner must be different"})
		return
	}
	if _, err := h.GetOrgMember(orgID, newOwnerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "new owner not found"})
		return
	}
	if _, err := h.GetOrgMember(orgID, currentUserID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OrgMember{}).
			Where("scope_id = ? AND user_id = ?", orgID, currentUserID).
			Update("role", "admin").Error; err != nil {
			return err
		}
		return tx.Model(&models.OrgMember{}).
			Where("scope_id = ? AND user_id = ?", orgID, newOwnerID).
			Update("role", "owner").Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer owner"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err == nil {
		h.Audit(c, "org.owner_transfer", "org", orgUUID, nil, gin.H{"new_owner": newOwnerID})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DeleteOrg(c *gin.Context) {
	if !h.RequirePermission(c, "org_management.delete") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}
	h.Audit(c, "org.delete", "org", org.ID, org, nil)
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		return orgsvc.DeleteOrgCascade(tx, orgID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete org"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func parseExpiresAt(days int) *time.Time {
	if days <= 0 {
		days = 7
	}
	exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	return &exp
}

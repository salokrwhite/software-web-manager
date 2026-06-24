package org

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/db/schema"
	attachment "software-web-manager/backend/internal/services/attachment"
	orgsvc "software-web-manager/backend/internal/services/org"
	"strings"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOrgRequest struct {
	Name string `json:"name" binding:"required"`
}

type addMemberRequest struct {
	UserEmail string `json:"user_email" binding:"required,email"`
	Role      string `json:"role" binding:"required"`
}

func (h *Handler) GetOrgPublic(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	// Viewing your own org's info requires org_management.view; previewing
	// another org (join flow) stays open to any authenticated user.
	isSelf := orgID == strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if isSelf {
		if !common.RequirePermission(c, "org_management.view") {
			return
		}
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	orgType := ""
	if schema.HasOrgTypeColumn(h.DB) {
		orgType = org.OrgType
	}
	resp := gin.H{
		"id":         org.ID,
		"name":       org.Name,
		"status":     org.Status,
		"org_type":   orgType,
		"created_at": org.CreatedAt,
	}
	// Plan is only exposed on the gated self view, not in cross-org join previews.
	if isSelf {
		resp["plan"] = org.Plan
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListOrgs(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	systemRole := strings.ToLower(strings.TrimSpace(c.GetString(middleware.ContextSystemRole)))
	type orgListItem struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Plan        string    `json:"plan"`
		CreatedBy   uuid.UUID `json:"created_by"`
		CreatedAt   string    `json:"created_at"`
		Role        string    `json:"role"`
		OrgType     string    `json:"org_type"`
		MemberCount int64     `json:"member_count"`
		AppCount    int64     `json:"app_count"`
	}
	if schema.HasOrgTypeColumn(h.DB) && systemRole != "org_admin" {
		if _, _, err := orgsvc.NewService(h.DB).EnsurePersonalMember(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal org"})
			return
		}
	}
	var items []orgListItem
	selectOrgType := "'' as org_type"
	if schema.HasOrgTypeColumn(h.DB) {
		selectOrgType = "o.org_type as org_type"
	}
	whereClause := "WHERE m.scope_type = 'org' AND m.user_id = ?"
	if schema.HasOrgTypeColumn(h.DB) && systemRole == "org_admin" {
		whereClause += " AND (o.org_type IS NULL OR o.org_type <> 'personal')"
	}
	query := fmt.Sprintf(`
		SELECT o.id, o.name, o.plan, o.created_by, o.created_at, m.role, %s,
		       (SELECT COUNT(*) FROM memberships om WHERE om.scope_type = 'org' AND om.scope_id = o.id) AS member_count,
		       (SELECT COUNT(*) FROM apps a WHERE a.org_id = o.id) AS app_count
		FROM memberships m
		JOIN orgs o ON o.id = m.scope_id
		%s
		ORDER BY o.created_at DESC
	`, selectOrgType, whereClause)
	if err := h.DB.Raw(query, userID).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orgs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateOrg(c *gin.Context) {
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	org := models.Org{Name: req.Name, Plan: "free", OrgType: "enterprise", Status: "active", CreatedBy: uid}
	member := models.OrgMember{OrgID: org.ID, UserID: uid, Role: "owner"}
	hasOrgTypeColumn := schema.HasOrgTypeColumn(h.DB)
	if !hasOrgTypeColumn {
		org.OrgType = ""
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if hasOrgTypeColumn {
			if err := tx.Create(&org).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Omit("org_type").Create(&org).Error; err != nil {
				return err
			}
		}
		member.OrgID = org.ID
		return tx.Create(&member).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create org"})
		return
	}
	common.Audit(h.DB, c, "org.create", "org", org.ID, nil, org)
	c.JSON(http.StatusOK, gin.H{"org": org})
}

func (h *Handler) ListOrgMembers(c *gin.Context) {
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if !common.RequirePermission(c, "member_manage.view") {
		return
	}
	type orgMemberItem struct {
		OrgID     uuid.UUID `json:"OrgID"`
		UserID    uuid.UUID `json:"UserID"`
		Email     string    `json:"Email"`
		Role      string    `json:"Role"`
		Status    string    `json:"Status"`
		CreatedAt string    `json:"CreatedAt"`
	}
	var members []orgMemberItem
	if err := h.DB.Raw(`
		SELECT om.scope_id AS org_id, om.user_id, u.email, om.role, u.status, om.created_at
		FROM memberships om
		JOIN users u ON u.id = om.user_id
		WHERE om.scope_type = 'org' AND om.scope_id = ?
		ORDER BY om.created_at ASC
	`, orgID).Scan(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": members})
}

func (h *Handler) AddOrgMember(c *gin.Context) {
	if !common.RequirePermission(c, "member_manage.create") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if !orgsvc.NewService(h.DB).IsAssignableOrgRole(orgID, req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	if req.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use transfer owner to assign owner role"})
		return
	}
	req.UserEmail = strings.ToLower(strings.TrimSpace(req.UserEmail))
	var user models.User
	if err := h.DB.Where("email = ?", req.UserEmail).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
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
	common.Audit(h.DB, c, "org_member.add", "org_member", member.OrgID, nil, member)
	c.JSON(http.StatusOK, gin.H{"member": member})
}

func (h *Handler) UpgradeOrg(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	var otherOrgCount int64
	if err := h.DB.Model(&models.OrgMember{}).
		Where("scope_type = ? AND user_id = ? AND scope_id <> ?", models.ScopeOrg, userUUID, orgUUID).
		Count(&otherOrgCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check org memberships"})
		return
	}
	if otherOrgCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "leave_other_orgs_required"})
		return
	}
	if err := h.EnsureStorage(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}
	orgName := strings.TrimSpace(c.PostForm("org_name"))
	keepApprovedAppsRaw := strings.ToLower(strings.TrimSpace(c.PostForm("keep_approved_apps")))
	keepApprovedApps := true
	if keepApprovedAppsRaw != "" {
		if keepApprovedAppsRaw == "false" || keepApprovedAppsRaw == "0" || keepApprovedAppsRaw == "no" || keepApprovedAppsRaw == "off" {
			keepApprovedApps = false
		}
	}

	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if schema.HasOrgTypeColumn(h.DB) {
		if strings.ToLower(strings.TrimSpace(org.OrgType)) != "personal" {
			c.JSON(http.StatusForbidden, gin.H{"error": "org not personal"})
			return
		}
	}
	if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": middleware.OrgStatusCode(org.Status)})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}
	files := form.File["materials"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "materials required"})
		return
	}
	for _, file := range files {
		if file.Size > common.MaxEnterpriseMaterialSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material too large"})
			return
		}
	}

	materials, statusCode, err := common.StoreAttachments(h.Storage, h.Cfg.StorageDriver, c, attachment.OwnerOrgRegistrationMaterial, orgUUID, &orgUUID, nil, "materials", filepath.ToSlash(filepath.Join("orgs", orgUUID.String(), "registration_materials")), len(files), common.MaxEnterpriseMaterialSize)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{
		"status":           "pending",
		"rejection_reason": nil,
		"allow_resubmit":   false,
		"resubmit_token":   nil,
		"rejected_by":      nil,
		"rejected_at":      nil,
		"approved_by":      nil,
		"approved_at":      nil,
	}
	if schema.HasOrgTypeColumn(h.DB) {
		updates["org_type"] = "enterprise"
	}
	if orgName != "" && !strings.EqualFold(orgName, "undefined") && !strings.EqualFold(orgName, "null") {
		updates["name"] = orgName
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if !keepApprovedApps {
			var appIDs []string
			if err := tx.Model(&models.App{}).Where("org_id = ?", orgUUID).Pluck("id", &appIDs).Error; err != nil {
				return err
			}
			if len(appIDs) > 0 {
				var releaseIDs []string
				if err := tx.Model(&models.Release{}).Where("app_id IN ?", appIDs).Pluck("id", &releaseIDs).Error; err != nil {
					return err
				}
				if len(releaseIDs) > 0 {
					if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.Artifact{}).Error; err != nil {
						return err
					}
					if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.ReleaseChannel{}).Error; err != nil {
						return err
					}
				}
				if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Release{}).Error; err != nil {
					return err
				}
				if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Channel{}).Error; err != nil {
					return err
				}
				if err := tx.Where("scope_id IN ?", appIDs).Delete(&models.AppMember{}).Error; err != nil {
					return err
				}
				if tx.Migrator().HasTable(&models.Feedback{}) {
					var feedbackIDs []string
					if err := tx.Model(&models.Feedback{}).Where("app_id IN ?", appIDs).Pluck("id", &feedbackIDs).Error; err != nil {
						return err
					}
					if err := attachment.DeleteByOwners(tx, attachment.OwnerFeedback, feedbackIDs); err != nil {
						return err
					}
					if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Feedback{}).Error; err != nil {
						return err
					}
				}
				if tx.Migrator().HasTable(&models.AppSecret{}) {
					if err := tx.Where("app_id IN ?", appIDs).Delete(&models.AppSecret{}).Error; err != nil {
						return err
					}
				}
				if err := tx.Where("id IN ?", appIDs).Delete(&models.App{}).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Model(&models.Org{}).Where("id = ?", orgUUID).Updates(updates).Error; err != nil {
			return err
		}
		for i := range materials {
			if err := tx.Create(&materials[i]).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade org"})
		return
	}

	name := org.Name
	if orgName != "" && !strings.EqualFold(orgName, "undefined") && !strings.EqualFold(orgName, "null") {
		name = orgName
	}
	c.JSON(http.StatusOK, gin.H{
		"pending":      true,
		"apps_deleted": !keepApprovedApps,
		"org": gin.H{
			"id":   orgUUID,
			"name": name,
		},
	})
}

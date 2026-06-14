package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/utils"
	"strings"
	"time"
)

func (h *Handler) ListSystemOrgs(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	orgType := strings.ToLower(strings.TrimSpace(c.Query("org_type")))
	query := `
		SELECT o.id, o.name, o.plan, o.status, o.created_by, o.created_at, o.approved_by, o.approved_at,
		       (SELECT u.email FROM memberships om
		        JOIN users u ON u.id = om.user_id
		        WHERE om.scope_type = 'org' AND om.scope_id = o.id AND om.role = 'owner'
		        ORDER BY om.created_at ASC
		        LIMIT 1) AS owner_email,
		       (SELECT COUNT(*) FROM memberships om WHERE om.scope_type = 'org' AND om.scope_id = o.id) AS member_count,
		       (SELECT COUNT(*) FROM apps a WHERE a.org_id = o.id) AS app_count
		FROM orgs o
	`
	where := "WHERE 1=1"
	args := []any{}
	if status != "" {
		where += " AND o.status = ?"
		args = append(args, status)
	}
	if orgType != "" && h.hasOrgTypeColumn() {
		if orgType == "personal" {
			where += " AND (o.org_type = ? OR o.org_type IS NULL OR o.org_type = '')"
			args = append(args, "personal")
		} else {
			where += " AND o.org_type = ?"
			args = append(args, orgType)
		}
	}
	var items []systemOrgListItem
	if err := h.DB.Raw(query+where+" ORDER BY o.created_at DESC", args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orgs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) ListOrgRegistrationMaterials(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	if h.Storage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	var materials []models.Attachment
	if err := h.DB.Where("owner_type = ? AND owner_id = ?", attachmentOwnerOrgRegistrationMaterial, orgID).Order("created_at desc").Find(&materials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list materials"})
		return
	}
	items := make([]orgRegistrationMaterialItem, 0, len(materials))
	for _, material := range materials {
		url := ""
		if strings.EqualFold(h.Cfg.StorageDriver, "local") {
			url = h.buildLocalFileURL(c, material.StoragePath, 24*time.Hour)
		} else if h.Storage != nil {
			url, _ = h.Storage.GetDownloadURL(c.Request.Context(), material.StoragePath, 24*time.Hour)
		}
		items = append(items, orgRegistrationMaterialItem{
			ID:          material.ID,
			FileName:    material.FileName,
			ContentType: material.ContentType,
			Size:        material.Size,
			CreatedAt:   material.CreatedAt,
			DownloadURL: url,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateSystemOrg(c *gin.Context) {
	var req createSystemOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.OwnerEmail = strings.ToLower(strings.TrimSpace(req.OwnerEmail))
	req.OrgName = strings.TrimSpace(req.OrgName)
	plan := "free"
	if req.Plan != nil && strings.TrimSpace(*req.Plan) != "" {
		plan = normalizeOrgPlanValue(*req.Plan)
		planTypes, planErr := h.getOrgPlanTypes()
		if planErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org plan types"})
			return
		}
		if !isAllowedOrgPlan(plan, planTypes) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org plan"})
			return
		}
	}

	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	now := time.Now()

	var user models.User
	if err := h.DB.Where("email = ?", req.OwnerEmail).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
			return
		}
		password := strings.TrimSpace(req.Password)
		if password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
			return
		}
		if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
			return
		}
		hash, err := utils.HashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		user = models.User{
			Email:        req.OwnerEmail,
			PasswordHash: hash,
			Status:       "active",
			SystemRole:   "org_admin",
		}
		if err := h.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	} else {
		if strings.ToLower(strings.TrimSpace(user.SystemRole)) != "system_admin" && strings.ToLower(strings.TrimSpace(user.SystemRole)) != "org_admin" {
			if err := h.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("system_role", "org_admin").Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user role"})
				return
			}
		}
		if err := h.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("status", "active").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate user"})
			return
		}
	}

	var org models.Org
	var member models.OrgMember
	hasOrgTypeColumn := h.hasOrgTypeColumn()
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		org = models.Org{
			Name:       req.OrgName,
			Plan:       plan,
			OrgType:    "enterprise",
			Status:     "active",
			CreatedBy:  adminID,
			ApprovedBy: &adminID,
			ApprovedAt: &now,
		}
		if !hasOrgTypeColumn {
			org.OrgType = ""
		}
		if hasOrgTypeColumn {
			if err := tx.Create(&org).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Omit("org_type").Create(&org).Error; err != nil {
				return err
			}
		}
		member = models.OrgMember{OrgID: org.ID, UserID: user.ID, Role: "owner"}
		return tx.Create(&member).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create org"})
		return
	}
	h.auditWithOrg(c, org.ID, "system.org.create", "org", org.ID, nil, org)
	c.JSON(http.StatusOK, gin.H{
		"org":   org,
		"owner": gin.H{"id": user.ID, "email": user.Email},
	})
}

func (h *Handler) ApproveSystemOrg(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	var before models.Org
	_ = h.DB.Where("id = ?", orgID).First(&before).Error
	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	now := time.Now()
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Org{}).
			Where("id = ?", orgID).
			Updates(map[string]any{
				"status":           "active",
				"approved_by":      adminID,
				"approved_at":      now,
				"rejection_reason": nil,
				"allow_resubmit":   false,
				"resubmit_token":   nil,
				"rejected_by":      nil,
				"rejected_at":      nil,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		var owner models.OrgMember
		if err := tx.Where("scope_id = ? AND role = ?", orgID, "owner").First(&owner).Error; err == nil {
			if err := tx.Model(&models.User{}).
				Where("id = ? AND system_role <> ?", owner.UserID, "system_admin").
				Update("system_role", "org_admin").Error; err != nil {
				return err
			}
		}
		return tx.Exec(`
			UPDATE users u
			JOIN memberships om ON om.user_id = u.id AND om.scope_type = 'org'
			SET u.status = 'active'
			WHERE om.scope_id = ?
		`, orgID).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve org"})
		return
	}
	var after models.Org
	if err := h.DB.Where("id = ?", orgID).First(&after).Error; err == nil {
		h.auditWithOrg(c, after.ID, "system.org.approve", "org", after.ID, before, after)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RejectSystemOrg(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	var req rejectSystemOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var before models.Org
	if err := h.DB.Where("id = ?", orgID).First(&before).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if strings.ToLower(strings.TrimSpace(before.Status)) != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not pending"})
		return
	}
	now := time.Now()
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
	}
	var resubmitToken *string
	if req.AllowResubmit {
		token := uuid.NewString()
		resubmitToken = &token
	}
	updates := map[string]any{
		"status":         "rejected",
		"allow_resubmit": req.AllowResubmit,
		"resubmit_token": resubmitToken,
		"rejected_by":    adminID,
		"rejected_at":    now,
		"approved_by":    nil,
		"approved_at":    nil,
	}
	if reason == "" {
		updates["rejection_reason"] = nil
	} else {
		updates["rejection_reason"] = reason
	}
	if err := h.DB.Model(&models.Org{}).Where("id = ?", orgID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject org"})
		return
	}
	var after models.Org
	if err := h.DB.Where("id = ?", orgID).First(&after).Error; err == nil {
		h.auditWithOrg(c, after.ID, "system.org.reject", "org", after.ID, before, after)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DisableSystemOrg(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	var before models.Org
	_ = h.DB.Where("id = ?", orgID).First(&before).Error
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Org{}).
			Where("id = ?", orgID).
			Update("status", "disabled")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Exec(`
			UPDATE users u
			JOIN memberships om ON om.user_id = u.id AND om.scope_type = 'org'
			SET u.status = 'disabled'
			WHERE om.scope_id = ?
		`, orgID).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable org"})
		return
	}
	var after models.Org
	if err := h.DB.Where("id = ?", orgID).First(&after).Error; err == nil {
		h.auditWithOrg(c, after.ID, "system.org.disable", "org", after.ID, before, after)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemOrgs(c *gin.Context) {
	var req batchDeleteSystemOrgsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var orgs []models.Org
	if err := h.DB.Where("id IN ?", ids).Find(&orgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load orgs"})
		return
	}
	if len(orgs) == 0 || len(orgs) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "orgs not found"})
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var ownerIDs []uuid.UUID
		if err := tx.Model(&models.OrgMember{}).
			Where("scope_id IN ? AND role = ?", ids, "owner").
			Pluck("user_id", &ownerIDs).Error; err != nil {
			return err
		}
		for _, orgID := range ids {
			if err := deleteOrgCascade(tx, orgID); err != nil {
				return err
			}
		}
		seenOwners := make(map[uuid.UUID]struct{}, len(ownerIDs))
		for _, ownerID := range ownerIDs {
			if _, exists := seenOwners[ownerID]; exists {
				continue
			}
			seenOwners[ownerID] = struct{}{}

			var remainingMemberships int64
			if err := tx.Model(&models.OrgMember{}).Where("scope_type = ? AND user_id = ?", models.ScopeOrg, ownerID).Count(&remainingMemberships).Error; err != nil {
				return err
			}
			if remainingMemberships > 0 {
				continue
			}

			var user models.User
			if err := tx.Where("id = ?", ownerID).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if strings.ToLower(strings.TrimSpace(user.SystemRole)) != "org_admin" {
				continue
			}

			if err := tx.Where("id = ?", ownerID).Delete(&models.User{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete orgs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(ids)})
}

func (h *Handler) BatchDeleteSystemOrgApprovalLogs(c *gin.Context) {
	var req batchDeleteSystemOrgApprovalLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	actions := []string{"system.org.approve", "system.org.reject"}
	res := h.DB.Where("target_type = ? AND target_id IN ? AND action IN ?", "org", ids, actions).Delete(&models.AuditLog{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

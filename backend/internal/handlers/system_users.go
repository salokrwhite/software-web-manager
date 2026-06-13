package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type systemUserItem struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	Status     string    `json:"status"`
	SystemRole string    `json:"system_role"`
	OrgName    string    `json:"org_name"`
	OrgRole    string    `json:"org_role"`
	OrgCount   int64     `json:"org_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type resetSystemUserPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

type batchDeleteSystemUsersRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type updateSystemUserRequest struct {
	Email      *string `json:"email"`
	Password   *string `json:"password"`
	Status     *string `json:"status"`
	SystemRole *string `json:"system_role"`
}

func (h *Handler) ListSystemUsers(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	status := strings.TrimSpace(c.Query("status"))
	systemRole := strings.TrimSpace(c.Query("system_role"))
	orgID := strings.TrimSpace(c.Query("org_id"))
	role := strings.TrimSpace(c.Query("role"))

	hasOrgTypeColumn := h.hasOrgTypeColumn()
	orgTypeFilter := ""
	if hasOrgTypeColumn {
		orgTypeFilter = " AND o.org_type = 'enterprise'"
	}
	orgInfoJoin := " JOIN orgs o ON o.id = om.scope_id"

	ownerExists := "SELECT 1 FROM memberships om"
	if hasOrgTypeColumn {
		ownerExists += " JOIN orgs o ON o.id = om.scope_id"
	}
	ownerExists += " WHERE om.scope_type = 'org' AND om.user_id = u.id AND om.role = 'owner'"
	if hasOrgTypeColumn {
		ownerExists += " AND o.org_type = 'enterprise'"
	}

	roleExists := "SELECT 1 FROM memberships om"
	if hasOrgTypeColumn {
		roleExists += " JOIN orgs o ON o.id = om.scope_id"
	}
	roleExists += " WHERE om.scope_type = 'org' AND om.user_id = u.id AND om.role = ?"
	if hasOrgTypeColumn {
		roleExists += " AND o.org_type = 'enterprise'"
	}

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	where := "WHERE 1=1"
	args := []any{}
	if q != "" {
		where += " AND u.email LIKE ?"
		args = append(args, "%"+q+"%")
	}
	if status != "" {
		where += " AND u.status = ?"
		args = append(args, status)
	}
	if systemRole != "" {
		if systemRole == "org_admin" {
			where += " AND (u.system_role = ? OR EXISTS (" + ownerExists + "))"
			args = append(args, systemRole)
		} else {
			where += " AND u.system_role = ?"
			args = append(args, systemRole)
		}
	}
	if orgID != "" {
		where += " AND EXISTS (SELECT 1 FROM memberships om WHERE om.scope_type = 'org' AND om.user_id = u.id AND om.scope_id = ?)"
		args = append(args, orgID)
	}
	if role != "" {
		where += " AND EXISTS (" + roleExists + ")"
		args = append(args, role)
	}

	var total int64
	if err := h.DB.Raw("SELECT COUNT(*) FROM users u "+where, args...).Scan(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count users"})
		return
	}

	query := `
		SELECT u.id, u.email, u.status,
		       CASE
		         WHEN u.system_role = 'none' AND EXISTS (` + ownerExists + `) THEN 'org_admin'
		         ELSE u.system_role
		       END AS system_role,
		       u.created_at,
		       COALESCE((SELECT o.name FROM memberships om` + orgInfoJoin + ` WHERE om.scope_type = 'org' AND om.user_id = u.id` + orgTypeFilter + ` ORDER BY om.created_at ASC LIMIT 1), '') AS org_name,
		       COALESCE((SELECT om.role FROM memberships om` + orgInfoJoin + ` WHERE om.scope_type = 'org' AND om.user_id = u.id` + orgTypeFilter + ` ORDER BY om.created_at ASC LIMIT 1), '') AS org_role,
		       (SELECT COUNT(*) FROM memberships om` + orgInfoJoin + ` WHERE om.scope_type = 'org' AND om.user_id = u.id` + orgTypeFilter + `) AS org_count
		FROM users u
	` + where + `
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	var items []systemUserItem
	if err := h.DB.Raw(query, args...).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) DisableSystemUser(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	if _, err := uuid.Parse(targetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	currentUserID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if currentUserID != "" && currentUserID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable current user"})
		return
	}

	var user models.User
	if err := h.DB.Where("id = ?", targetID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if strings.ToLower(strings.TrimSpace(user.SystemRole)) == "system_admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable system admin"})
		return
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", targetID).Update("status", "disabled").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) EnableSystemUser(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	if _, err := uuid.Parse(targetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var user models.User
	if err := h.DB.Where("id = ?", targetID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", targetID).Update("status", "active").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ResetSystemUserPassword(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	if _, err := uuid.Parse(targetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req resetSystemUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
		return
	}

	var user models.User
	if err := h.DB.Where("id = ?", targetID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", targetID).Update("password_hash", hash).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateSystemUser(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	if _, err := uuid.Parse(targetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req updateSystemUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Email == nil && req.Password == nil && req.Status == nil && req.SystemRole == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}

	var user models.User
	if err := h.DB.Where("id = ?", targetID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
		return
	}

	currentUserID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	isCurrentUser := currentUserID != "" && currentUserID == targetID
	isSystemAdmin := strings.ToLower(strings.TrimSpace(user.SystemRole)) == "system_admin"

	updates := map[string]any{}

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		updates["email"] = email
	}

	if req.Password != nil {
		password := strings.TrimSpace(*req.Password)
		if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
			return
		}
		hash, err := utils.HashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		updates["password_hash"] = hash
	}

	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status required"})
			return
		}
		if status != "active" && status != "pending" && status != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		if isCurrentUser && status == "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable current user"})
			return
		}
		if isSystemAdmin && status != strings.ToLower(strings.TrimSpace(user.Status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot change system admin status"})
			return
		}
		updates["status"] = status
	}

	if req.SystemRole != nil {
		role := strings.ToLower(strings.TrimSpace(*req.SystemRole))
		if role == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "system_role required"})
			return
		}
		if role != "system_admin" && role != "org_admin" && role != "none" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid system_role"})
			return
		}
		if isCurrentUser && role != strings.ToLower(strings.TrimSpace(user.SystemRole)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot change current user role"})
			return
		}
		if isSystemAdmin && role != strings.ToLower(strings.TrimSpace(user.SystemRole)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot change system admin role"})
			return
		}
		updates["system_role"] = role
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
		return
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", targetID).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemUsers(c *gin.Context) {
	var req batchDeleteSystemUsersRequest
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
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

	currentUserID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if currentUserID != "" {
		for _, id := range ids {
			if id == currentUserID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current user"})
				return
			}
		}
	}

	var users []models.User
	if err := h.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load users"})
		return
	}
	if len(users) == 0 || len(users) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "users not found"})
		return
	}
	for _, user := range users {
		if strings.ToLower(strings.TrimSpace(user.SystemRole)) == "system_admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete system admin"})
			return
		}
	}

	hasOrgTypeColumn := h.hasOrgTypeColumn()
	personalOrgIDs := make([]string, 0)
	if hasOrgTypeColumn {
		if err := h.DB.Model(&models.Org{}).
			Where("created_by IN ? AND org_type = ?", ids, "personal").
			Pluck("id", &personalOrgIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal orgs"})
			return
		}
	}

	var orgCount int64
	orgCreatorQuery := h.DB.Model(&models.Org{}).Where("created_by IN ?", ids)
	if hasOrgTypeColumn {
		orgCreatorQuery = orgCreatorQuery.Where("org_type = ?", "enterprise")
	}
	if err := orgCreatorQuery.Count(&orgCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check org creators"})
		return
	}
	if orgCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete org creator"})
		return
	}

	var ownerCount int64
	ownerQuery := h.DB.Model(&models.OrgMember{}).Where("scope_type = ? AND user_id IN ? AND role = ?", models.ScopeOrg, ids, "owner")
	if hasOrgTypeColumn {
		ownerQuery = ownerQuery.Joins("JOIN orgs o ON o.id = memberships.scope_id").Where("o.org_type = ?", "enterprise")
	}
	if err := ownerQuery.Count(&ownerCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check org owners"})
		return
	}
	if ownerCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete org owner"})
		return
	}

	var ticketIDs []string
	if err := h.DB.Model(&models.Ticket{}).Where("created_by IN ? OR assignee_user_id IN ?", ids, ids).Pluck("id", &ticketIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}

	var ticketAttachmentPaths []string
	if len(ticketIDs) > 0 {
		var err error
		ticketAttachmentPaths, err = loadAttachmentStoragePaths(h.DB, attachmentOwnerTicket, ticketIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket attachments"})
			return
		}
	}

	var messageIDs []string
	messageQuery := h.DB.Model(&models.TicketMessage{})
	if len(ticketIDs) > 0 {
		messageQuery = messageQuery.Where("ticket_id IN ?", ticketIDs).Or("user_id IN ?", ids)
	} else {
		messageQuery = messageQuery.Where("user_id IN ?", ids)
	}
	if err := messageQuery.Pluck("id", &messageIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket messages"})
		return
	}

	var messageAttachmentPaths []string
	if len(messageIDs) > 0 {
		var err error
		messageAttachmentPaths, err = loadAttachmentStoragePaths(h.DB, attachmentOwnerTicketMessage, messageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load message attachments"})
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, orgID := range personalOrgIDs {
			if err := deleteOrgCascade(tx, orgID); err != nil {
				return err
			}
		}
		if len(messageIDs) > 0 {
			if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicketMessage, messageIDs); err != nil {
				return err
			}
			if err := tx.Where("id IN ?", messageIDs).Delete(&models.TicketMessage{}).Error; err != nil {
				return err
			}
		}
		if len(ticketIDs) > 0 {
			if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicket, ticketIDs); err != nil {
				return err
			}
			if err := tx.Where("id IN ?", ticketIDs).Delete(&models.Ticket{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("scope_type = ? AND user_id IN ?", models.ScopeApp, ids).Delete(&models.AppMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("scope_type = ? AND user_id IN ?", models.ScopeOrg, ids).Delete(&models.OrgMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("created_by IN ?", ids).Delete(&models.OrgInvite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", ids).Delete(&models.AuditLog{}).Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable(&models.OrgJoinRequest{}) {
			if err := tx.Where("user_id IN ? OR reviewed_by IN ?", ids, ids).Delete(&models.OrgJoinRequest{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", ids).Delete(&models.User{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete users"})
		return
	}

	paths := append(ticketAttachmentPaths, messageAttachmentPaths...)
	h.deleteStoragePaths(c, paths)
	if len(ticketIDs) > 0 {
		uniqueTickets := make(map[string]struct{}, len(ticketIDs))
		for _, ticketID := range ticketIDs {
			if ticketID == "" {
				continue
			}
			if _, ok := uniqueTickets[ticketID]; ok {
				continue
			}
			uniqueTickets[ticketID] = struct{}{}
			h.deleteLocalTicketDir(ticketID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"deleted": len(ids)})
}

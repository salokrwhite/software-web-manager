package org

import (
	"errors"
	"net/http"
	"software-web-manager/backend/internal/api/common"
	orgsvc "software-web-manager/backend/internal/services/org"
	"strings"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOrgInviteRequest struct {
	Email         string `json:"email" binding:"required,email"`
	Role          string `json:"role" binding:"required"`
	ExpiresInDays int    `json:"expires_in_days"`
}

type acceptOrgInviteRequest struct {
	Password string `json:"password"`
}

func (h *Handler) CreateOrgInvite(c *gin.Context) {
	if !common.RequirePermission(c, "member_invite.manage") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req createOrgInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if !orgsvc.NewService(h.DB).IsAssignableOrgRole(orgID, req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	if req.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use transfer owner to assign owner role"})
		return
	}
	if req.ExpiresInDays <= 0 {
		req.ExpiresInDays = 7
	}
	if req.ExpiresInDays > 365 {
		req.ExpiresInDays = 365
	}
	var existingUser models.User
	if err := h.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		var member models.OrgMember
		if err := h.DB.Where("scope_id = ? AND user_id = ?", orgID, existingUser.ID).First(&member).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "user already in org"})
			return
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	creatorID, err := uuid.Parse(c.GetString(middleware.ContextUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rawToken, err := token.RandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	invite := models.OrgInvite{
		OrgID:     orgUUID,
		Email:     req.Email,
		Role:      req.Role,
		TokenHash: crypto.SHA256Hex([]byte(rawToken)),
		ExpiresAt: parseExpiresAt(req.ExpiresInDays),
		CreatedBy: creatorID,
	}
	if err := h.DB.Create(&invite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
		return
	}
	inviteLink := h.buildInviteLink(c, rawToken)
	common.Audit(h.DB, c, "org_invite.create", "org_invite", invite.ID, nil, invite)
	c.JSON(http.StatusOK, gin.H{
		"invite_id":   invite.ID,
		"invite_link": inviteLink,
		"expires_at":  invite.ExpiresAt,
	})
}

func (h *Handler) ListOrgInvites(c *gin.Context) {
	if !common.RequirePermission(c, "member_invite.manage") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var invites []models.OrgInvite
	if err := h.DB.Where("org_id = ?", orgID).Order("created_at desc").Find(&invites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invites"})
		return
	}
	now := time.Now()
	items := make([]gin.H, 0, len(invites))
	for _, invite := range invites {
		status := "active"
		if invite.RevokedAt != nil {
			status = "revoked"
		} else if invite.UsedAt != nil {
			status = "used"
		} else if invite.ExpiresAt != nil && invite.ExpiresAt.Before(now) {
			status = "expired"
		}
		items = append(items, gin.H{
			"id":         invite.ID,
			"email":      invite.Email,
			"role":       invite.Role,
			"status":     status,
			"created_at": invite.CreatedAt,
			"expires_at": invite.ExpiresAt,
			"created_by": invite.CreatedBy,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) RevokeOrgInvite(c *gin.Context) {
	if !common.RequirePermission(c, "member_invite.manage") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	inviteID := c.Param("invite_id")
	now := time.Now()
	res := h.DB.Model(&models.OrgInvite{}).
		Where("id = ? AND org_id = ? AND revoked_at IS NULL AND used_at IS NULL", inviteID, orgID).
		Update("revoked_at", now)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke invite"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invite not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type batchDeleteOrgInvitesRequest struct {
	InviteIDs []string `json:"invite_ids" binding:"required"`
}

func (h *Handler) BatchDeleteOrgInvites(c *gin.Context) {
	if !common.RequirePermission(c, "member_invite.manage") {
		return
	}
	orgID := c.Param("id")
	if orgID != c.GetString(middleware.ContextOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req batchDeleteOrgInvitesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.InviteIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite_ids required"})
		return
	}
	ids := make([]uuid.UUID, 0, len(req.InviteIDs))
	for _, raw := range req.InviteIDs {
		parsed, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
			return
		}
		ids = append(ids, parsed)
	}
	res := h.DB.Where("org_id = ? AND id IN ?", orgID, ids).Delete(&models.OrgInvite{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete invites"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

func (h *Handler) AcceptOrgInvite(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}
	var req acceptOrgInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Password = strings.TrimSpace(req.Password)
	var invite models.OrgInvite
	if err := h.DB.Where("token_hash = ?", crypto.SHA256Hex([]byte(token))).First(&invite).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid invite"})
		return
	}
	if invite.RevokedAt != nil || invite.UsedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite not available"})
		return
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite expired"})
		return
	}
	var org models.Org
	orgType := ""
	if err := h.DB.Where("id = ?", invite.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": middleware.OrgStatusCode(org.Status)})
			return
		}
		orgType = strings.TrimSpace(org.OrgType)
	}
	var user models.User
	if err := h.DB.Where("email = ?", invite.Email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
			return
		}
		if req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
			return
		}
		if len(req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password too short"})
			return
		}
		hash, err := crypto.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		user = models.User{Email: invite.Email, PasswordHash: hash, Status: "active", SystemRole: "none"}
		if err := h.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	} else if req.Password != "" && !crypto.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	} else if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": middleware.UserStatusCode(user.Status)})
		return
	}
	var member models.OrgMember
	memberErr := h.DB.Where("scope_id = ? AND user_id = ?", invite.OrgID, user.ID).First(&member).Error
	if memberErr != nil {
		if !errors.Is(memberErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query member"})
			return
		}
		member = models.OrgMember{OrgID: invite.OrgID, UserID: user.ID, Role: invite.Role}
		if err := h.DB.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
			return
		}
	}
	usedAt := time.Now()
	_ = h.DB.Model(&models.OrgInvite{}).
		Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", invite.ID).
		Update("used_at", usedAt).Error
	memberRole := member.Role
	if memberRole == "" {
		memberRole = invite.Role
	}
	effectiveRole := orgsvc.NewService(h.DB).ResolveEffectiveOrgRole(invite.OrgID.String(), memberRole)
	systemRole, err := orgsvc.NewService(h.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user role"})
		return
	}
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), invite.OrgID.String(), effectiveRole, systemRole, user.TokenVersion, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": user.ID, "email": user.Email},
		"org_id":      invite.OrgID,
		"role":        effectiveRole,
		"system_role": systemRole,
		"org_type":    orgType,
		"tokens":      tokens,
	})
}

func (h *Handler) GetOrgInvitePublic(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}
	var invite models.OrgInvite
	if err := h.DB.Where("token_hash = ?", crypto.SHA256Hex([]byte(token))).First(&invite).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid invite"})
		return
	}
	if invite.RevokedAt != nil || invite.UsedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite not available"})
		return
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite expired"})
		return
	}
	var org models.Org
	orgType := ""
	orgName := ""
	if err := h.DB.Where("id = ?", invite.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": middleware.OrgStatusCode(org.Status)})
			return
		}
		orgType = strings.TrimSpace(org.OrgType)
		orgName = org.Name
	}
	c.JSON(http.StatusOK, gin.H{
		"org_id":     invite.OrgID,
		"org_name":   orgName,
		"org_type":   orgType,
		"role":       invite.Role,
		"expires_at": invite.ExpiresAt,
	})
}

func (h *Handler) ListUserOrgInvites(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	type inviteRow struct {
		ID        uuid.UUID
		OrgID     uuid.UUID
		OrgName   string
		Role      string
		CreatedAt time.Time
		ExpiresAt *time.Time
		UsedAt    *time.Time
		RevokedAt *time.Time
	}
	var rows []inviteRow
	if err := h.DB.Raw(`
		SELECT i.id, i.org_id, o.name as org_name, i.role, i.created_at, i.expires_at, i.used_at, i.revoked_at
		FROM org_invites i
		JOIN orgs o ON o.id = i.org_id
		WHERE i.email = ?
		ORDER BY i.created_at DESC
	`, strings.ToLower(strings.TrimSpace(user.Email))).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invites"})
		return
	}
	now := time.Now()
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		status := "active"
		if row.RevokedAt != nil {
			status = "revoked"
		} else if row.UsedAt != nil {
			status = "used"
		} else if row.ExpiresAt != nil && row.ExpiresAt.Before(now) {
			status = "expired"
		}
		items = append(items, gin.H{
			"id":         row.ID,
			"org_id":     row.OrgID,
			"org_name":   row.OrgName,
			"role":       row.Role,
			"status":     status,
			"created_at": row.CreatedAt,
			"expires_at": row.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AcceptOrgInviteByID(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	inviteID := strings.TrimSpace(c.Param("invite_id"))
	if inviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite_id required"})
		return
	}
	var invite models.OrgInvite
	if err := h.DB.Where("id = ? AND email = ?", inviteID, strings.ToLower(strings.TrimSpace(user.Email))).First(&invite).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invite not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invite"})
		return
	}
	if invite.RevokedAt != nil || invite.UsedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite not available"})
		return
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite expired"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", invite.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": middleware.OrgStatusCode(org.Status)})
			return
		}
	}

	var member models.OrgMember
	memberErr := h.DB.Where("scope_id = ? AND user_id = ?", invite.OrgID, user.ID).First(&member).Error
	if memberErr != nil {
		if !errors.Is(memberErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query member"})
			return
		}
		member = models.OrgMember{OrgID: invite.OrgID, UserID: user.ID, Role: invite.Role}
		if err := h.DB.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
			return
		}
	}
	usedAt := time.Now()
	_ = h.DB.Model(&models.OrgInvite{}).
		Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", invite.ID).
		Update("used_at", usedAt).Error

	memberRole := member.Role
	if memberRole == "" {
		memberRole = invite.Role
	}
	effectiveRole := orgsvc.NewService(h.DB).ResolveEffectiveOrgRole(invite.OrgID.String(), memberRole)
	systemRole, err := orgsvc.NewService(h.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user role"})
		return
	}
	orgType := ""
	if strings.TrimSpace(org.OrgType) != "" {
		orgType = strings.TrimSpace(org.OrgType)
	}
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), invite.OrgID.String(), effectiveRole, systemRole, user.TokenVersion, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": user.ID, "email": user.Email},
		"org_id":      invite.OrgID,
		"role":        effectiveRole,
		"system_role": systemRole,
		"org_type":    orgType,
		"tokens":      tokens,
	})
}

func (h *Handler) buildInviteLink(c *gin.Context, token string) string {
	base := strings.TrimSpace(h.Cfg.WebBaseURL)
	if base == "" {
		if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
			base = origin
		} else if host := strings.TrimSpace(c.Request.Host); host != "" {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			base = scheme + "://" + host
		}
	}
	if base == "" {
		return "/invite/" + token
	}
	return strings.TrimRight(base, "/") + "/invite/" + token
}

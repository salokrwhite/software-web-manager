package auth

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/handlers"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type registerRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	EmailCode string `json:"email_code"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type adminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	OTPCode  string `json:"otp_code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	allowRegister, err := h.AllowUserRegisterEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	if !allowRegister {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_register_disabled"})
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var existingCount int64
	if err := h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&existingCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if err := h.ensureEmailVerificationCodesTable(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize email verification table"})
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var user models.User
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.consumeRegisterEmailCode(tx, req.Email, req.EmailCode); err != nil {
			return err
		}
		user = models.User{
			Email:        req.Email,
			PasswordHash: hash,
			Status:       "active",
			SystemRole:   "none",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errRegisterEmailCodeRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "email_code_required"})
			return
		case errors.Is(err, errRegisterEmailCodeInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "email_code_invalid"})
			return
		case errors.Is(err, errRegisterEmailCodeExpired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "email_code_expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{"id": user.ID, "email": user.Email},
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !crypto.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": handlers.UserStatusCode(user.Status)})
		return
	}

	systemRole, err := h.ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user role"})
		return
	}
	if systemRole == "org_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin login required", "code": "admin_login_required"})
		return
	}
	if systemRole == "system_admin" {
		tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), "", "", systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":        gin.H{"id": user.ID, "email": user.Email},
			"org_id":      "",
			"role":        "",
			"system_role": systemRole,
			"org_type":    "",
			"tokens":      tokens,
		})
		return
	}

	orgType := ""
	hasOrgTypeColumn := h.HasOrgTypeColumn()
	if hasOrgTypeColumn {
		personalOrg, personalMember, err := h.EnsurePersonalOrgMember(user.ID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure personal org"})
			return
		}
		if personalOrg.ID != (uuid.UUID{}) {
			if strings.ToLower(strings.TrimSpace(personalOrg.Status)) != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": handlers.OrgStatusCode(personalOrg.Status)})
				return
			}
			effectiveRole := h.ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
			tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), personalMember.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"user":        gin.H{"id": user.ID, "email": user.Email},
				"org_id":      personalMember.OrgID,
				"role":        effectiveRole,
				"system_role": systemRole,
				"org_type":    personalOrg.OrgType,
				"tokens":      tokens,
			})
			return
		}
	}
	var member models.OrgMember
	if err := h.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var org models.Org
			query := h.DB.Where("created_by = ?", user.ID)
			if hasOrgTypeColumn {
				query = query.Where("org_type = ?", "personal")
			}
			err = query.
				Order("created_at desc").
				First(&org).
				Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query personal org"})
					return
				}
				org = models.Org{
					Name:      "个人空间",
					Plan:      "free",
					Status:    "active",
					CreatedBy: user.ID,
				}
				if hasOrgTypeColumn {
					org.OrgType = "personal"
				}
				member = models.OrgMember{OrgID: org.ID, UserID: user.ID, Role: "owner"}
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
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create personal org"})
					return
				}
			} else {
				orgType = org.OrgType
				var existing models.OrgMember
				if err := h.DB.Where("scope_id = ? AND user_id = ?", org.ID, user.ID).First(&existing).Error; err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
						return
					}
					member = models.OrgMember{OrgID: org.ID, UserID: user.ID, Role: "owner"}
					if err := h.DB.Create(&member).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create org membership"})
						return
					}
				} else {
					member = existing
				}
			}
			if orgType == "" {
				orgType = "personal"
			}
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": handlers.OrgStatusCode(org.Status)})
				return
			}
			effectiveRole := h.ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
			tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"user":        gin.H{"id": user.ID, "email": user.Email},
				"org_id":      member.OrgID,
				"role":        effectiveRole,
				"system_role": systemRole,
				"org_type":    orgType,
				"tokens":      tokens,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": handlers.OrgStatusCode(org.Status)})
			return
		}
		orgType = org.OrgType
	}

	effectiveRole := h.ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": user.ID, "email": user.Email},
		"org_id":      member.OrgID,
		"role":        effectiveRole,
		"system_role": systemRole,
		"org_type":    orgType,
		"tokens":      tokens,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims, err := auth.ParseToken(h.Cfg.JWTSecret, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	var user models.User
	if err := h.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": handlers.UserStatusCode(user.Status)})
		return
	}
	systemRole, err := h.ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user role"})
		return
	}
	orgID := strings.TrimSpace(claims.OrgID)
	role := strings.TrimSpace(claims.Role)
	orgType := ""
	if orgID != "" && systemRole != "system_admin" {
		member, err := h.GetOrgMember(orgID, user.ID.String())
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if h.HasOrgTypeColumn() {
					personalOrg, personalMember, err := h.EnsurePersonalOrgMember(user.ID.String())
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load personal org"})
						return
					}
					if personalOrg.ID != (uuid.UUID{}) {
						orgID = personalMember.OrgID.String()
						role = h.ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
						orgType = personalOrg.OrgType
					} else {
						orgID = ""
						role = ""
						orgType = ""
					}
				} else {
					orgID = ""
					role = ""
					orgType = ""
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
				return
			}
		} else {
			role = h.ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		}
	}

	if orgID != "" {
		var org models.Org
		if err := h.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": handlers.OrgStatusCode(org.Status)})
				return
			}
			if h.HasOrgTypeColumn() {
				orgType = org.OrgType
			}
		}
	}

	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, claims.UserID, orgID, role, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tokens":      tokens,
		"org_id":      orgID,
		"role":        role,
		"org_type":    orgType,
		"system_role": systemRole,
	})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !crypto.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": handlers.UserStatusCode(user.Status)})
		return
	}
	systemRole, err := h.ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user role"})
		return
	}
	if systemRole != "system_admin" && systemRole != "org_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}

	if systemRole == "org_admin" && user.OTPEnabled {
		if strings.TrimSpace(req.OTPCode) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "otp required", "code": "otp_required"})
			return
		}
		secret := ""
		if user.OTPSecret != nil {
			secret = strings.TrimSpace(*user.OTPSecret)
		}
		if !handlers.ValidateTOTP(secret, req.OTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid otp", "code": "otp_invalid"})
			return
		}
	}

	if systemRole == "org_admin" {
		var member models.OrgMember
		memberLoaded := false
		if h.HasOrgTypeColumn() {
			if err := h.DB.Raw(`
				SELECT m.scope_id, m.user_id, m.role, m.created_at
				FROM memberships m
				JOIN orgs o ON o.id = m.scope_id
				WHERE m.scope_type = 'org' AND m.user_id = ? AND COALESCE(o.org_type, '') <> 'personal'
				ORDER BY o.created_at DESC
				LIMIT 1
			`, user.ID).Scan(&member).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
				return
			}
			if member.OrgID != (uuid.UUID{}) {
				memberLoaded = true
			}
		}
		if !memberLoaded {
			if err := h.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusForbidden, gin.H{"error": "user has no org"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query org membership"})
				return
			}
		}
		var org models.Org
		if err := h.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": handlers.OrgStatusCode(org.Status)})
				return
			}
		}
		orgType := org.OrgType
		effectiveRole := h.ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), member.OrgID.String(), effectiveRole, systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":        gin.H{"id": user.ID, "email": user.Email},
			"org_id":      member.OrgID,
			"role":        effectiveRole,
			"system_role": systemRole,
			"org_type":    orgType,
			"tokens":      tokens,
		})
		return
	}

	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), "", "owner", systemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": user.ID, "email": user.Email},
		"system_role": systemRole,
		"org_type":    "",
		"tokens":      tokens,
	})
}

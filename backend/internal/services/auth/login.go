package auth

import (
	"errors"
	"strings"

	authcore "software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	orgsvc "software-web-manager/backend/internal/services/org"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginResult is the resolved identity and issued tokens for a successful login
// or token refresh.
type LoginResult struct {
	User       models.User
	OrgID      string
	Role       string
	SystemRole string
	OrgType    string
	Tokens     authcore.TokenPair
}

func (s *Service) issueTokens(userID, orgID, role, systemRole string) (authcore.TokenPair, error) {
	return authcore.IssueTokens(s.Cfg.JWTSecret, s.Cfg.JWTIssuer, userID, orgID, role, systemRole, s.Cfg.AccessTokenMinutes, s.Cfg.RefreshTokenHours)
}

// Login authenticates a user by email/password and resolves their org context.
func (s *Service) Login(email, password string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, newError(401, "invalid credentials")
	}
	if !crypto.CheckPassword(user.PasswordHash, password) {
		return nil, newError(401, "invalid credentials")
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		return nil, &UserNotActiveError{Status: user.Status}
	}

	systemRole, err := orgsvc.NewService(s.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		return nil, newError(500, "failed to resolve user role")
	}
	if systemRole == "org_admin" {
		return nil, newErrorCode(403, "admin login required", "admin_login_required")
	}
	if systemRole == "system_admin" {
		tokens, err := s.issueTokens(user.ID.String(), "", "", systemRole)
		if err != nil {
			return nil, newError(500, "failed to issue token")
		}
		return &LoginResult{User: user, OrgID: "", Role: "", SystemRole: systemRole, OrgType: "", Tokens: tokens}, nil
	}

	orgType := ""
	hasOrgTypeColumn := schema.HasOrgTypeColumn(s.DB)
	if hasOrgTypeColumn {
		personalOrg, personalMember, err := orgsvc.NewService(s.DB).EnsurePersonalMember(user.ID.String())
		if err != nil {
			return nil, newError(500, "failed to ensure personal org")
		}
		if personalOrg.ID != (uuid.UUID{}) {
			if strings.ToLower(strings.TrimSpace(personalOrg.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: personalOrg.Status}
			}
			effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
			tokens, err := s.issueTokens(user.ID.String(), personalMember.OrgID.String(), effectiveRole, systemRole)
			if err != nil {
				return nil, newError(500, "failed to issue token")
			}
			return &LoginResult{User: user, OrgID: personalMember.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: personalOrg.OrgType, Tokens: tokens}, nil
		}
	}

	var member models.OrgMember
	if err := s.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var org models.Org
			query := s.DB.Where("created_by = ?", user.ID)
			if hasOrgTypeColumn {
				query = query.Where("org_type = ?", "personal")
			}
			err = query.
				Order("created_at desc").
				First(&org).
				Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, newError(500, "failed to query personal org")
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
				if err := s.DB.Transaction(func(tx *gorm.DB) error {
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
					return nil, newError(500, "failed to create personal org")
				}
			} else {
				orgType = org.OrgType
				var existing models.OrgMember
				if err := s.DB.Where("scope_id = ? AND user_id = ?", org.ID, user.ID).First(&existing).Error; err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, newError(500, "failed to query org membership")
					}
					member = models.OrgMember{OrgID: org.ID, UserID: user.ID, Role: "owner"}
					if err := s.DB.Create(&member).Error; err != nil {
						return nil, newError(500, "failed to create org membership")
					}
				} else {
					member = existing
				}
			}
			if orgType == "" {
				orgType = "personal"
			}
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: org.Status}
			}
			effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
			tokens, err := s.issueTokens(user.ID.String(), member.OrgID.String(), effectiveRole, systemRole)
			if err != nil {
				return nil, newError(500, "failed to issue token")
			}
			return &LoginResult{User: user, OrgID: member.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: orgType, Tokens: tokens}, nil
		}
		return nil, newError(500, "failed to query org membership")
	}

	var org models.Org
	if err := s.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
			return nil, &OrgNotActiveError{Status: org.Status}
		}
		orgType = org.OrgType
	}

	effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
	tokens, err := s.issueTokens(user.ID.String(), member.OrgID.String(), effectiveRole, systemRole)
	if err != nil {
		return nil, newError(500, "failed to issue token")
	}
	return &LoginResult{User: user, OrgID: member.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: orgType, Tokens: tokens}, nil
}

// Refresh validates a refresh token and re-resolves the org context, issuing a
// fresh token pair.
func (s *Service) Refresh(refreshToken string) (*LoginResult, error) {
	claims, err := authcore.ParseToken(s.Cfg.JWTSecret, refreshToken)
	if err != nil {
		return nil, newError(401, "invalid refresh token")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, newError(401, "invalid refresh token")
	}
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, newError(401, "invalid refresh token")
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		return nil, &UserNotActiveError{Status: user.Status}
	}
	systemRole, err := orgsvc.NewService(s.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		return nil, newError(500, "failed to resolve user role")
	}
	orgID := strings.TrimSpace(claims.OrgID)
	role := strings.TrimSpace(claims.Role)
	orgType := ""
	if orgID != "" && systemRole != "system_admin" {
		member, err := orgsvc.NewService(s.DB).GetMember(orgID, user.ID.String())
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if schema.HasOrgTypeColumn(s.DB) {
					personalOrg, personalMember, err := orgsvc.NewService(s.DB).EnsurePersonalMember(user.ID.String())
					if err != nil {
						return nil, newError(500, "failed to load personal org")
					}
					if personalOrg.ID != (uuid.UUID{}) {
						orgID = personalMember.OrgID.String()
						role = orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(personalMember.OrgID.String(), personalMember.Role)
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
				return nil, newError(500, "failed to query org membership")
			}
		} else {
			role = orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		}
	}

	if orgID != "" {
		var org models.Org
		if err := s.DB.Where("id = ?", orgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: org.Status}
			}
			if schema.HasOrgTypeColumn(s.DB) {
				orgType = org.OrgType
			}
		}
	}

	tokens, err := s.issueTokens(claims.UserID, orgID, role, systemRole)
	if err != nil {
		return nil, newError(500, "failed to issue token")
	}
	return &LoginResult{User: user, OrgID: orgID, Role: role, SystemRole: systemRole, OrgType: orgType, Tokens: tokens}, nil
}

// AdminLogin authenticates an admin (system_admin or org_admin), enforcing OTP
// for OTP-enabled org admins.
func (s *Service) AdminLogin(email, password, otpCode string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, newError(401, "invalid credentials")
	}
	if !crypto.CheckPassword(user.PasswordHash, password) {
		return nil, newError(401, "invalid credentials")
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != "active" {
		return nil, &UserNotActiveError{Status: user.Status}
	}
	systemRole, err := orgsvc.NewService(s.DB).ResolveSystemRole(user.ID.String(), user.SystemRole)
	if err != nil {
		return nil, newError(500, "failed to resolve user role")
	}
	if systemRole != "system_admin" && systemRole != "org_admin" {
		return nil, newError(403, "insufficient role")
	}

	if systemRole == "org_admin" && user.OTPEnabled {
		if strings.TrimSpace(otpCode) == "" {
			return nil, newErrorCode(401, "otp required", "otp_required")
		}
		secret := ""
		if user.OTPSecret != nil {
			secret = strings.TrimSpace(*user.OTPSecret)
		}
		if !crypto.ValidateTOTP(secret, otpCode) {
			return nil, newErrorCode(401, "invalid otp", "otp_invalid")
		}
	}

	if systemRole == "org_admin" {
		var member models.OrgMember
		memberLoaded := false
		if schema.HasOrgTypeColumn(s.DB) {
			if err := s.DB.Raw(`
				SELECT m.scope_id, m.user_id, m.role, m.created_at
				FROM memberships m
				JOIN orgs o ON o.id = m.scope_id
				WHERE m.scope_type = 'org' AND m.user_id = ? AND COALESCE(o.org_type, '') <> 'personal'
				ORDER BY o.created_at DESC
				LIMIT 1
			`, user.ID).Scan(&member).Error; err != nil {
				return nil, newError(500, "failed to query org membership")
			}
			if member.OrgID != (uuid.UUID{}) {
				memberLoaded = true
			}
		}
		if !memberLoaded {
			if err := s.DB.Where("scope_type = ? AND user_id = ?", models.ScopeOrg, user.ID).First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, newError(403, "user has no org")
				}
				return nil, newError(500, "failed to query org membership")
			}
		}
		var org models.Org
		if err := s.DB.Where("id = ?", member.OrgID).First(&org).Error; err == nil {
			if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
				return nil, &OrgNotActiveError{Status: org.Status}
			}
		}
		orgType := org.OrgType
		effectiveRole := orgsvc.NewService(s.DB).ResolveEffectiveOrgRole(member.OrgID.String(), member.Role)
		tokens, err := s.issueTokens(user.ID.String(), member.OrgID.String(), effectiveRole, systemRole)
		if err != nil {
			return nil, newError(500, "failed to issue token")
		}
		return &LoginResult{User: user, OrgID: member.OrgID.String(), Role: effectiveRole, SystemRole: systemRole, OrgType: orgType, Tokens: tokens}, nil
	}

	tokens, err := s.issueTokens(user.ID.String(), "", "owner", systemRole)
	if err != nil {
		return nil, newError(500, "failed to issue token")
	}
	return &LoginResult{User: user, OrgID: "", Role: "owner", SystemRole: systemRole, OrgType: "", Tokens: tokens}, nil
}

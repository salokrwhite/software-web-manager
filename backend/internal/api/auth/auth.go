package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	authsvc "software-web-manager/backend/internal/services/auth"

	"github.com/gin-gonic/gin"
)

// Login brute-force throttle: after too many failed attempts for an
// (email, client IP) pair within the window, further attempts are rejected.
const (
	loginMaxFailures = 10
	loginFailWindow  = 15 * time.Minute
)

func loginAttemptKey(email, ip string) string {
	return "swm:login:fail:" + strings.ToLower(strings.TrimSpace(email)) + ":" + ip
}

// loginBlocked reports whether the (email, ip) pair is currently locked out.
// Fails open if the store is unavailable (availability over the rate limit).
func (h *Handler) loginBlocked(c *gin.Context, key string) bool {
	if h.ReplayStore == nil {
		return false
	}
	n, err := h.ReplayStore.Get(c.Request.Context(), key).Int()
	if err != nil {
		return false
	}
	return n >= loginMaxFailures
}

func (h *Handler) recordLoginFailure(c *gin.Context, key string) {
	if h.ReplayStore == nil {
		return
	}
	ctx := c.Request.Context()
	n, err := h.ReplayStore.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	if n == 1 {
		_ = h.ReplayStore.Expire(ctx, key, loginFailWindow).Err()
	}
}

func (h *Handler) clearLoginFailures(c *gin.Context, key string) {
	if h.ReplayStore == nil {
		return
	}
	_ = h.ReplayStore.Del(c.Request.Context(), key).Err()
}

// isCredentialFailure reports whether the auth error is a 401 (wrong
// password / OTP), which is what we count toward the lockout.
func isCredentialFailure(err error) bool {
	var ae *authsvc.Error
	if errors.As(err, &ae) {
		// "otp_required" is a normal intermediate step (prompt for the code), not a
		// failed attempt — don't count it toward the lockout.
		return ae.Status == http.StatusUnauthorized && ae.Code != "otp_required"
	}
	return false
}

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

// writeAuthError maps a service-layer auth error to its HTTP response.
func (h *Handler) writeAuthError(c *gin.Context, err error) {
	var unae *authsvc.UserNotActiveError
	if errors.As(err, &unae) {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not active", "code": middleware.UserStatusCode(unae.Status)})
		return
	}
	var onae *authsvc.OrgNotActiveError
	if errors.As(err, &onae) {
		c.JSON(http.StatusForbidden, gin.H{"error": "org not active", "code": middleware.OrgStatusCode(onae.Status)})
		return
	}
	var ae *authsvc.Error
	if errors.As(err, &ae) {
		body := gin.H{"error": ae.Message}
		if ae.Code != "" {
			body["code"] = ae.Code
		}
		c.JSON(ae.Status, body)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func (h *Handler) Register(c *gin.Context) {
	svc := authsvc.NewService(h.DB, h.Cfg)
	if err := svc.EnsureRegistrationAllowed(); err != nil {
		h.writeAuthError(c, err)
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := svc.RegisterUser(req.Email, req.Password, req.EmailCode)
	if err != nil {
		h.writeAuthError(c, err)
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

	key := loginAttemptKey(req.Email, c.ClientIP())
	if h.loginBlocked(c, key) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many failed attempts, try again later", "code": "too_many_attempts"})
		return
	}
	result, err := authsvc.NewService(h.DB, h.Cfg).Login(req.Email, req.Password)
	if err != nil {
		if isCredentialFailure(err) {
			h.recordLoginFailure(c, key)
		}
		h.writeAuthError(c, err)
		return
	}
	h.clearLoginFailures(c, key)
	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": result.User.ID, "email": result.User.Email},
		"org_id":      result.OrgID,
		"role":        result.Role,
		"system_role": result.SystemRole,
		"org_type":    result.OrgType,
		"tokens":      result.Tokens,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := authsvc.NewService(h.DB, h.Cfg).Refresh(req.RefreshToken)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tokens":      result.Tokens,
		"org_id":      result.OrgID,
		"role":        result.Role,
		"org_type":    result.OrgType,
		"system_role": result.SystemRole,
	})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := loginAttemptKey(req.Email, c.ClientIP())
	if h.loginBlocked(c, key) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many failed attempts, try again later", "code": "too_many_attempts"})
		return
	}
	result, err := authsvc.NewService(h.DB, h.Cfg).AdminLogin(req.Email, req.Password, req.OTPCode)
	if err != nil {
		if isCredentialFailure(err) {
			h.recordLoginFailure(c, key)
		}
		h.writeAuthError(c, err)
		return
	}
	h.clearLoginFailures(c, key)

	if result.SystemRole == "org_admin" {
		c.JSON(http.StatusOK, gin.H{
			"user":        gin.H{"id": result.User.ID, "email": result.User.Email},
			"org_id":      result.OrgID,
			"role":        result.Role,
			"system_role": result.SystemRole,
			"org_type":    result.OrgType,
			"tokens":      result.Tokens,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": result.User.ID, "email": result.User.Email},
		"system_role": result.SystemRole,
		"org_type":    "",
		"tokens":      result.Tokens,
	})
}

package auth

import (
	"errors"
	"net/http"

	"software-web-manager/backend/internal/middleware"
	authsvc "software-web-manager/backend/internal/services/auth"

	"github.com/gin-gonic/gin"
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

	result, err := authsvc.NewService(h.DB, h.Cfg).Login(req.Email, req.Password)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}
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

	result, err := authsvc.NewService(h.DB, h.Cfg).AdminLogin(req.Email, req.Password, req.OTPCode)
	if err != nil {
		h.writeAuthError(c, err)
		return
	}

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

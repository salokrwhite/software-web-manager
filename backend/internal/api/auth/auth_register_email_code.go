package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	authsvc "software-web-manager/backend/internal/services/auth"
	"software-web-manager/backend/internal/services/system"

	"github.com/gin-gonic/gin"
)

type sendRegisterEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func renderRegisterEmailCodeTemplate(content, siteName, code string, expiresMinutes int) string {
	template := strings.TrimSpace(content)
	if template == "" {
		template = system.DefaultRegisterEmailCodeTemplate
	}
	if strings.TrimSpace(siteName) == "" {
		siteName = system.DefaultSiteName
	}
	replacer := strings.NewReplacer(
		"{{code}}", code,
		"{{minutes}}", strconv.Itoa(expiresMinutes),
		"{{expire_minutes}}", strconv.Itoa(expiresMinutes),
		"{{site_name}}", siteName,
	)
	return replacer.Replace(template)
}

func (h *Handler) SendRegisterEmailCode(c *gin.Context) {
	svc := authsvc.NewService(h.DB, h.Cfg)
	if err := svc.EnsureRegistrationAllowed(); err != nil {
		h.writeAuthError(c, err)
		return
	}

	var req sendRegisterEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	registered, err := svc.EmailRegistered(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if registered {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	if err := svc.EnsureEmailVerificationTable(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize email verification table"})
		return
	}

	cfg, siteName, template, err := svc.RegisterEmailContext()
	if err != nil {
		if errors.Is(err, authsvc.ErrEmailNotConfigured) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_email_not_configured"})
			return
		}
		if authsvc.IsSMTPConfigValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}

	code, recordID, retryAfterSeconds, err := svc.CreateRegisterCode(email, strings.TrimSpace(c.ClientIP()))
	if err != nil {
		if errors.Is(err, authsvc.ErrEmailCodeTooFrequent) {
			if retryAfterSeconds < 1 {
				retryAfterSeconds = authsvc.RegisterEmailCodeCooldownSeconds
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               "email_code_send_too_frequent",
				"retry_after_seconds": retryAfterSeconds,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create email verification code"})
		return
	}

	subjectSiteName := strings.TrimSpace(siteName)
	if subjectSiteName == "" {
		subjectSiteName = system.DefaultSiteName
	}
	subject := fmt.Sprintf("%s 注册验证码", subjectSiteName)
	body := renderRegisterEmailCodeTemplate(template, siteName, code, authsvc.RegisterEmailCodeExpiresMinutes)
	if err := system.SendMail(cfg, email, subject, body); err != nil {
		svc.InvalidateCode(recordID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "register_email_send_failed",
			"detail": system.SanitizeSMTPError(err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":                 true,
		"expires_in_seconds": authsvc.RegisterEmailCodeExpiresMinutes * 60,
	})
}

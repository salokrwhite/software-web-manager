// Package install serves the first-run installation endpoints (status probe,
// database connectivity test, and the install bootstrap that runs migrations and
// creates the initial system admin).
package install

import (
	"context"
	"errors"
	"net/http"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/models"
	installsvc "software-web-manager/backend/internal/services/install"
	"software-web-manager/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

// Handler serves the installation endpoints over the shared core.
type Handler struct {
	*core.Handler
}

// New builds an install handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the install routes onto the public API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/install/status", h.GetInstallStatus)
	rg.POST("/install/test-db", h.TestDatabase)
	rg.POST("/install", h.Install)
}

type installStatusResponse struct {
	Installed bool `json:"installed"`
}

type testDbRequest struct {
	DbHost     string `json:"db_host" binding:"required"`
	DbPort     string `json:"db_port" binding:"required"`
	DbName     string `json:"db_name" binding:"required"`
	DbUser     string `json:"db_user" binding:"required"`
	DbPassword string `json:"db_password" binding:"required"`
}

type installRequest struct {
	DbHost     string `json:"db_host" binding:"required"`
	DbPort     string `json:"db_port" binding:"required"`
	DbName     string `json:"db_name" binding:"required"`
	DbUser     string `json:"db_user" binding:"required"`
	DbPassword string `json:"db_password" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	OrgName    string `json:"org_name"`
}

func (h *Handler) GetInstallStatus(c *gin.Context) {
	// 如果 DB 为 nil（安装模式），返回未安装
	if h.DB == nil {
		c.JSON(http.StatusOK, installStatusResponse{Installed: false})
		return
	}

	var count int64
	if err := h.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, installStatusResponse{Installed: false})
		return
	}
	c.JSON(http.StatusOK, installStatusResponse{Installed: count > 0})
}

func (h *Handler) TestDatabase(c *gin.Context) {
	var req testDbRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := installsvc.TestConnection(installsvc.ConnParams{
		DbHost:     req.DbHost,
		DbPort:     req.DbPort,
		DbName:     req.DbName,
		DbUser:     req.DbUser,
		DbPassword: req.DbPassword,
	}); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "数据库连接成功"})
}

func (h *Handler) Install(c *gin.Context) {
	var req installRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := installsvc.Run(installsvc.Params{
		ConnParams: installsvc.ConnParams{
			DbHost:     req.DbHost,
			DbPort:     req.DbPort,
			DbName:     req.DbName,
			DbUser:     req.DbUser,
			DbPassword: req.DbPassword,
		},
		Email:    req.Email,
		Password: req.Password,
		OrgName:  req.OrgName,
	})
	if err != nil {
		if errors.Is(err, installsvc.ErrAlreadyInstalled) {
			c.JSON(http.StatusForbidden, gin.H{"error": "system already installed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user := result.User
	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), "", "", user.SystemRole, user.TokenVersion, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	// 安装成功后，更新 Handler 的数据库连接
	// 这样后续请求就能正常使用数据库了
	h.DB = result.DB
	if h.Storage == nil {
		store, err := storage.New(context.Background(), h.Cfg)
		if err != nil && h.Cfg.StorageDriver != "local" {
			fallbackCfg := h.Cfg
			fallbackCfg.StorageDriver = "local"
			store, err = storage.New(context.Background(), fallbackCfg)
		}
		if err == nil {
			h.Storage = store
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        gin.H{"id": user.ID, "email": user.Email},
		"system_role": user.SystemRole,
		"tokens":      tokens,
	})
}

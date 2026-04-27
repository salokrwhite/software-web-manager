package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"software-web-manager/backend/internal/auth"
	dbpkg "software-web-manager/backend/internal/db"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/storage"
	"software-web-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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

func formatEnvValue(value string) string {
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, " \t#\"'") {
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return `"` + escaped + `"`
	}
	return value
}

func writeEnvConfig(updates map[string]string) error {
	candidates := []string{".env", "backend/.env"}
	var path string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		path = ".env"
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{}
	if len(content) > 0 {
		normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
		lines = strings.Split(normalized, "\n")
	}

	updated := make(map[string]bool, len(updates))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if val, ok := updates[key]; ok {
			lines[i] = key + "=" + formatEnvValue(val)
			updated[key] = true
		}
	}

	for key, val := range updates {
		if updated[key] {
			continue
		}
		lines = append(lines, key+"="+formatEnvValue(val))
	}

	output := strings.Join(lines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return os.WriteFile(path, []byte(output), 0o644)
}

func resolveMigrationsPath() (string, error) {
	candidates := []string{
		"./migrations",
		"backend/migrations",
		filepath.Join("..", "migrations"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("migrations directory not found")
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

	// 构建数据库连接字符串（先不指定数据库，用于测试连接和创建数据库）
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true&loc=Local",
		req.DbUser, req.DbPassword, req.DbHost, req.DbPort)

	// 测试连接
	db, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "无法连接到数据库，请检查主机、端口、用户名和密码"})
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "数据库连接失败"})
		return
	}
	defer sqlDB.Close()

	// 测试创建数据库权限
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", req.DbName)
	if err := db.Exec(createDBSQL).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "无法创建数据库，请检查用户权限"})
		return
	}

	// 测试连接指定数据库
	dsnWithDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		req.DbUser, req.DbPassword, req.DbHost, req.DbPort, req.DbName)
	db2, err := gorm.Open(mysql.Open(dsnWithDB), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "无法连接到指定数据库"})
		return
	}

	sqlDB2, err := db2.DB()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "数据库连接失败"})
		return
	}
	defer sqlDB2.Close()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "数据库连接成功"})
}

func (h *Handler) Install(c *gin.Context) {
	var req installRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		req.DbUser, req.DbPassword, req.DbHost, req.DbPort, req.DbName)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法连接到数据库: " + err.Error()})
		return
	}

	// 检查是否已安装
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		// 表不存在，继续安装
		count = 0
	}
	if count > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "system already installed"})
		return
	}

	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库迁移失败: " + err.Error()})
		return
	}

	if err := dbpkg.Migrate(db, migrationsPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库迁移失败: " + err.Error()})
		return
	}

	if err := db.AutoMigrate(&models.User{}, &models.SystemSetting{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库迁移失败: " + err.Error()})
		return
	}

	databaseURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		req.DbUser, req.DbPassword, req.DbHost, req.DbPort, req.DbName)
	if err := writeEnvConfig(map[string]string{
		"DATABASE_URL": databaseURL,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist database config: " + err.Error()})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var user models.User
	err = db.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:        req.Email,
			PasswordHash: hash,
			Status:       "active",
			SystemRole:   "system_admin",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to install"})
		return
	}

	tokens, err := auth.IssueTokens(h.Cfg.JWTSecret, h.Cfg.JWTIssuer, user.ID.String(), "", "", user.SystemRole, h.Cfg.AccessTokenMinutes, h.Cfg.RefreshTokenHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	// 安装成功后，更新 Handler 的数据库连接
	// 这样后续请求就能正常使用数据库了
	h.DB = db
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

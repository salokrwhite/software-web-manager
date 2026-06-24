// Package install provides the first-run installation logic (database
// connectivity test and install bootstrap) independent of the HTTP layer (no
// gin, no response writing).
package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"software-web-manager/backend/internal/crypto"
	dbpkg "software-web-manager/backend/internal/db"
	"software-web-manager/backend/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ErrAlreadyInstalled is returned by Run when the target database already has
// users (the system is installed).
var ErrAlreadyInstalled = errors.New("system already installed")

// ConnParams carries the database connection settings.
type ConnParams struct {
	DbHost     string
	DbPort     string
	DbName     string
	DbUser     string
	DbPassword string
}

// Params carries the full install bootstrap inputs.
type Params struct {
	ConnParams
	Email    string
	Password string
	OrgName  string
}

// Result is the outcome of a successful install.
type Result struct {
	DB   *gorm.DB
	User models.User
}

func dsnWithoutDB(p ConnParams) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true&loc=Local",
		p.DbUser, p.DbPassword, p.DbHost, p.DbPort)
}

func dsnWithDB(p ConnParams) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		p.DbUser, p.DbPassword, p.DbHost, p.DbPort, p.DbName)
}

// TestConnection verifies the database is reachable, can be created, and the
// target database is connectable. The returned error carries a user-facing
// message describing the failed stage.
func TestConnection(p ConnParams) error {
	db, err := gorm.Open(mysql.Open(dsnWithoutDB(p)), &gorm.Config{})
	if err != nil {
		return errors.New("无法连接到数据库，请检查主机、端口、用户名和密码")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("数据库连接失败")
	}
	defer sqlDB.Close()

	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", p.DbName)
	if err := db.Exec(createDBSQL).Error; err != nil {
		return errors.New("无法创建数据库，请检查用户权限")
	}

	db2, err := gorm.Open(mysql.Open(dsnWithDB(p)), &gorm.Config{})
	if err != nil {
		return errors.New("无法连接到指定数据库")
	}
	sqlDB2, err := db2.DB()
	if err != nil {
		return errors.New("数据库连接失败")
	}
	defer sqlDB2.Close()
	return nil
}

// Run performs the install: connect, ensure not already installed, run
// migrations, persist the database URL to the env file, and create the initial
// system admin. On failure it returns ErrAlreadyInstalled or an error whose
// message is suitable for direct display.
func Run(p Params) (*Result, error) {
	dsn := dsnWithDB(p.ConnParams)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		// 表不存在，继续安装
		count = 0
	}
	if count > 0 {
		return nil, ErrAlreadyInstalled
	}

	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := dbpkg.Migrate(db, migrationsPath); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SystemSetting{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	if err := writeEnvConfig(map[string]string{
		"DATABASE_URL": dsn,
	}); err != nil {
		return nil, fmt.Errorf("failed to persist database config: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(p.Email))
	hash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	var user models.User
	err = db.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:        email,
			PasswordHash: hash,
			Status:       "active",
			SystemRole:   "system_admin",
		}
		return tx.Create(&user).Error
	})
	if err != nil {
		return nil, errors.New("failed to install")
	}

	return &Result{DB: db, User: user}, nil
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

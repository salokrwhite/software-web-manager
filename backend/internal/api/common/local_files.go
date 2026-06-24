package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	localFileExpiryParam = "exp"
	localFileSigParam    = "sig"
	localFileDefaultTTL  = 24 * time.Hour
)

func localFileSigningKey(cfg config.Config) string {
	if strings.TrimSpace(cfg.AppSecretMasterKey) != "" {
		return cfg.AppSecretMasterKey
	}
	return cfg.JWTSecret
}

func normalizeLocalFilePath(storagePath string) string {
	raw := strings.ReplaceAll(strings.TrimSpace(storagePath), "\\", "/")
	for _, segment := range strings.Split(raw, "/") {
		if segment == ".." {
			return ""
		}
	}
	cleaned := path.Clean("/" + raw)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func signLocalFilePath(secret string, normalizedPath string, expiresAt int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(normalizedPath))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(strconv.FormatInt(expiresAt, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildLocalFileURL builds a signed, time-limited URL for serving a local file.
func BuildLocalFileURL(cfg config.Config, c *gin.Context, storagePath string, ttl time.Duration) string {
	normalizedPath := normalizeLocalFilePath(storagePath)
	if normalizedPath == "" {
		return ""
	}
	if ttl <= 0 {
		ttl = localFileDefaultTTL
	}
	expiresAt := time.Now().Add(ttl).Unix()
	sig := signLocalFilePath(localFileSigningKey(cfg), normalizedPath, expiresAt)
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		host = strings.TrimSpace(c.Request.URL.Host)
	}
	return fmt.Sprintf("%s://%s/files/%s?%s=%d&%s=%s", scheme, host, normalizedPath, localFileExpiryParam, expiresAt, localFileSigParam, sig)
}

func resolveLocalStoragePath(rootPath string, storagePath string) (string, error) {
	normalizedPath := normalizeLocalFilePath(storagePath)
	if normalizedPath == "" {
		return "", errors.New("invalid storage path")
	}
	rootAbs, err := filepath.Abs(strings.TrimSpace(rootPath))
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(normalizedPath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "", errors.New("invalid storage path")
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", errors.New("path escapes storage root")
	}
	return targetAbs, nil
}

// SanitizeUploadedFilename strips path components from an uploaded filename.
func SanitizeUploadedFilename(filename string) string {
	cleaned := strings.TrimSpace(filename)
	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	cleaned = path.Base(cleaned)
	cleaned = strings.TrimSpace(cleaned)
	switch cleaned {
	case "", ".", "/":
		return ""
	}
	return cleaned
}

// ServeLocalFile returns a gin handler that serves signed local files.
func ServeLocalFile(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(cfg.LocalStoragePath) == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "file storage not configured"})
			return
		}
		requestPath := strings.TrimPrefix(c.Param("filepath"), "/")
		normalizedPath := normalizeLocalFilePath(requestPath)
		if normalizedPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		expRaw := strings.TrimSpace(c.Query(localFileExpiryParam))
		sig := strings.ToLower(strings.TrimSpace(c.Query(localFileSigParam)))
		if expRaw == "" || sig == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "file token required"})
			return
		}
		expiresAt, err := strconv.ParseInt(expRaw, 10, 64)
		if err != nil || expiresAt <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid file token"})
			return
		}
		if time.Now().Unix() > expiresAt {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "file token expired"})
			return
		}
		expectedSig := signLocalFilePath(localFileSigningKey(cfg), normalizedPath, expiresAt)
		if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid file token"})
			return
		}
		fullPath, err := resolveLocalStoragePath(cfg.LocalStoragePath, normalizedPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.File(fullPath)
	}
}

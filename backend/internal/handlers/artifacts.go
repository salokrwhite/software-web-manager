package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) UploadArtifact(c *gin.Context) {
	releaseID := c.Param("id")
	userID := c.GetString(middleware.ContextUserID)
	orgID := c.GetString(middleware.ContextOrgID)
	platform := strings.TrimSpace(c.PostForm("platform"))
	arch := strings.TrimSpace(c.PostForm("arch"))
	fileType := strings.TrimSpace(c.PostForm("file_type"))
	signature := strings.TrimSpace(c.PostForm("signature"))
	replaceRaw := strings.TrimSpace(c.PostForm("replace"))
	shouldReplace := strings.EqualFold(replaceRaw, "true") || replaceRaw == "1" || strings.EqualFold(replaceRaw, "yes")
	if platform == "" || arch == "" || fileType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform, arch, file_type required"})
		return
	}

	release, err := h.getReleaseForOrg(orgID, releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	if _, ok := h.ensureAppWritable(c, orgID, release.AppID.String()); !ok {
		return
	}
	if !h.hasPermission(c, "release.manage") && !h.hasAppPermission(userID, release.AppID.String(), "release.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	hash := sha256.New()
	tee := io.TeeReader(file, hash)

	key := filepath.ToSlash(filepath.Join(release.AppID.String(), release.ID.String(), platform, arch, header.Filename))
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	storagePath, err := h.Storage.Save(c.Request.Context(), tee, header.Size, key, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store file"})
		return
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	downloadURL := ""
	if strings.EqualFold(h.Cfg.StorageDriver, "local") {
		downloadURL = localFileURL(c, storagePath)
	} else if h.Storage != nil {
		downloadURL, _ = h.Storage.GetDownloadURL(c.Request.Context(), storagePath, 24*time.Hour)
	}
	artifact := models.Artifact{
		ReleaseID:      release.ID,
		Platform:       platform,
		Arch:           arch,
		FileType:       fileType,
		Size:           header.Size,
		ChecksumSHA256: checksum,
		Signature:      signature,
		StorageDriver:  h.Cfg.StorageDriver,
		StoragePath:    storagePath,
		DownloadURL:    downloadURL,
	}
	if err := h.DB.Create(&artifact).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save artifact"})
		return
	}
	if shouldReplace {
		_ = h.DB.Where("release_id = ? AND platform = ? AND arch = ? AND id <> ?", release.ID, platform, arch, artifact.ID).Delete(&models.Artifact{}).Error
	}
	h.audit(c, "artifact.upload", "artifact", artifact.ID, nil, artifact)
	c.JSON(http.StatusOK, gin.H{"artifact": artifact})
}

func (h *Handler) ListArtifacts(c *gin.Context) {
	releaseID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	if _, err := h.getReleaseForOrg(orgID, releaseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	var items []models.Artifact
	if err := h.DB.Where("release_id = ?", releaseID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list artifacts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) DownloadArtifact(c *gin.Context) {
	artifactID := c.Param("id")
	orgID := c.GetString(middleware.ContextOrgID)
	var artifact models.Artifact
	if err := h.DB.Where("id = ?", artifactID).First(&artifact).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
		return
	}
	if _, err := h.getReleaseForOrg(orgID, artifact.ReleaseID.String()); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
		return
	}
	url := ""
	if strings.EqualFold(h.Cfg.StorageDriver, "local") {
		url = localFileURL(c, artifact.StoragePath)
	} else {
		var err error
		url, err = h.Storage.GetDownloadURL(c.Request.Context(), artifact.StoragePath, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download url"})
			return
		}
	}
	c.Redirect(http.StatusFound, url)
}






package handlers

import (
	"net/http"
	"strings"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/crypto"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/version"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// authzTTL bounds how long a signed authorization verdict stays valid. The
// client verifies it right after startup, so a short window is enough and limits
// the value of any captured response.
const authzTTL = 10 * time.Minute

type updateCheckRequest struct {
	ChannelCode    string                 `json:"channel_code"`
	CurrentVersion string                 `json:"current_version"`
	VersionCode    *int                   `json:"version_code"`
	Platform       string                 `json:"platform"`
	Arch           string                 `json:"arch"`
	DeviceID       string                 `json:"device_id"`
	UserID         string                 `json:"user_id"`
	Attributes     map[string]interface{} `json:"attributes"`
}

type updateCheckResponse struct {
	UpdateAvailable          bool                `json:"update_available"`
	Mandatory                bool                `json:"mandatory"`
	HeartbeatIntervalSeconds int                 `json:"heartbeat_interval_seconds"`
	OpenInBrowser            bool                `json:"open_in_browser,omitempty"`
	DeliveryMethod           string              `json:"delivery_method,omitempty"`
	ReleaseID                string              `json:"release_id,omitempty"`
	Version                  string              `json:"version,omitempty"`
	Notes                    string              `json:"notes,omitempty"`
	DownloadURL              string              `json:"download_url,omitempty"`
	ChecksumSHA256           string              `json:"checksum_sha256,omitempty"`
	Signature                string              `json:"signature,omitempty"`
	Size                     int64               `json:"size,omitempty"`
	RollbackAllowed          bool                `json:"rollback_allowed"`
	Maintenance              *maintenanceInfo    `json:"maintenance,omitempty"`
	Authz                    *auth.AuthzEnvelope `json:"authz,omitempty"`
}

type releaseRow struct {
	ReleaseID           uuid.UUID
	Version             string
	VersionCode         *int
	Notes               string
	ExternalDownloadURL string
	Status              string
	PublishedAt         *time.Time
	RolloutPercent      int
	Mandatory           bool
	WhitelistJSON       datatypes.JSON
	RegionRulesJSON     datatypes.JSON
	TargetingRulesJSON  datatypes.JSON
	RolloutStartAt      *time.Time
	RolloutEndAt        *time.Time
	Paused              bool
	ChannelStatus       string
}

func (h *Handler) UpdateCheck(c *gin.Context) {
	var req updateCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	app, _, ok := ClientAppOrgFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID != "" && h.CheckDeviceBlocked(c, app.ID, req.DeviceID) {
		return
	}
	if req.ChannelCode == "" {
		var channel models.Channel
		if err := h.DB.Where("app_id = ? AND is_default = true", app.ID).First(&channel).Error; err == nil {
			req.ChannelCode = channel.Code
		}
	}
	if req.ChannelCode != "" {
		req.ChannelCode = strings.ToLower(req.ChannelCode)
	}
	if req.ChannelCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_code required"})
		return
	}
	req.ChannelCode = strings.ToLower(req.ChannelCode)
	heartbeatInterval := app.HeartbeatIntervalSeconds
	if heartbeatInterval < 10 || heartbeatInterval > 3600 {
		heartbeatInterval = 60
	}

	maintenance := h.BuildMaintenanceInfo(app)
	// The signed authz envelope is the trust anchor the client requires before it
	// will run: it binds this app, the client-supplied X-Nonce challenge and the
	// device id, and is signed with a server-only Ed25519 key. Reaching respond()
	// means the device passed the block check, so we sign an "allow"; blocked
	// devices return earlier without an envelope and the client fails closed.
	authzNonce := strings.TrimSpace(c.GetHeader(signHeaderNonce))
	respond := func(resp updateCheckResponse) {
		resp.Maintenance = maintenance
		if h.AuthzSigner != nil {
			env := h.AuthzSigner.SignAllow(app.ID.String(), req.DeviceID, authzNonce, authzTTL)
			resp.Authz = &env
		}
		c.JSON(http.StatusOK, resp)
	}

	attrs := NormalizeAttributes(req.Attributes)
	if req.UserID != "" {
		attrs["user_id"] = req.UserID
	}
	region := ResolveRegion(h, attrs, c.ClientIP())
	if region.ISO != "" {
		attrs["country_iso"] = region.ISO
	}
	if region.Country != "" {
		attrs["country"] = region.Country
	} else if region.ISO != "" {
		attrs["country"] = region.ISO
	}
	if region.Province != "" {
		attrs["province"] = region.Province
	}
	if region.City != "" {
		attrs["city"] = region.City
	}
	if req.DeviceID != "" {
		_ = h.UpsertDevice(app.ID, req.DeviceID, req.Platform, req.Arch, attrs, req.CurrentVersion, c.ClientIP())
	}

	var rows []releaseRow
	selectExternalURL := "r.external_download_url"
	if !h.HasReleaseExternalDownloadURLColumn() {
		selectExternalURL = "''"
	}
	query := `
		SELECT r.id as release_id, r.version, r.version_code, r.notes, ` + selectExternalURL + ` as external_download_url, r.status, r.created_at,
			rc.published_at, rc.rollout_percent, rc.mandatory, rc.whitelist_json, rc.region_rules_json,
			rc.targeting_rules_json, rc.rollout_start_at, rc.rollout_end_at, rc.paused, rc.status as channel_status
		FROM release_channels rc
		JOIN releases r ON r.id = rc.release_id
		JOIN channels c ON c.id = rc.channel_id
		WHERE c.app_id = ? AND c.code = ? AND r.status = 'published' AND rc.status = 'active'
		ORDER BY rc.published_at DESC
	`
	if err := h.DB.Raw(query, app.ID, req.ChannelCode).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load releases"})
		return
	}

	appRegionRules := app.RegionRulesJSON
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	arch := strings.ToLower(strings.TrimSpace(req.Arch))
	var matched *releaseRow
	var currentMatched *releaseRow
	currentVersionRegionBlocked := false
	for _, row := range rows {
		if row.Paused || row.ChannelStatus != "active" {
			continue
		}
		if !withinRolloutWindow(row.RolloutStartAt, row.RolloutEndAt) {
			continue
		}
		newer := isNewer(req.CurrentVersion, req.VersionCode, row.Version, row.VersionCode)
		sameVersion := isSameVersion(req.CurrentVersion, req.VersionCode, row.Version, row.VersionCode)
		if !newer && !sameVersion {
			continue
		}
		rules := appRegionRules
		if regionRulesHasContent(row.RegionRulesJSON) {
			rules = row.RegionRulesJSON
		}
		if !matchesTargetingRules(row.TargetingRulesJSON, req, attrs) {
			continue
		}
		if !matchesRegionRules(rules, region) {
			if sameVersion {
				currentVersionRegionBlocked = true
			}
			continue
		}
		if !newer {
			if sameVersion && currentMatched == nil {
				rowCopy := row
				currentMatched = &rowCopy
			}
			continue
		}
		if isWhitelisted(req.DeviceID, row.WhitelistJSON) || crypto.HashPercent(req.DeviceID+row.ReleaseID.String()) < row.RolloutPercent {
			matched = &row
			break
		}
	}

	if matched == nil {
		if currentVersionRegionBlocked {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "update_region_blocked",
					"message": "region blocked",
				},
			})
			return
		}
		if currentMatched != nil {
			var currentArtifact models.Artifact
			if err := h.DB.Where(
				`release_id = ? AND (platform = ? OR platform = 'universal') AND (arch = ? OR arch = 'universal')`,
				currentMatched.ReleaseID,
				platform,
				arch,
			).Order("CASE WHEN platform = 'universal' THEN 1 ELSE 0 END ASC").
				Order("CASE WHEN arch = 'universal' THEN 1 ELSE 0 END ASC").
				First(&currentArtifact).Error; err == nil {
				respond(updateCheckResponse{
					UpdateAvailable:          false,
					HeartbeatIntervalSeconds: heartbeatInterval,
					ReleaseID:                currentMatched.ReleaseID.String(),
					Version:                  currentMatched.Version,
					ChecksumSHA256:           currentArtifact.ChecksumSHA256,
					Signature:                currentArtifact.Signature,
					Size:                     currentArtifact.Size,
					RollbackAllowed:          true,
				})
				return
			}
		}
		respond(updateCheckResponse{
			UpdateAvailable:          false,
			HeartbeatIntervalSeconds: heartbeatInterval,
		})
		return
	}

	var artifact models.Artifact
	if err := h.DB.Where(
		`release_id = ? AND (platform = ? OR platform = 'universal') AND (arch = ? OR arch = 'universal')`,
		matched.ReleaseID,
		platform,
		arch,
	).Order("CASE WHEN platform = 'universal' THEN 1 ELSE 0 END ASC").
		Order("CASE WHEN arch = 'universal' THEN 1 ELSE 0 END ASC").
		First(&artifact).Error; err != nil {
		if link := strings.TrimSpace(matched.ExternalDownloadURL); link != "" && h.HasReleaseExternalDownloadURLColumn() {
			mandatory := matched.Mandatory
			if channelMinVersion := h.GetChannelMinVersion(app.ID, req.ChannelCode); channelMinVersion != "" {
				if version.CompareVersion(req.CurrentVersion, channelMinVersion) < 0 {
					mandatory = true
				}
			}
			respond(updateCheckResponse{
				UpdateAvailable:          true,
				Mandatory:                mandatory,
				HeartbeatIntervalSeconds: heartbeatInterval,
				OpenInBrowser:            true,
				DeliveryMethod:           "external_link",
				ReleaseID:                matched.ReleaseID.String(),
				Version:                  matched.Version,
				Notes:                    matched.Notes,
				DownloadURL:              link,
				RollbackAllowed:          true,
			})
			return
		}
		respond(updateCheckResponse{
			UpdateAvailable:          false,
			HeartbeatIntervalSeconds: heartbeatInterval,
		})
		return
	}

	downloadURL := ""
	if strings.EqualFold(h.Cfg.StorageDriver, "local") {
		downloadURL = h.BuildLocalFileURL(c, artifact.StoragePath, 24*time.Hour)
	} else {
		var err error
		downloadURL, err = h.Storage.GetDownloadURL(c.Request.Context(), artifact.StoragePath, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download url"})
			return
		}
	}

	mandatory := matched.Mandatory
	if channelMinVersion := h.GetChannelMinVersion(app.ID, req.ChannelCode); channelMinVersion != "" {
		if version.CompareVersion(req.CurrentVersion, channelMinVersion) < 0 {
			mandatory = true
		}
	}

	resp := updateCheckResponse{
		UpdateAvailable:          true,
		Mandatory:                mandatory,
		HeartbeatIntervalSeconds: heartbeatInterval,
		OpenInBrowser:            false,
		DeliveryMethod:           "package",
		ReleaseID:                matched.ReleaseID.String(),
		Version:                  matched.Version,
		Notes:                    matched.Notes,
		DownloadURL:              downloadURL,
		ChecksumSHA256:           artifact.ChecksumSHA256,
		Signature:                artifact.Signature,
		Size:                     artifact.Size,
		RollbackAllowed:          true,
	}

	respond(resp)
}

package core

import (
	"net/http"

	"software-web-manager/backend/internal/models"
	appsvc "software-web-manager/backend/internal/services/app"
	orgsvc "software-web-manager/backend/internal/services/org"
	releasesvc "software-web-manager/backend/internal/services/release"

	"github.com/gin-gonic/gin"
)

// These domain queries delegate to their service packages. They are kept as thin
// methods so existing call sites (h.GetAppForOrg, ...) remain unchanged.

func (h *Handler) GetAppForOrg(orgID, appID string) (models.App, error) {
	return appsvc.NewService(h.DB).GetForOrg(orgID, appID)
}

func (h *Handler) GetReleaseForOrg(orgID, releaseID string) (models.Release, error) {
	return releasesvc.NewService(h.DB).GetForOrg(orgID, releaseID)
}

func (h *Handler) GetOrgMember(orgID string, userID string) (models.OrgMember, error) {
	return orgsvc.NewService(h.DB).GetMember(orgID, userID)
}

func (h *Handler) CountOrgOwners(orgID string) (int64, error) {
	return orgsvc.NewService(h.DB).CountOwners(orgID)
}

func (h *Handler) IsEnterpriseOwner(userID string) (bool, error) {
	return orgsvc.NewService(h.DB).IsEnterpriseOwner(userID)
}

func (h *Handler) ResolveSystemRole(userID string, systemRole string) (string, error) {
	return orgsvc.NewService(h.DB).ResolveSystemRole(userID, systemRole)
}

func (h *Handler) IsPersonalOrg(orgID string) (bool, error) {
	return orgsvc.NewService(h.DB).IsPersonal(orgID)
}

func (h *Handler) EnsurePersonalOrgMember(userID string) (models.Org, models.OrgMember, error) {
	return orgsvc.NewService(h.DB).EnsurePersonalMember(userID)
}

// EnsureAppWritable loads an app for the org and, for personal orgs, enforces
// that it is in a writable state. On failure it writes the HTTP response and
// returns ok=false; the writability rule itself lives in services/org.
func (h *Handler) EnsureAppWritable(c *gin.Context, orgID, appID string) (models.App, bool) {
	app, err := h.GetAppForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return app, false
	}
	personal, err := h.IsPersonalOrg(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return app, false
	}
	if personal {
		if block := orgsvc.AppWriteBlockForPersonal(app.Status); block != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":            block.Code,
				"status":           block.Status,
				"rejection_reason": app.RejectionReason,
			})
			return app, false
		}
	}
	return app, true
}

package common

import (
	"net/http"

	"software-web-manager/backend/internal/models"
	appsvc "software-web-manager/backend/internal/services/app"
	orgsvc "software-web-manager/backend/internal/services/org"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EnsureAppWritable loads an app scoped to the org and, for personal orgs,
// enforces that it is in a writable state. On failure it writes the HTTP
// response and returns ok=false. The writability rule lives in services/org.
func EnsureAppWritable(db *gorm.DB, c *gin.Context, orgID, appID string) (models.App, bool) {
	app, err := appsvc.NewService(db).GetForOrg(orgID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return app, false
	}
	personal, err := orgsvc.NewService(db).IsPersonal(orgID)
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

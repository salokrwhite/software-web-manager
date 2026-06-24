// Package geo serves the geo lookup endpoints: resolving an IP to a region and
// listing the known country/province/city catalog.
package geo

import (
	"net/http"
	"strings"

	"software-web-manager/backend/internal/core"
	geosvc "software-web-manager/backend/internal/services/geo"

	"github.com/gin-gonic/gin"
)

// Handler serves the geo endpoints over the shared core.
type Handler struct {
	*core.Handler
}

// New builds a geo handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the geo routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/geo/resolve", h.ResolveGeo)
	rg.GET("/geo/regions", h.ListGeoRegions)
}

func (h *Handler) ResolveGeo(c *gin.Context) {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ip required"})
		return
	}
	if h.RegionResolver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "geo resolver not available"})
		return
	}
	region, err := h.RegionResolver.Resolve(ip)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to resolve ip"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"country":  region.Country,
		"province": region.Province,
		"city":     region.City,
		"iso":      region.ISO,
	})
}

func (h *Handler) ListGeoRegions(c *gin.Context) {
	list, err := geosvc.LoadRegions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load regions"})
		return
	}
	c.JSON(http.StatusOK, list)
}

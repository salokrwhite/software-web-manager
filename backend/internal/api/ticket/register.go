package ticket

import (
	"software-web-manager/backend/internal/core"

	"github.com/gin-gonic/gin"
)

// Handler serves the ticket-domain endpoints (org-user tickets and system tickets).
type Handler struct {
	*core.Handler
}

// New builds a ticket handler over the shared core.
func New(core *core.Handler) *Handler {
	return &Handler{Handler: core}
}

// RegisterRoutes wires the org-user ticket routes onto the authenticated API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/tickets", h.ListTickets)
	rg.GET("/tickets/:id", h.GetTicket)
	rg.POST("/tickets", h.CreateTicket)
	rg.PATCH("/tickets/:id/close", h.CloseTicket)
	rg.DELETE("/tickets/:id", h.DeleteTicket)
	rg.POST("/tickets/batch-delete", h.BatchDeleteTickets)
	rg.GET("/tickets/:id/messages", h.ListTicketMessages)
	rg.POST("/tickets/:id/messages", h.CreateTicketMessage)
}

// RegisterSystemRoutes wires the system (admin) ticket routes onto the system API group.
func (h *Handler) RegisterSystemRoutes(rg *gin.RouterGroup) {
	rg.GET("/tickets", h.ListSystemTickets)
	rg.GET("/tickets/:id", h.GetSystemTicket)
	rg.POST("/tickets", h.CreateSystemTicket)
	rg.DELETE("/tickets/:id", h.DeleteSystemTicket)
	rg.POST("/tickets/batch-delete", h.BatchDeleteSystemTickets)
	rg.PATCH("/tickets/:id/status", h.UpdateTicketStatus)
	rg.GET("/tickets/:id/messages", h.ListSystemTicketMessages)
	rg.POST("/tickets/:id/messages", h.CreateSystemTicketMessage)
}

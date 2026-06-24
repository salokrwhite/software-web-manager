package ticket

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	ticketsvc "software-web-manager/backend/internal/services/ticket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *Handler) CreateSystemTicket(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if err := h.EnsureStorage(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	orgID := strings.TrimSpace(c.PostForm("org_id"))
	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}
	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}

	ticketID := uuid.New()
	attachments, statusCode, err := h.storeTicketAttachments(c, ticketID, orgUUID, userUUID)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	ticket := models.Ticket{
		ID:           ticketID,
		OrgID:        orgUUID,
		CreatedBy:    userUUID,
		Title:        title,
		Description:  description,
		Status:       "submitted",
		AssigneeType: "system",
	}

	if err := ticketsvc.NewService(h.DB).CreateWithAttachments(&ticket, attachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	common.AuditWithOrg(h.DB, c, orgUUID, "ticket.create", "ticket", ticket.ID, nil, ticket)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

func (h *Handler) ListSystemTickets(c *gin.Context) {
	orgID := strings.TrimSpace(c.Query("org_id"))
	status := ticketsvc.NormalizeStatus(c.Query("status"))
	q := strings.TrimSpace(c.Query("q"))
	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := common.ParseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := common.ParseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	items, total, err := ticketsvc.NewService(h.DB).ListSystem(orgID, status, q, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) GetSystemTicket(c *gin.Context) {
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}

	row, err := ticketsvc.NewService(h.DB).GetDetailSystem(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	attachments, err := h.loadTicketAttachments(c, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attachments"})
		return
	}

	resp := ticketDetailResponse{
		TicketDetailRow: row,
		Attachments:     attachments,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteSystemTicket(c *gin.Context) {
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}
	if _, err := uuid.Parse(ticketID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket_id"})
		return
	}

	svc := ticketsvc.NewService(h.DB)
	ticket, err := svc.GetByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	paths, err := svc.PurgeTickets([]string{ticketID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	h.DeleteStoragePaths(c.Request.Context(), paths)
	h.DeleteLocalTicketDir(ticketID)

	common.AuditWithOrg(h.DB, c, ticket.OrgID, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemTickets(c *gin.Context) {
	var req batchDeleteTicketsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids, err := ticketsvc.NormalizeIDs(req.IDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tickets []models.Ticket
	if err := h.DB.Where("id IN ?", ids).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}
	if len(tickets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tickets not found"})
		return
	}

	foundIDs := make([]string, 0, len(tickets))
	for _, ticket := range tickets {
		foundIDs = append(foundIDs, ticket.ID.String())
	}

	paths, err := ticketsvc.NewService(h.DB).PurgeTickets(foundIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tickets"})
		return
	}

	h.DeleteStoragePaths(c.Request.Context(), paths)
	for _, ticketID := range foundIDs {
		h.DeleteLocalTicketDir(ticketID)
	}

	for _, ticket := range tickets {
		common.AuditWithOrg(h.DB, c, ticket.OrgID, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": len(foundIDs)})
}

func (h *Handler) ListSystemTicketMessages(c *gin.Context) {
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}
	if _, err := ticketsvc.NewService(h.DB).GetByID(ticketID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}
	items, err := h.loadTicketMessages(c, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateSystemTicketMessage(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}
	if err := h.EnsureStorage(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	svc := ticketsvc.NewService(h.DB)
	ticket, err := svc.GetByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	content := strings.TrimSpace(c.PostForm("content"))
	messageID := uuid.New()
	attachments, statusCode, err := h.storeTicketMessageAttachments(c, ticketID, messageID, ticket.OrgID, userUUID)
	if err != nil {
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}
	if content == "" && len(attachments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content or attachments required"})
		return
	}

	message := models.TicketMessage{
		ID:         messageID,
		TicketID:   ticket.ID,
		OrgID:      ticket.OrgID,
		UserID:     userUUID,
		SenderType: "system",
		Content:    content,
	}

	if err := svc.CreateMessageWithAttachments(&message, attachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}

	common.AuditWithOrg(h.DB, c, ticket.OrgID, "ticket.message.create", "ticket_message", message.ID, nil, message)
	if msg, err := h.loadTicketMessageByID(c, message.ID.String()); err == nil {
		h.PublishTicketEvent("ticket.message.created", ticket.ID.String(), ticket.OrgID.String(), msg)
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

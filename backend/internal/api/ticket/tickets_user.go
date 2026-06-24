package ticket

import (
	"errors"
	"net/http"
	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/rbac"
	ticketsvc "software-web-manager/backend/internal/services/ticket"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *Handler) CreateTicket(c *gin.Context) {
	if !common.RequirePermission(c, "ticket.manage") {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if orgID == "" || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
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

	svc := ticketsvc.NewService(h.DB)
	isPersonalOrg, err := svc.IsPersonalOrg(orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	assigneeType := strings.ToLower(strings.TrimSpace(c.PostForm("assignee_type")))
	assigneeUserIDRaw := strings.TrimSpace(c.PostForm("assignee_user_id"))
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	if assigneeType == "" {
		assigneeType = "system"
	}
	if isPersonalOrg {
		assigneeType = "system"
		assigneeUserIDRaw = ""
	}
	if assigneeType != "system" && assigneeType != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignee_type"})
		return
	}

	var assigneeUserID *uuid.UUID
	if assigneeType == "user" {
		if assigneeUserIDRaw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignee_user_id required"})
			return
		}
		parsed, err := uuid.Parse(assigneeUserIDRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignee_user_id"})
			return
		}
		inOrg, err := svc.AssigneeInOrg(orgUUID, parsed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate assignee"})
			return
		}
		if !inOrg {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignee not in org"})
			return
		}
		assigneeUserID = &parsed
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
		ID:             ticketID,
		OrgID:          orgUUID,
		CreatedBy:      userUUID,
		Title:          title,
		Description:    description,
		Status:         "submitted",
		AssigneeType:   assigneeType,
		AssigneeUserID: assigneeUserID,
	}

	if err := svc.CreateWithAttachments(&ticket, attachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	common.Audit(h.DB, c, "ticket.create", "ticket", ticket.ID, nil, ticket)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

func (h *Handler) ListTickets(c *gin.Context) {
	if !common.RequirePermission(c, rbac.PermissionRoleViewer) {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	status := ticketsvc.NormalizeStatus(c.Query("status"))
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

	items, total, err := ticketsvc.NewService(h.DB).ListForOrg(orgID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) GetTicket(c *gin.Context) {
	if !common.RequirePermission(c, rbac.PermissionRoleViewer) {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}

	row, err := ticketsvc.NewService(h.DB).GetDetailForOrg(orgID, ticketID)
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

func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}

	var req updateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nextStatus := ticketsvc.NormalizeStatus(req.Status)
	if !ticketsvc.IsValidStatus(nextStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
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

	before := ticket
	changed, err := svc.ApplyStatusTransition(ticket, nextStatus)
	if err != nil {
		if errors.Is(err, ticketsvc.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}
	if !changed {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	var after models.Ticket
	if err := h.DB.Where("id = ?", ticket.ID).First(&after).Error; err == nil {
		common.AuditWithOrg(h.DB, c, ticket.OrgID, "ticket.status.update", "ticket", ticket.ID, before, after)
		h.PublishTicketEvent("ticket.status.updated", after.ID.String(), after.OrgID.String(), gin.H{
			"id":             after.ID.String(),
			"status":         after.Status,
			"in_progress_at": after.InProgressAt,
			"resolved_at":    after.ResolvedAt,
			"updated_at":     after.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CloseTicket(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if orgID == "" || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
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
	ticket, err := svc.GetForOrg(orgID, ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}
	if ticket.CreatedBy.String() != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	before := ticket
	changed, err := svc.ApplyStatusTransition(ticket, "resolved")
	if err != nil {
		if errors.Is(err, ticketsvc.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close ticket"})
		return
	}
	if !changed {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	var after models.Ticket
	if err := h.DB.Where("id = ?", ticket.ID).First(&after).Error; err == nil {
		common.AuditWithOrg(h.DB, c, ticket.OrgID, "ticket.status.update", "ticket", ticket.ID, before, after)
		h.PublishTicketEvent("ticket.status.updated", after.ID.String(), after.OrgID.String(), gin.H{
			"id":             after.ID.String(),
			"status":         after.Status,
			"in_progress_at": after.InProgressAt,
			"resolved_at":    after.ResolvedAt,
			"updated_at":     after.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DeleteTicket(c *gin.Context) {
	if !common.RequirePermission(c, "ticket.manage") {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
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
	ticket, err := svc.GetForOrg(orgID, ticketID)
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

	common.Audit(h.DB, c, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteTickets(c *gin.Context) {
	if !common.RequirePermission(c, "ticket.manage") {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
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
	if err := h.DB.Where("id IN ? AND org_id = ?", ids, orgID).Find(&tickets).Error; err != nil {
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
		common.Audit(h.DB, c, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": len(foundIDs)})
}

func (h *Handler) ListTicketMessages(c *gin.Context) {
	if !common.RequirePermission(c, rbac.PermissionRoleViewer) {
		return
	}
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
		return
	}

	if _, err := ticketsvc.NewService(h.DB).GetForOrg(orgID, ticketID); err != nil {
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

func (h *Handler) CreateTicketMessage(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if orgID == "" || userID == "" {
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
	ticket, err := svc.GetForOrg(orgID, ticketID)
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
		SenderType: "org",
		Content:    content,
	}

	if err := svc.CreateMessageWithAttachments(&message, attachments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}

	common.Audit(h.DB, c, "ticket.message.create", "ticket_message", message.ID, nil, message)
	if msg, err := h.loadTicketMessageByID(c, message.ID.String()); err == nil {
		h.PublishTicketEvent("ticket.message.created", ticket.ID.String(), ticket.OrgID.String(), msg)
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

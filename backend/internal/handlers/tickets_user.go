package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"
	"strings"
	"time"
)

func (h *Handler) CreateTicket(c *gin.Context) {
	if !h.requirePermission(c, "ticket.manage") {
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
	if err := h.ensureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	isPersonalOrg := false
	if h.hasOrgTypeColumn() {
		var org models.Org
		if err := h.DB.Select("id", "org_type").Where("id = ?", orgUUID).First(&org).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
			return
		}
		isPersonalOrg = strings.EqualFold(strings.TrimSpace(org.OrgType), "personal")
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
		var count int64
		if err := h.DB.Model(&models.OrgMember{}).
			Where("org_id = ? AND user_id = ?", orgUUID, parsed).
			Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate assignee"})
			return
		}
		if count == 0 {
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

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	h.audit(c, "ticket.create", "ticket", ticket.ID, nil, ticket)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

func (h *Handler) ListTickets(c *gin.Context) {
	orgID := strings.TrimSpace(c.GetString(middleware.ContextOrgID))
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	status := normalizeTicketStatus(c.Query("status"))
	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 0 {
			offset = n
		}
	}

	countDB := h.DB.Model(&models.Ticket{}).Where("org_id = ?", orgID)
	if status != "" {
		countDB = countDB.Where("status = ?", status)
	}
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tickets"})
		return
	}

	db := h.DB.Table("tickets t").
		Select(`t.id, t.title, t.status, t.assignee_type, t.assignee_user_id, t.created_at,
		        u.email as created_by_email, au.email as assignee_email,
		        (SELECT COUNT(*) FROM attachments ta WHERE ta.owner_type = 'ticket' AND ta.owner_id = t.id) as attachment_count`).
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.org_id = ?", orgID)
	if status != "" {
		db = db.Where("t.status = ?", status)
	}

	var items []ticketListItem
	if err := db.Order("t.created_at desc").Limit(limit).Offset(offset).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) GetTicket(c *gin.Context) {
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

	db := h.DB.Table("tickets t").
		Select(`t.id, t.org_id, t.created_by, t.title, t.description, t.status, t.assignee_type, t.assignee_user_id,
		        t.in_progress_at, t.resolved_at, t.created_at, t.updated_at,
		        u.email as created_by_email, au.email as assignee_email`).
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.id = ? AND t.org_id = ?", ticketID, orgID)

	var row ticketDetailRow
	if err := db.Take(&row).Error; err != nil {
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
		ticketDetailRow: row,
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
	nextStatus := normalizeTicketStatus(req.Status)
	if !isValidTicketStatus(nextStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	var ticket models.Ticket
	if err := h.DB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	currentStatus := normalizeTicketStatus(ticket.Status)
	if !canTransitionTicketStatus(currentStatus, nextStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition"})
		return
	}
	if currentStatus == nextStatus {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status": nextStatus,
	}
	if nextStatus == "in_progress" {
		updates["in_progress_at"] = now
	}
	if nextStatus == "resolved" {
		updates["resolved_at"] = now
	}

	before := ticket
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", ticket.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}
	var after models.Ticket
	if err := h.DB.Where("id = ?", ticket.ID).First(&after).Error; err == nil {
		h.auditWithOrg(c, ticket.OrgID, "ticket.status.update", "ticket", ticket.ID, before, after)
		h.publishTicketEvent("ticket.status.updated", after.ID.String(), after.OrgID.String(), gin.H{
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

	var ticket models.Ticket
	if err := h.DB.Where("id = ? AND org_id = ?", ticketID, orgID).First(&ticket).Error; err != nil {
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

	currentStatus := normalizeTicketStatus(ticket.Status)
	nextStatus := "resolved"
	if currentStatus == nextStatus {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if !canTransitionTicketStatus(currentStatus, nextStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status transition"})
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status":      nextStatus,
		"resolved_at": now,
	}

	before := ticket
	if err := h.DB.Model(&models.Ticket{}).Where("id = ?", ticket.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close ticket"})
		return
	}
	var after models.Ticket
	if err := h.DB.Where("id = ?", ticket.ID).First(&after).Error; err == nil {
		h.auditWithOrg(c, ticket.OrgID, "ticket.status.update", "ticket", ticket.ID, before, after)
		h.publishTicketEvent("ticket.status.updated", after.ID.String(), after.OrgID.String(), gin.H{
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
	if !h.requirePermission(c, "ticket.manage") {
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

	var ticket models.Ticket
	if err := h.DB.Where("id = ? AND org_id = ?", ticketID, orgID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	attachmentPaths, err := loadAttachmentStoragePaths(h.DB, attachmentOwnerTicket, []string{ticketID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attachments"})
		return
	}

	var messageIDs []string
	if err := h.DB.Model(&models.TicketMessage{}).Where("ticket_id = ?", ticketID).Pluck("id", &messageIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load messages"})
		return
	}

	var messageAttachmentPaths []string
	if len(messageIDs) > 0 {
		messageAttachmentPaths, err = loadAttachmentStoragePaths(h.DB, attachmentOwnerTicketMessage, messageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load message attachments"})
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(messageIDs) > 0 {
			if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicketMessage, messageIDs); err != nil {
				return err
			}
		}
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicket, []string{ticketID}); err != nil {
			return err
		}
		if err := tx.Where("id = ?", ticketID).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	paths := append(attachmentPaths, messageAttachmentPaths...)
	h.deleteStoragePaths(c, paths)
	h.deleteLocalTicketDir(ticketID)

	h.audit(c, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteTickets(c *gin.Context) {
	if !h.requirePermission(c, "ticket.manage") {
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
	ids, err := normalizeTicketIDs(req.IDs)
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

	attachmentPaths, err := loadAttachmentStoragePaths(h.DB, attachmentOwnerTicket, foundIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attachments"})
		return
	}

	var messageIDs []string
	if err := h.DB.Model(&models.TicketMessage{}).Where("ticket_id IN ?", foundIDs).Pluck("id", &messageIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load messages"})
		return
	}

	var messageAttachmentPaths []string
	if len(messageIDs) > 0 {
		messageAttachmentPaths, err = loadAttachmentStoragePaths(h.DB, attachmentOwnerTicketMessage, messageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load message attachments"})
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(messageIDs) > 0 {
			if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicketMessage, messageIDs); err != nil {
				return err
			}
		}
		if err := tx.Where("ticket_id IN ?", foundIDs).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := deleteAttachmentsByOwners(tx, attachmentOwnerTicket, foundIDs); err != nil {
			return err
		}
		if err := tx.Where("id IN ?", foundIDs).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tickets"})
		return
	}

	paths := append(attachmentPaths, messageAttachmentPaths...)
	h.deleteStoragePaths(c, paths)
	for _, ticketID := range foundIDs {
		h.deleteLocalTicketDir(ticketID)
	}

	for _, ticket := range tickets {
		h.audit(c, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": len(foundIDs)})
}

func (h *Handler) ListTicketMessages(c *gin.Context) {
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

	var ticket models.Ticket
	if err := h.DB.Where("id = ? AND org_id = ?", ticketID, orgID).First(&ticket).Error; err != nil {
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
	if err := h.ensureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	var ticket models.Ticket
	if err := h.DB.Where("id = ? AND org_id = ?", ticketID, orgID).First(&ticket).Error; err != nil {
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

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}

	h.audit(c, "ticket.message.create", "ticket_message", message.ID, nil, message)
	if msg, err := h.loadTicketMessageByID(c, message.ID.String()); err == nil {
		h.publishTicketEvent("ticket.message.created", ticket.ID.String(), ticket.OrgID.String(), msg)
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

package ticket

import (
	"errors"
	"net/http"
	"strings"

	"software-web-manager/backend/internal/api/common"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

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
	if err := h.EnsureStorage(c); err != nil {
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

	h.AuditWithOrg(c, orgUUID, "ticket.create", "ticket", ticket.ID, nil, ticket)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

func (h *Handler) ListSystemTickets(c *gin.Context) {
	orgID := strings.TrimSpace(c.Query("org_id"))
	status := normalizeTicketStatus(c.Query("status"))
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

	countDB := h.DB.Model(&models.Ticket{})
	if orgID != "" {
		countDB = countDB.Where("org_id = ?", orgID)
	}
	if status != "" {
		countDB = countDB.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		countDB = countDB.Where("title LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tickets"})
		return
	}

	db := h.DB.Table("tickets t").
		Select(`t.id, t.org_id, o.name as org_name, t.title, t.status, t.assignee_type, t.assignee_user_id, t.created_at,
		        u.email as created_by_email, au.email as assignee_email,
		        (SELECT COUNT(*) FROM attachments ta WHERE ta.owner_type = 'ticket' AND ta.owner_id = t.id) as attachment_count`).
		Joins("LEFT JOIN orgs o ON o.id = t.org_id").
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id")
	if orgID != "" {
		db = db.Where("t.org_id = ?", orgID)
	}
	if status != "" {
		db = db.Where("t.status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("t.title LIKE ? OR t.description LIKE ?", like, like)
	}

	var items []systemTicketListItem
	if err := db.Order("t.created_at desc").Limit(limit).Offset(offset).Scan(&items).Error; err != nil {
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

	db := h.DB.Table("tickets t").
		Select(`t.id, t.org_id, o.name as org_name, t.created_by, t.title, t.description, t.status, t.assignee_type, t.assignee_user_id,
		        t.in_progress_at, t.resolved_at, t.created_at, t.updated_at,
		        u.email as created_by_email, au.email as assignee_email`).
		Joins("LEFT JOIN orgs o ON o.id = t.org_id").
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.id = ?", ticketID)

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

	var ticket models.Ticket
	if err := h.DB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket"})
		return
	}

	attachmentPaths, err := core.LoadAttachmentStoragePaths(h.DB, core.AttachmentOwnerTicket, []string{ticketID})
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
		messageAttachmentPaths, err = core.LoadAttachmentStoragePaths(h.DB, core.AttachmentOwnerTicketMessage, messageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load message attachments"})
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(messageIDs) > 0 {
			if err := core.DeleteAttachmentsByOwners(tx, core.AttachmentOwnerTicketMessage, messageIDs); err != nil {
				return err
			}
		}
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := core.DeleteAttachmentsByOwners(tx, core.AttachmentOwnerTicket, []string{ticketID}); err != nil {
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
	h.DeleteStoragePaths(c, paths)
	h.DeleteLocalTicketDir(ticketID)

	h.AuditWithOrg(c, ticket.OrgID, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) BatchDeleteSystemTickets(c *gin.Context) {
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

	attachmentPaths, err := core.LoadAttachmentStoragePaths(h.DB, core.AttachmentOwnerTicket, foundIDs)
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
		messageAttachmentPaths, err = core.LoadAttachmentStoragePaths(h.DB, core.AttachmentOwnerTicketMessage, messageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load message attachments"})
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(messageIDs) > 0 {
			if err := core.DeleteAttachmentsByOwners(tx, core.AttachmentOwnerTicketMessage, messageIDs); err != nil {
				return err
			}
		}
		if err := tx.Where("ticket_id IN ?", foundIDs).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := core.DeleteAttachmentsByOwners(tx, core.AttachmentOwnerTicket, foundIDs); err != nil {
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
	h.DeleteStoragePaths(c, paths)
	for _, ticketID := range foundIDs {
		h.DeleteLocalTicketDir(ticketID)
	}

	for _, ticket := range tickets {
		h.AuditWithOrg(c, ticket.OrgID, "ticket.delete", "ticket", ticket.ID, ticket, nil)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": len(foundIDs)})
}

func (h *Handler) ListSystemTicketMessages(c *gin.Context) {
	ticketID := strings.TrimSpace(c.Param("id"))
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id required"})
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
	if err := h.EnsureStorage(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
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

	h.AuditWithOrg(c, ticket.OrgID, "ticket.message.create", "ticket_message", message.ID, nil, message)
	if msg, err := h.loadTicketMessageByID(c, message.ID.String()); err == nil {
		h.PublishTicketEvent("ticket.message.created", ticket.ID.String(), ticket.OrgID.String(), msg)
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

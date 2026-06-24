package ticket

import (
	"strings"
	"time"

	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	attachment "software-web-manager/backend/internal/services/attachment"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TicketListItem is one row of the org-scoped ticket list.
type TicketListItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AssigneeType    string    `json:"assignee_type"`
	AssigneeUserID  *string   `json:"assignee_user_id"`
	AssigneeEmail   string    `json:"assignee_email"`
	CreatedByEmail  string    `json:"created_by_email"`
	CreatedAt       time.Time `json:"created_at"`
	AttachmentCount int64     `json:"attachment_count"`
}

// SystemTicketListItem is one row of the system (admin) ticket list.
type SystemTicketListItem struct {
	ID              string    `json:"id"`
	OrgID           string    `json:"org_id"`
	OrgName         string    `json:"org_name"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AssigneeType    string    `json:"assignee_type"`
	AssigneeUserID  *string   `json:"assignee_user_id"`
	AssigneeEmail   string    `json:"assignee_email"`
	CreatedByEmail  string    `json:"created_by_email"`
	CreatedAt       time.Time `json:"created_at"`
	AttachmentCount int64     `json:"attachment_count"`
}

// TicketDetailRow is the flattened ticket detail (without attachments).
type TicketDetailRow struct {
	ID             string     `json:"id"`
	OrgID          string     `json:"org_id"`
	OrgName        string     `json:"org_name,omitempty"`
	CreatedBy      string     `json:"created_by"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	AssigneeType   string     `json:"assignee_type"`
	AssigneeUserID *string    `json:"assignee_user_id"`
	CreatedByEmail string     `json:"created_by_email"`
	AssigneeEmail  string     `json:"assignee_email"`
	InProgressAt   *time.Time `json:"in_progress_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// GetByID loads a ticket by id, returning the underlying gorm error when missing.
func (s *Service) GetByID(ticketID string) (models.Ticket, error) {
	var ticket models.Ticket
	if err := s.DB.Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		return ticket, err
	}
	return ticket, nil
}

// GetForOrg loads a ticket scoped to an org, returning the underlying gorm error
// when missing.
func (s *Service) GetForOrg(orgID, ticketID string) (models.Ticket, error) {
	var ticket models.Ticket
	if err := s.DB.Where("id = ? AND org_id = ?", ticketID, orgID).First(&ticket).Error; err != nil {
		return ticket, err
	}
	return ticket, nil
}

// IsPersonalOrg reports whether the org is a personal org. It returns false when
// the org_type column is absent.
func (s *Service) IsPersonalOrg(orgID uuid.UUID) (bool, error) {
	if !schema.HasOrgTypeColumn(s.DB) {
		return false, nil
	}
	var org models.Org
	if err := s.DB.Select("id", "org_type").Where("id = ?", orgID).First(&org).Error; err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(org.OrgType), "personal"), nil
}

// AssigneeInOrg reports whether the given user is a member of the org and may be
// assigned a ticket.
func (s *Service) AssigneeInOrg(orgID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := s.DB.Model(&models.OrgMember{}).
		Where("scope_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateWithAttachments persists a ticket and any attachment rows in one
// transaction.
func (s *Service) CreateWithAttachments(ticket *models.Ticket, attachments []models.Attachment) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	})
}

// CreateMessageWithAttachments persists a ticket message and any attachment rows
// in one transaction.
func (s *Service) CreateMessageWithAttachments(message *models.TicketMessage, attachments []models.Attachment) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	})
}

// ApplyStatusTransition validates and applies a status change to an already
// loaded ticket. It returns changed=false (and no error) when the ticket is
// already in the target status, and ErrInvalidStatusTransition when the
// transition is not allowed.
func (s *Service) ApplyStatusTransition(ticket models.Ticket, nextStatus string) (bool, error) {
	current := NormalizeStatus(ticket.Status)
	next := NormalizeStatus(nextStatus)
	if current == next {
		return false, nil
	}
	if !CanTransition(current, next) {
		return false, ErrInvalidStatusTransition
	}

	now := time.Now()
	updates := map[string]any{
		"status": next,
	}
	if next == "in_progress" {
		updates["in_progress_at"] = now
	}
	if next == "resolved" {
		updates["resolved_at"] = now
	}
	if err := s.DB.Model(&models.Ticket{}).Where("id = ?", ticket.ID).Updates(updates).Error; err != nil {
		return false, err
	}
	return true, nil
}

// ListForOrg returns org-scoped tickets and the total count for the filter.
func (s *Service) ListForOrg(orgID, status string, limit, offset int) ([]TicketListItem, int64, error) {
	countDB := s.DB.Model(&models.Ticket{}).Where("org_id = ?", orgID)
	if status != "" {
		countDB = countDB.Where("status = ?", status)
	}
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db := s.DB.Table("tickets t").
		Select(`t.id, t.title, t.status, t.assignee_type, t.assignee_user_id, t.created_at,
		        u.email as created_by_email, au.email as assignee_email,
		        (SELECT COUNT(*) FROM attachments ta WHERE ta.owner_type = 'ticket' AND ta.owner_id = t.id) as attachment_count`).
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.org_id = ?", orgID)
	if status != "" {
		db = db.Where("t.status = ?", status)
	}

	var items []TicketListItem
	if err := db.Order("t.created_at desc").Limit(limit).Offset(offset).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetDetailForOrg loads an org-scoped ticket detail row.
func (s *Service) GetDetailForOrg(orgID, ticketID string) (TicketDetailRow, error) {
	db := s.DB.Table("tickets t").
		Select(`t.id, t.org_id, t.created_by, t.title, t.description, t.status, t.assignee_type, t.assignee_user_id,
		        t.in_progress_at, t.resolved_at, t.created_at, t.updated_at,
		        u.email as created_by_email, au.email as assignee_email`).
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.id = ? AND t.org_id = ?", ticketID, orgID)

	var row TicketDetailRow
	if err := db.Take(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// ListSystem returns system-wide tickets and the total count for the filter.
func (s *Service) ListSystem(orgID, status, q string, limit, offset int) ([]SystemTicketListItem, int64, error) {
	countDB := s.DB.Model(&models.Ticket{})
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
		return nil, 0, err
	}

	db := s.DB.Table("tickets t").
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

	var items []SystemTicketListItem
	if err := db.Order("t.created_at desc").Limit(limit).Offset(offset).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetDetailSystem loads a system (admin) ticket detail row including org name.
func (s *Service) GetDetailSystem(ticketID string) (TicketDetailRow, error) {
	db := s.DB.Table("tickets t").
		Select(`t.id, t.org_id, o.name as org_name, t.created_by, t.title, t.description, t.status, t.assignee_type, t.assignee_user_id,
		        t.in_progress_at, t.resolved_at, t.created_at, t.updated_at,
		        u.email as created_by_email, au.email as assignee_email`).
		Joins("LEFT JOIN orgs o ON o.id = t.org_id").
		Joins("LEFT JOIN users u ON u.id = t.created_by").
		Joins("LEFT JOIN users au ON au.id = t.assignee_user_id").
		Where("t.id = ?", ticketID)

	var row TicketDetailRow
	if err := db.Take(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// PurgeTickets collects the storage paths for the given tickets and their
// messages, then deletes the message rows, attachment rows, and ticket rows in a
// single transaction. The returned paths must be removed from object storage by
// the caller.
func (s *Service) PurgeTickets(ticketIDs []string) ([]string, error) {
	attachmentPaths, err := attachment.LoadStoragePaths(s.DB, attachment.OwnerTicket, ticketIDs)
	if err != nil {
		return nil, err
	}

	var messageIDs []string
	if err := s.DB.Model(&models.TicketMessage{}).Where("ticket_id IN ?", ticketIDs).Pluck("id", &messageIDs).Error; err != nil {
		return nil, err
	}

	var messageAttachmentPaths []string
	if len(messageIDs) > 0 {
		messageAttachmentPaths, err = attachment.LoadStoragePaths(s.DB, attachment.OwnerTicketMessage, messageIDs)
		if err != nil {
			return nil, err
		}
	}

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if len(messageIDs) > 0 {
			if err := attachment.DeleteByOwners(tx, attachment.OwnerTicketMessage, messageIDs); err != nil {
				return err
			}
		}
		if err := tx.Where("ticket_id IN ?", ticketIDs).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := attachment.DeleteByOwners(tx, attachment.OwnerTicket, ticketIDs); err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ticketIDs).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return append(attachmentPaths, messageAttachmentPaths...), nil
}

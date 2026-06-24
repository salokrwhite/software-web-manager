package feedback

import (
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FeedbackListItem is one row of the org-scoped feedback list.
type FeedbackListItem struct {
	ID              string     `json:"id"`
	Content         string     `json:"content"`
	Rating          *int       `json:"rating"`
	Contact         string     `json:"contact"`
	DeviceID        string     `json:"device_id"`
	AppVersion      string     `json:"app_version"`
	ChannelCode     string     `json:"channel_code"`
	Status          string     `json:"status"`
	InternalNote    string     `json:"internal_note"`
	HandledBy       *string    `json:"handled_by"`
	HandledAt       *time.Time `json:"handled_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AttachmentCount int64      `json:"attachment_count"`
}

// ListFilter carries the parsed filter/pagination inputs for a feedback list
// query. StatusProvided distinguishes an explicit status filter from the default.
type ListFilter struct {
	Keyword        string
	Rating         *int
	Status         string
	StatusProvided bool
	HasAttachment  string
	FromTime       *time.Time
	ToTime         *time.Time
	Limit          int
	Offset         int
}

// UpdateInput carries the fields a feedback update may change.
type UpdateInput struct {
	Status       *string
	InternalNote *string
	UserID       string
}

// DefaultChannelCode returns the code of the app's default channel, or "" when
// none exists.
func (s *Service) DefaultChannelCode(appID uuid.UUID) string {
	var channel models.Channel
	if err := s.DB.Where("app_id = ? AND is_default = true", appID).First(&channel).Error; err == nil {
		return channel.Code
	}
	return ""
}

// CreateWithAttachments persists a feedback row and any attachment rows in one
// transaction.
func (s *Service) CreateWithAttachments(feedback *models.Feedback, attachments []models.Attachment) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(feedback).Error; err != nil {
			return err
		}
		if len(attachments) > 0 {
			return tx.Create(&attachments).Error
		}
		return nil
	})
}

// GetForApp loads a feedback row scoped to the given org and app, returning the
// underlying gorm error when missing.
func (s *Service) GetForApp(orgID, appID, feedbackID string) (models.Feedback, error) {
	var feedback models.Feedback
	if err := s.DB.Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).First(&feedback).Error; err != nil {
		return feedback, err
	}
	return feedback, nil
}

// List returns the filtered, paginated feedback rows, the total count, and the
// per-status counts (across the filter excluding status).
func (s *Service) List(orgID, appID string, f ListFilter) ([]FeedbackListItem, int64, map[string]int64, error) {
	applyFeedbackFilters := func(db *gorm.DB, includeStatus bool) *gorm.DB {
		db = db.Where("org_id = ? AND app_id = ?", orgID, appID)
		if f.Keyword != "" {
			db = db.Where("(content LIKE ? OR contact LIKE ?)", "%"+f.Keyword+"%", "%"+f.Keyword+"%")
		}
		if f.Rating != nil {
			db = db.Where("rating = ?", *f.Rating)
		}
		if f.FromTime != nil {
			db = db.Where("created_at >= ?", *f.FromTime)
		}
		if f.ToTime != nil {
			db = db.Where("created_at <= ?", *f.ToTime)
		}
		switch f.HasAttachment {
		case "true", "1", "yes":
			db = db.Where("EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = feedbacks.id)")
		case "false", "0", "no":
			db = db.Where("NOT EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = feedbacks.id)")
		}
		if includeStatus && f.StatusProvided {
			db = db.Where("status = ?", f.Status)
		}
		return db
	}

	countDB := applyFeedbackFilters(s.DB.Model(&models.Feedback{}), true)
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, nil, err
	}

	type statusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusRows []statusCount
	statusDB := applyFeedbackFilters(s.DB.Model(&models.Feedback{}), false)
	if err := statusDB.Select("status, COUNT(*) as count").Group("status").Scan(&statusRows).Error; err != nil {
		return nil, 0, nil, err
	}
	statusCounts := map[string]int64{"open": 0, "processing": 0, "resolved": 0, "closed": 0}
	for _, row := range statusRows {
		statusCounts[NormalizeStatus(row.Status)] = row.Count
	}

	db := s.DB.Table("feedbacks f").
		Select(`f.id, f.content, f.rating, f.contact, f.device_id, f.app_version, f.channel_code,
		        f.status, f.internal_note, CAST(f.handled_by AS CHAR) as handled_by, f.handled_at, f.created_at, f.updated_at,
		        (SELECT COUNT(*) FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id) as attachment_count`).
		Where("f.org_id = ? AND f.app_id = ?", orgID, appID)
	if f.Keyword != "" {
		db = db.Where("(f.content LIKE ? OR f.contact LIKE ?)", "%"+f.Keyword+"%", "%"+f.Keyword+"%")
	}
	if f.Rating != nil {
		db = db.Where("f.rating = ?", *f.Rating)
	}
	if f.FromTime != nil {
		db = db.Where("f.created_at >= ?", *f.FromTime)
	}
	if f.ToTime != nil {
		db = db.Where("f.created_at <= ?", *f.ToTime)
	}
	if f.StatusProvided {
		db = db.Where("f.status = ?", f.Status)
	}
	switch f.HasAttachment {
	case "true", "1", "yes":
		db = db.Where("EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id)")
	case "false", "0", "no":
		db = db.Where("NOT EXISTS (SELECT 1 FROM attachments fa WHERE fa.owner_type = 'feedback' AND fa.owner_id = f.id)")
	}

	var items []FeedbackListItem
	if err := db.Order("f.updated_at desc, f.created_at desc").Limit(f.Limit).Offset(f.Offset).Scan(&items).Error; err != nil {
		return nil, 0, nil, err
	}
	return items, total, statusCounts, nil
}

// Update validates and applies a feedback update, returning the reloaded row.
// It returns ErrInvalidStatus / ErrInternalNoteTooLong / ErrNoUpdates for invalid
// input and ErrNotFound when no matching feedback row exists.
func (s *Service) Update(orgID, appID, feedbackID string, in UpdateInput) (models.Feedback, error) {
	updates := map[string]interface{}{}
	statusChanged := false
	if in.Status != nil {
		status := NormalizeStatus(*in.Status)
		if !IsValidStatus(status) {
			return models.Feedback{}, ErrInvalidStatus
		}
		updates["status"] = status
		statusChanged = true
	}
	if in.InternalNote != nil {
		note := strings.TrimSpace(*in.InternalNote)
		if len(note) > 5000 {
			return models.Feedback{}, ErrInternalNoteTooLong
		}
		updates["internal_note"] = note
	}
	if len(updates) == 0 {
		return models.Feedback{}, ErrNoUpdates
	}
	now := time.Now()
	if statusChanged {
		updates["handled_at"] = &now
		if parsedUserID, err := uuid.Parse(in.UserID); err == nil {
			updates["handled_by"] = &parsedUserID
		}
	}
	updates["updated_at"] = now

	result := s.DB.Model(&models.Feedback{}).
		Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).
		Updates(updates)
	if result.Error != nil {
		return models.Feedback{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.Feedback{}, ErrNotFound
	}

	var feedback models.Feedback
	if err := s.DB.Where("id = ? AND org_id = ? AND app_id = ?", feedbackID, orgID, appID).First(&feedback).Error; err != nil {
		return models.Feedback{}, err
	}
	return feedback, nil
}

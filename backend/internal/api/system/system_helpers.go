package system

import (
	"encoding/json"
	"strings"
	"time"
)

type auditNoteRow struct {
	TargetID  string    `gorm:"column:target_id"`
	AfterJSON []byte    `gorm:"column:after_json"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func extractAuditNote(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	noteRaw, ok := payload["note"]
	if !ok {
		return ""
	}
	note, ok := noteRaw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(note)
}

func (h *Handler) LoadSubmitNotes(targetType, action string, ids []string) map[string]string {
	notes := make(map[string]string)
	if h == nil || h.DB == nil || len(ids) == 0 {
		return notes
	}
	var rows []auditNoteRow
	if err := h.DB.Table("audit_logs").
		Select("target_id, after_json, created_at").
		Where("target_type = ? AND action = ? AND target_id IN ?", targetType, action, ids).
		Order("created_at desc").
		Scan(&rows).Error; err != nil {
		return notes
	}
	for _, row := range rows {
		if _, exists := notes[row.TargetID]; exists {
			continue
		}
		note := extractAuditNote(row.AfterJSON)
		if note != "" {
			notes[row.TargetID] = note
		}
	}
	return notes
}

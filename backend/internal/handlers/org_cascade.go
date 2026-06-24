package handlers

import (
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// DeleteOrgCascade removes an org and all of its dependent records within tx.
func DeleteOrgCascade(tx *gorm.DB, orgID string) error {
	if err := tx.Where("org_id = ?", orgID).Delete(&models.AuditLog{}).Error; err != nil {
		return err
	}
	if err := tx.Where("org_id = ?", orgID).Delete(&models.Event{}).Error; err != nil {
		return err
	}
	if tx.Migrator().HasTable(&models.OrgJoinRequest{}) {
		if err := tx.Where("org_id = ?", orgID).Delete(&models.OrgJoinRequest{}).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable(&models.ReleaseTemplate{}) {
		if err := tx.Where("org_id = ?", orgID).Delete(&models.ReleaseTemplate{}).Error; err != nil {
			return err
		}
	}
	var ticketIDs []string
	if err := tx.Model(&models.Ticket{}).Where("org_id = ?", orgID).Pluck("id", &ticketIDs).Error; err != nil {
		return err
	}
	var ticketMessageIDs []string
	if err := tx.Model(&models.TicketMessage{}).Where("org_id = ?", orgID).Pluck("id", &ticketMessageIDs).Error; err != nil {
		return err
	}
	if len(ticketMessageIDs) > 0 {
		if err := DeleteAttachmentsByOwners(tx, AttachmentOwnerTicketMessage, ticketMessageIDs); err != nil {
			return err
		}
	}
	if len(ticketIDs) > 0 {
		if err := DeleteAttachmentsByOwners(tx, AttachmentOwnerTicket, ticketIDs); err != nil {
			return err
		}
	}
	if err := tx.Where("org_id = ?", orgID).Delete(&models.TicketMessage{}).Error; err != nil {
		return err
	}
	if err := tx.Where("org_id = ?", orgID).Delete(&models.Ticket{}).Error; err != nil {
		return err
	}
	var appIDs []string
	if err := tx.Model(&models.App{}).Where("org_id = ?", orgID).Pluck("id", &appIDs).Error; err != nil {
		return err
	}
	if len(appIDs) > 0 {
		if tx.Migrator().HasTable(&models.Feedback{}) {
			var feedbackIDs []string
			if err := tx.Model(&models.Feedback{}).Where("app_id IN ?", appIDs).Pluck("id", &feedbackIDs).Error; err != nil {
				return err
			}
			if len(feedbackIDs) > 0 {
				if err := DeleteAttachmentsByOwners(tx, AttachmentOwnerFeedback, feedbackIDs); err != nil {
					return err
				}
			}
			if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Feedback{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("app_id IN ?", appIDs).Delete(&models.DailyMetric{}).Error; err != nil {
			return err
		}
		if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Device{}).Error; err != nil {
			return err
		}
		var releaseIDs []string
		if err := tx.Model(&models.Release{}).Where("app_id IN ?", appIDs).Pluck("id", &releaseIDs).Error; err != nil {
			return err
		}
		if len(releaseIDs) > 0 {
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.Artifact{}).Error; err != nil {
				return err
			}
			if err := tx.Where("release_id IN ?", releaseIDs).Delete(&models.ReleaseChannel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", releaseIDs).Delete(&models.Release{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("app_id IN ?", appIDs).Delete(&models.Channel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("scope_id IN ?", appIDs).Delete(&models.AppMember{}).Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable(&models.AppSecret{}) {
			if err := tx.Where("app_id IN ?", appIDs).Delete(&models.AppSecret{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("id IN ?", appIDs).Delete(&models.App{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("scope_id = ?", orgID).Delete(&models.OrgMember{}).Error; err != nil {
		return err
	}
	if err := tx.Where("org_id = ?", orgID).Delete(&models.OrgInvite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("org_id = ?", orgID).Delete(&models.Attachment{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", orgID).Delete(&models.Org{}).Error
}

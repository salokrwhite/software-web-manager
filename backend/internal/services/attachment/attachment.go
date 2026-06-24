package attachment

import (
	"software-web-manager/backend/internal/models"

	"gorm.io/gorm"
)

// Owner types an attachment can belong to.
const (
	OwnerTicket                  = "ticket"
	OwnerTicketMessage           = "ticket_message"
	OwnerFeedback                = "feedback"
	OwnerOrgRegistrationMaterial = "org_registration_material"
)

// LoadStoragePaths returns the storage paths of attachments owned by the given
// owners (used to delete the underlying files after the rows are removed).
func LoadStoragePaths(tx *gorm.DB, ownerType string, ownerIDs []string) ([]string, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	var paths []string
	err := tx.Model(&models.Attachment{}).
		Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).
		Pluck("storage_path", &paths).Error
	return paths, err
}

// DeleteByOwners removes attachment rows owned by the given owners.
func DeleteByOwners(tx *gorm.DB, ownerType string, ownerIDs []string) error {
	if len(ownerIDs) == 0 {
		return nil
	}
	return tx.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Delete(&models.Attachment{}).Error
}

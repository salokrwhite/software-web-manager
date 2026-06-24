package channel

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// List returns the release channels for an app, newest published first.
func (s *Service) List(appID string) ([]ReleaseChannelItem, error) {
	var items []ReleaseChannelItem
	if err := s.DB.Raw(`
		SELECT rc.id, rc.release_id, rc.channel_id, c.code as channel_code, c.name as channel_name,
			r.version as release_version, r.status as release_status,
			rc.rollout_percent, rc.mandatory, rc.whitelist_json as whitelist, rc.status, rc.paused, rc.targeting_rules_json as targeting_rules, rc.region_rules_json as region_rules,
			rc.rollout_start_at, rc.rollout_end_at, rc.published_at
		FROM release_channels rc
		JOIN releases r ON r.id = rc.release_id
		JOIN channels c ON c.id = rc.channel_id
		WHERE c.app_id = ?
		ORDER BY rc.published_at DESC
	`, appID).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateInput carries the inputs for creating/publishing a release channel.
type CreateInput struct {
	AppID          string
	ReleaseID      string
	ChannelCode    string
	RolloutPercent int
	Paused         bool
	Whitelist      []string
	RolloutStartAt *time.Time
	RolloutEndAt   *time.Time
}

// CreateResult carries the persisted release channel and the context needed to
// emit a client update.
type CreateResult struct {
	ReleaseChannel models.ReleaseChannel
	Release        models.Release
	ChannelCode    string
	PublishedAt    time.Time
}

// Create publishes a release to a channel: it validates the rollout percent,
// release approval state, and channel, then upserts the release_channel row
// (preserving a scheduled reservation) and publishes an approved release.
func (s *Service) Create(input CreateInput) (*CreateResult, error) {
	rolloutPercent := input.RolloutPercent
	if rolloutPercent == 0 {
		rolloutPercent = 100
	}
	if rolloutPercent <= 0 || rolloutPercent > 100 {
		return nil, ErrInvalidRolloutPercent
	}

	var release models.Release
	if err := s.DB.Where("id = ?", input.ReleaseID).First(&release).Error; err != nil || release.AppID.String() != input.AppID {
		return nil, ErrReleaseNotFound
	}
	releaseStatus := strings.ToLower(strings.TrimSpace(release.Status))
	if releaseStatus != "approved" && releaseStatus != "published" {
		return nil, ErrReleaseNotApproved
	}

	var channel models.Channel
	channelCode := strings.ToLower(strings.TrimSpace(input.ChannelCode))
	if channelCode == "" {
		return nil, ErrChannelCodeRequired
	}
	if err := s.DB.Where("app_id = ? AND code = ?", input.AppID, channelCode).First(&channel).Error; err != nil {
		return nil, ErrChannelNotFound
	}

	whitelist := input.Whitelist
	if whitelist == nil {
		whitelist = []string{}
	}
	whitelistBytes, _ := json.Marshal(whitelist)
	publishedAt := time.Now()
	var relChannel models.ReleaseChannel
	keepScheduled := false

	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("release_id = ? AND channel_id = ?", release.ID, channel.ID).First(&relChannel).Error
		if err == nil {
			// 若该通道由发布模板预约中(scheduled)，保留其预约状态/激活时间/时间窗，
			// 只应用灰度参数，避免创建灰度策略时静默绕过预约直接上线。
			keepScheduled = strings.EqualFold(strings.TrimSpace(relChannel.Status), "scheduled")
			updates := map[string]interface{}{
				"rollout_percent": rolloutPercent,
				"paused":          input.Paused,
				"whitelist_json":  datatypes.JSON(whitelistBytes),
			}
			if keepScheduled {
				updates["status"] = "scheduled"
			} else {
				updates["status"] = "active"
				updates["rollout_start_at"] = input.RolloutStartAt
				updates["rollout_end_at"] = input.RolloutEndAt
				if relChannel.PublishedAt == nil {
					updates["published_at"] = publishedAt
				}
			}
			if err := tx.Model(&models.ReleaseChannel{}).Where("id = ?", relChannel.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", relChannel.ID).First(&relChannel).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			relChannel = models.ReleaseChannel{
				ID:             uuid.New(),
				ReleaseID:      release.ID,
				ChannelID:      channel.ID,
				RolloutPercent: rolloutPercent,
				Mandatory:      false,
				WhitelistJSON:  datatypes.JSON(whitelistBytes),
				Status:         "active",
				Paused:         input.Paused,
				RolloutStartAt: input.RolloutStartAt,
				RolloutEndAt:   input.RolloutEndAt,
				PublishedAt:    &publishedAt,
			}
			if err := tx.Create(&relChannel).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		if !keepScheduled && releaseStatus == "approved" {
			if err := tx.Model(&models.Release{}).Where("id = ?", release.ID).Updates(map[string]interface{}{
				"status":       "published",
				"published_at": publishedAt,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateResult{
		ReleaseChannel: relChannel,
		Release:        release,
		ChannelCode:    channelCode,
		PublishedAt:    publishedAt,
	}, nil
}

// LoadForUpdate loads a release channel and its release for an update, returning
// ErrReleaseChannelNotFound / ErrReleaseNotFound when missing.
func (s *Service) LoadForUpdate(relChannelID string) (models.ReleaseChannel, models.Release, error) {
	var relChannel models.ReleaseChannel
	if err := s.DB.Where("id = ?", relChannelID).First(&relChannel).Error; err != nil {
		return relChannel, models.Release{}, ErrReleaseChannelNotFound
	}
	var release models.Release
	if err := s.DB.Where("id = ?", relChannel.ReleaseID).First(&release).Error; err != nil {
		return relChannel, release, ErrReleaseNotFound
	}
	return relChannel, release, nil
}

// ApplyUpdate applies the given column updates to a release channel, then reloads
// it and resolves its channel code (for client-update emission).
func (s *Service) ApplyUpdate(relChannelID string, updates map[string]interface{}) (models.ReleaseChannel, string, error) {
	var relChannel models.ReleaseChannel
	if err := s.DB.Model(&models.ReleaseChannel{}).Where("id = ?", relChannelID).Updates(updates).Error; err != nil {
		return relChannel, "", err
	}
	_ = s.DB.Where("id = ?", relChannelID).First(&relChannel).Error
	channelCode := ""
	var channel models.Channel
	if err := s.DB.Where("id = ?", relChannel.ChannelID).First(&channel).Error; err == nil {
		channelCode = channel.Code
	}
	return relChannel, channelCode, nil
}

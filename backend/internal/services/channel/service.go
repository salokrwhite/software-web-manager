// Package channel provides release-channel data access and rollout logic that is
// independent of the HTTP layer (no gin, no response writing).
package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Sentinel errors returned by channel operations so the HTTP layer can map them
// to the right status code.
var (
	ErrInvalidRolloutPercent  = errors.New("invalid rollout_percent")
	ErrReleaseNotFound        = errors.New("release not found")
	ErrReleaseNotApproved     = errors.New("release not approved")
	ErrChannelCodeRequired    = errors.New("channel_code required")
	ErrChannelNotFound        = errors.New("channel not found")
	ErrReleaseChannelNotFound = errors.New("release channel not found")
)

// Service exposes release-channel queries and commands over a single database
// handle.
type Service struct {
	DB *gorm.DB
}

// NewService builds a channel service from a database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// ReleaseChannelItem is one row of the release-channel list.
type ReleaseChannelItem struct {
	ID             uuid.UUID      `json:"id"`
	ReleaseID      uuid.UUID      `json:"release_id"`
	ChannelID      uuid.UUID      `json:"channel_id"`
	ChannelCode    string         `json:"channel_code"`
	ChannelName    string         `json:"channel_name"`
	ReleaseVersion string         `json:"release_version"`
	ReleaseStatus  string         `json:"release_status"`
	RolloutPercent int            `json:"rollout_percent"`
	Mandatory      bool           `json:"mandatory"`
	Whitelist      datatypes.JSON `json:"whitelist" gorm:"column:whitelist"`
	Status         string         `json:"status"`
	Paused         bool           `json:"paused"`
	TargetingRules datatypes.JSON `json:"targeting_rules" gorm:"column:targeting_rules"`
	RegionRules    datatypes.JSON `json:"region_rules" gorm:"column:region_rules"`
	RolloutStartAt *time.Time     `json:"rollout_start_at"`
	RolloutEndAt   *time.Time     `json:"rollout_end_at"`
	PublishedAt    *time.Time     `json:"published_at"`
}

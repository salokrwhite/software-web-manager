package client

import (
	"encoding/json"
	"strings"

	"software-web-manager/backend/internal/version"

	"gorm.io/datatypes"
)

type targetingRules struct {
	OSVersions []string `json:"os_versions"`
	UserIDs    []string `json:"user_ids"`
	DeviceIDs  []string `json:"device_ids"`
	Platforms  []string `json:"platforms"`
	Archs      []string `json:"archs"`
	MinVersion string   `json:"min_version"`
	MaxVersion string   `json:"max_version"`
}

func matchesTargetingRules(raw datatypes.JSON, req updateCheckRequest, attrs map[string]string) bool {
	if len(raw) == 0 {
		return true
	}
	var rules targetingRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return false
	}
	platform := req.Platform
	if platform == "" {
		platform = attrs["platform"]
	}
	arch := req.Arch
	if arch == "" {
		arch = attrs["arch"]
	}
	if len(rules.OSVersions) > 0 && !matchList(attrs["os_version"], rules.OSVersions) {
		return false
	}
	if len(rules.UserIDs) > 0 && !matchList(attrs["user_id"], rules.UserIDs) {
		return false
	}
	if len(rules.DeviceIDs) > 0 && !matchList(req.DeviceID, rules.DeviceIDs) {
		return false
	}
	if len(rules.Platforms) > 0 && !matchList(platform, rules.Platforms) {
		return false
	}
	if len(rules.Archs) > 0 && !matchList(arch, rules.Archs) {
		return false
	}
	if rules.MinVersion != "" && req.CurrentVersion != "" && version.CompareVersion(req.CurrentVersion, rules.MinVersion) < 0 {
		return false
	}
	if rules.MaxVersion != "" && req.CurrentVersion != "" && version.CompareVersion(req.CurrentVersion, rules.MaxVersion) > 0 {
		return false
	}
	if (rules.MinVersion != "" || rules.MaxVersion != "") && req.CurrentVersion == "" {
		return false
	}
	return true
}

func matchList(value string, list []string) bool {
	if value == "" {
		return false
	}
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), value) {
			return true
		}
	}
	return false
}

func isNewer(currentVersion string, currentCode *int, releaseVersion string, releaseCode *int) bool {
	if currentCode != nil && releaseCode != nil {
		return *releaseCode > *currentCode
	}
	return version.CompareVersion(releaseVersion, currentVersion) > 0
}

func isSameVersion(currentVersion string, currentCode *int, releaseVersion string, releaseCode *int) bool {
	if currentCode != nil && releaseCode != nil {
		return *releaseCode == *currentCode
	}
	current := strings.TrimSpace(currentVersion)
	release := strings.TrimSpace(releaseVersion)
	if current == "" || release == "" {
		return false
	}
	return strings.EqualFold(current, release)
}

func isWhitelisted(deviceID string, whitelist datatypes.JSON) bool {
	if deviceID == "" || len(whitelist) == 0 {
		return false
	}
	var list []string
	if err := json.Unmarshal(whitelist, &list); err != nil {
		return false
	}
	for _, v := range list {
		if v == deviceID {
			return true
		}
	}
	return false
}

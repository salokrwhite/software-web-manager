package system

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// ValidationError marks a user-facing (HTTP 400) settings error. Callers that map
// service results to HTTP can type-assert against it to distinguish client errors
// from internal (HTTP 500) failures.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// ServiceStatusComponent is a single component on the public service-status page.
type ServiceStatusComponent struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// ServiceStatusIncident is a single incident on the public service-status page.
type ServiceStatusIncident struct {
	Title       string `json:"title"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Description string `json:"description,omitempty"`
}

// SettingsResponse is the serialized view of all system settings.
type SettingsResponse struct {
	SiteName                    string   `json:"site_name"`
	AllowUserRegister           bool     `json:"allow_user_register"`
	AllowEnterpriseRegister     bool     `json:"allow_enterprise_register"`
	MailActivationEnabled       bool     `json:"mail_activation_enabled"`
	OrgPlanTypes                []string `json:"org_plan_types"`
	HomePageAnnouncementEnabled bool     `json:"home_page_announcement_enabled"`
	HomePageAnnouncementContent string   `json:"home_page_announcement_content"`
	AppsPageAnnouncementEnabled bool     `json:"apps_page_announcement_enabled"`
	AppsPageAnnouncementContent string   `json:"apps_page_announcement_content"`
	// Deprecated: kept for backward compatibility.
	PageAnnouncementEnabled bool `json:"page_announcement_enabled"`
	// Deprecated: kept for backward compatibility.
	PageAnnouncementContent   string                   `json:"page_announcement_content"`
	SMTPSenderName            string                   `json:"smtp_sender_name"`
	SMTPSenderEmail           string                   `json:"smtp_sender_email"`
	SMTPHost                  string                   `json:"smtp_host"`
	SMTPPort                  int                      `json:"smtp_port"`
	SMTPUsername              string                   `json:"smtp_username"`
	SMTPConnTTLSeconds        int                      `json:"smtp_conn_ttl_seconds"`
	SMTPForceSSL              bool                     `json:"smtp_force_ssl"`
	SMTPPasswordConfigured    bool                     `json:"smtp_password_configured"`
	RegisterEmailCodeTemplate string                   `json:"register_email_code_template"`
	ServiceStatusOverall      string                   `json:"service_status_overall_status"`
	ServiceStatusMessage      string                   `json:"service_status_overall_message"`
	ServiceStatusAnnouncement string                   `json:"service_status_announcement"`
	ServiceStatusComponents   []ServiceStatusComponent `json:"service_status_components"`
	ServiceStatusIncidents    []ServiceStatusIncident  `json:"service_status_incidents"`
	ServiceStatusUpdatedAt    string                   `json:"service_status_updated_at"`
	SSOEnabled                bool                     `json:"sso_enabled"`
	SSODisplayName            string                   `json:"sso_display_name"`
	SSOIssuer                 string                   `json:"sso_issuer"`
	SSOAuthorizeEndpoint      string                   `json:"sso_authorize_endpoint"`
	SSOTokenEndpoint          string                   `json:"sso_token_endpoint"`
	SSOUserinfoEndpoint       string                   `json:"sso_userinfo_endpoint"`
	SSOJWKSURI                string                   `json:"sso_jwks_uri"`
	SSOClientID               string                   `json:"sso_client_id"`
	SSOScopes                 string                   `json:"sso_scopes"`
	SSORedirectURI            string                   `json:"sso_redirect_uri"`
	SSOClientSecretConfigured bool                     `json:"sso_client_secret_configured"`
}

// UpdateSettingsRequest carries optional fields; only non-nil fields are applied.
type UpdateSettingsRequest struct {
	SiteName                    *string   `json:"site_name"`
	AllowUserRegister           *bool     `json:"allow_user_register"`
	AllowEnterpriseRegister     *bool     `json:"allow_enterprise_register"`
	MailActivationEnabled       *bool     `json:"mail_activation_enabled"`
	OrgPlanTypes                *[]string `json:"org_plan_types"`
	HomePageAnnouncementEnabled *bool     `json:"home_page_announcement_enabled"`
	HomePageAnnouncementContent *string   `json:"home_page_announcement_content"`
	AppsPageAnnouncementEnabled *bool     `json:"apps_page_announcement_enabled"`
	AppsPageAnnouncementContent *string   `json:"apps_page_announcement_content"`
	// Deprecated: kept for backward compatibility.
	PageAnnouncementEnabled *bool `json:"page_announcement_enabled"`
	// Deprecated: kept for backward compatibility.
	PageAnnouncementContent   *string                   `json:"page_announcement_content"`
	SMTPSenderName            *string                   `json:"smtp_sender_name"`
	SMTPSenderEmail           *string                   `json:"smtp_sender_email"`
	SMTPHost                  *string                   `json:"smtp_host"`
	SMTPPort                  *int                      `json:"smtp_port"`
	SMTPUsername              *string                   `json:"smtp_username"`
	SMTPPassword              *string                   `json:"smtp_password"`
	SMTPConnTTLSeconds        *int                      `json:"smtp_conn_ttl_seconds"`
	SMTPForceSSL              *bool                     `json:"smtp_force_ssl"`
	RegisterEmailCodeTemplate *string                   `json:"register_email_code_template"`
	ServiceStatusOverall      *string                   `json:"service_status_overall_status"`
	ServiceStatusMessage      *string                   `json:"service_status_overall_message"`
	ServiceStatusAnnouncement *string                   `json:"service_status_announcement"`
	ServiceStatusComponents   *[]ServiceStatusComponent `json:"service_status_components"`
	ServiceStatusIncidents    *[]ServiceStatusIncident  `json:"service_status_incidents"`
	SSOEnabled                *bool                     `json:"sso_enabled"`
	SSODisplayName            *string                   `json:"sso_display_name"`
	SSOIssuer                 *string                   `json:"sso_issuer"`
	SSOAuthorizeEndpoint      *string                   `json:"sso_authorize_endpoint"`
	SSOTokenEndpoint          *string                   `json:"sso_token_endpoint"`
	SSOUserinfoEndpoint       *string                   `json:"sso_userinfo_endpoint"`
	SSOJWKSURI                *string                   `json:"sso_jwks_uri"`
	SSOClientID               *string                   `json:"sso_client_id"`
	SSOClientSecret           *string                   `json:"sso_client_secret"`
	SSOScopes                 *string                   `json:"sso_scopes"`
	SSORedirectURI            *string                   `json:"sso_redirect_uri"`
}

// GetBool reads a boolean setting, falling back to the default when missing/blank/invalid.
func GetBool(items map[string]models.SystemSetting, key string, defaultValue bool) bool {
	item, ok := items[key]
	if !ok {
		return defaultValue
	}
	value := strings.TrimSpace(item.SettingValue)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(strings.ToLower(value))
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetString reads a string setting, falling back to the default when missing/blank.
func GetString(items map[string]models.SystemSetting, key string, defaultValue string) string {
	item, ok := items[key]
	if !ok {
		return defaultValue
	}
	value := strings.TrimSpace(item.SettingValue)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetInt reads an integer setting, falling back to the default when missing/blank/invalid.
func GetInt(items map[string]models.SystemSetting, key string, defaultValue int) int {
	item, ok := items[key]
	if !ok {
		return defaultValue
	}
	value := strings.TrimSpace(item.SettingValue)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// NormalizeOrgPlanTypes filters/dedups an org plan type list, defaulting to all plans.
func NormalizeOrgPlanTypes(values []string) []string {
	allowed := map[string]struct{}{
		"free":       {},
		"team":       {},
		"enterprise": {},
	}
	result := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, value := range values {
		v := strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[v]; !ok {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	if len(result) == 0 {
		return []string{"free", "team", "enterprise"}
	}
	return result
}

// NormalizeOrgPlanValue lower-cases and trims a plan value.
func NormalizeOrgPlanValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// IsAllowedOrgPlan reports whether value is among the configured plan types.
func IsAllowedOrgPlan(value string, planTypes []string) bool {
	normalized := NormalizeOrgPlanValue(value)
	for _, item := range planTypes {
		if normalized == NormalizeOrgPlanValue(item) {
			return true
		}
	}
	return false
}

func getOrgPlanTypesSetting(items map[string]models.SystemSetting) []string {
	item, ok := items[SettingOrgPlanTypesKey]
	if !ok {
		return NormalizeOrgPlanTypes(nil)
	}
	raw := strings.TrimSpace(item.SettingValue)
	if raw == "" {
		return NormalizeOrgPlanTypes(nil)
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return NormalizeOrgPlanTypes(values)
	}
	parts := strings.Split(raw, ",")
	return NormalizeOrgPlanTypes(parts)
}

func defaultServiceStatusComponents() []ServiceStatusComponent {
	return []ServiceStatusComponent{
		{Name: "Web 控制台", Status: "operational", Description: "管理后台与控制台访问"},
		{Name: "API 服务", Status: "operational", Description: "开放 API 与管理 API"},
		{Name: "文件分发", Status: "operational", Description: "安装包与制品下载"},
	}
}

func normalizeServiceStatusOverall(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "operational", "ok", "healthy":
		return "operational"
	case "degraded", "partial", "degraded_performance":
		return "degraded"
	case "outage", "down", "major_outage":
		return "outage"
	case "maintenance":
		return "maintenance"
	default:
		return DefaultServiceStatusOverall
	}
}

func normalizeServiceComponentStatus(value string) string {
	return normalizeServiceStatusOverall(value)
}

func normalizeServiceIncidentStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "investigating":
		return "investigating"
	case "identified":
		return "identified"
	case "monitoring":
		return "monitoring"
	case "resolved":
		return "resolved"
	default:
		return "investigating"
	}
}

func validateServiceStatusMessage(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = DefaultServiceStatusMessage
	}
	if len([]rune(trimmed)) > MaxServiceStatusMessageLength {
		return "", errors.New("service_status_overall_message too long")
	}
	return trimmed, nil
}

func validateServiceStatusAnnouncement(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len([]rune(trimmed)) > MaxServiceStatusAnnouncementLength {
		return "", errors.New("service_status_announcement too long")
	}
	return trimmed, nil
}

func validateServiceStatusComponents(values []ServiceStatusComponent) ([]ServiceStatusComponent, error) {
	if len(values) > MaxServiceStatusItemsCount {
		return nil, errors.New("service_status_components invalid")
	}
	out := make([]ServiceStatusComponent, 0, len(values))
	for i := range values {
		name := strings.TrimSpace(values[i].Name)
		if name == "" {
			continue
		}
		if len([]rune(name)) > MaxServiceStatusTitleLength {
			return nil, errors.New("service_status_components invalid")
		}
		description := strings.TrimSpace(values[i].Description)
		if len([]rune(description)) > MaxServiceStatusDescriptionLength {
			return nil, errors.New("service_status_components invalid")
		}
		out = append(out, ServiceStatusComponent{
			Name:        name,
			Status:      normalizeServiceComponentStatus(values[i].Status),
			Description: description,
		})
	}
	return out, nil
}

func validateServiceStatusIncidents(values []ServiceStatusIncident) ([]ServiceStatusIncident, error) {
	if len(values) > MaxServiceStatusItemsCount {
		return nil, errors.New("service_status_incidents invalid")
	}
	out := make([]ServiceStatusIncident, 0, len(values))
	for i := range values {
		title := strings.TrimSpace(values[i].Title)
		if title == "" {
			continue
		}
		if len([]rune(title)) > MaxServiceStatusTitleLength {
			return nil, errors.New("service_status_incidents invalid")
		}
		description := strings.TrimSpace(values[i].Description)
		if len([]rune(description)) > MaxServiceStatusDescriptionLength {
			return nil, errors.New("service_status_incidents invalid")
		}
		startedAt, startErr := normalizeServiceStatusTime(values[i].StartedAt)
		if startErr != nil {
			return nil, errors.New("service_status_incidents invalid")
		}
		updatedAt, updateErr := normalizeServiceStatusTime(values[i].UpdatedAt)
		if updateErr != nil {
			return nil, errors.New("service_status_incidents invalid")
		}
		if len([]rune(startedAt)) > MaxServiceStatusTimeLength || len([]rune(updatedAt)) > MaxServiceStatusTimeLength {
			return nil, errors.New("service_status_incidents invalid")
		}
		out = append(out, ServiceStatusIncident{
			Title:       title,
			Status:      normalizeServiceIncidentStatus(values[i].Status),
			StartedAt:   startedAt,
			UpdatedAt:   updatedAt,
			Description: description,
		})
	}
	return out, nil
}

func normalizeServiceStatusTime(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	parseLayouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04Z07:00",
	}
	for _, layout := range parseLayouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.Format(serviceStatusTimeLayout), nil
		}
	}
	localLayouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range localLayouts {
		parsed, err := time.ParseInLocation(layout, trimmed, time.Local)
		if err == nil {
			return parsed.Format(serviceStatusTimeLayout), nil
		}
	}
	return "", errors.New("service_status_incidents invalid")
}

func getServiceStatusComponentsSetting(items map[string]models.SystemSetting) []ServiceStatusComponent {
	item, ok := items[SettingServiceStatusComponentsKey]
	if !ok || strings.TrimSpace(item.SettingValue) == "" {
		return defaultServiceStatusComponents()
	}
	var values []ServiceStatusComponent
	if err := json.Unmarshal([]byte(item.SettingValue), &values); err != nil {
		return defaultServiceStatusComponents()
	}
	normalized, err := validateServiceStatusComponents(values)
	if err != nil || len(normalized) == 0 {
		return defaultServiceStatusComponents()
	}
	return normalized
}

func getServiceStatusIncidentsSetting(items map[string]models.SystemSetting) []ServiceStatusIncident {
	item, ok := items[SettingServiceStatusIncidentsKey]
	if !ok || strings.TrimSpace(item.SettingValue) == "" {
		return []ServiceStatusIncident{}
	}
	var values []ServiceStatusIncident
	if err := json.Unmarshal([]byte(item.SettingValue), &values); err != nil {
		return []ServiceStatusIncident{}
	}
	normalized, err := validateServiceStatusIncidents(values)
	if err != nil {
		return []ServiceStatusIncident{}
	}
	return normalized
}

func buildServiceStatusUpdatedAt(raw string) string {
	now := time.Now()
	trimmed := strings.TrimSpace(raw)
	if isServiceStatusHeartbeatDue(trimmed, now) {
		return now.Format(time.RFC3339)
	}
	return trimmed
}

func isServiceStatusHeartbeatDue(raw string, now time.Time) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return true
	}
	return now.Sub(parsed) >= serviceStatusHeartbeatInterval
}

// BuildSettingsResponse maps a settings map into the serialized response shape.
func BuildSettingsResponse(items map[string]models.SystemSetting) SettingsResponse {
	siteName := DefaultSiteName
	if item, ok := items[SettingSiteNameKey]; ok {
		value := strings.TrimSpace(item.SettingValue)
		if value != "" {
			siteName = value
		}
	}
	smtpPort := GetInt(items, SettingSMTPPortKey, DefaultSMTPPort)
	if !validateSMTPPort(smtpPort) {
		smtpPort = DefaultSMTPPort
	}
	smtpConnTTLSeconds := GetInt(items, SettingSMTPConnTTLSecondsKey, DefaultSMTPConnTTLSeconds)
	if !validateSMTPConnTTLSeconds(smtpConnTTLSeconds) {
		smtpConnTTLSeconds = DefaultSMTPConnTTLSeconds
	}
	smtpPasswordConfigured := strings.TrimSpace(GetString(items, SettingSMTPPasswordKey, "")) != ""
	legacyPageAnnouncementEnabled := GetBool(items, SettingPageAnnouncementEnabledKey, DefaultPageAnnouncementEnabled)
	legacyPageAnnouncementContent := GetString(items, SettingPageAnnouncementContentKey, DefaultPageAnnouncementContent)
	homePageAnnouncementEnabled := GetBool(items, SettingHomePageAnnouncementEnabledKey, legacyPageAnnouncementEnabled)
	homePageAnnouncementContent := GetString(items, SettingHomePageAnnouncementContentKey, legacyPageAnnouncementContent)
	if strings.TrimSpace(homePageAnnouncementContent) == "" {
		homePageAnnouncementContent = DefaultHomePageAnnouncementContent
	}
	appsPageAnnouncementEnabled := GetBool(items, SettingAppsPageAnnouncementEnabledKey, legacyPageAnnouncementEnabled)
	appsPageAnnouncementContent := GetString(items, SettingAppsPageAnnouncementContentKey, legacyPageAnnouncementContent)
	if strings.TrimSpace(appsPageAnnouncementContent) == "" {
		appsPageAnnouncementContent = DefaultAppsPageAnnouncementContent
	}
	serviceStatusMessage := GetString(items, SettingServiceStatusMessageKey, DefaultServiceStatusMessage)
	if strings.TrimSpace(serviceStatusMessage) == "" {
		serviceStatusMessage = DefaultServiceStatusMessage
	}
	return SettingsResponse{
		SiteName:                    siteName,
		AllowUserRegister:           GetBool(items, SettingAllowUserRegisterKey, DefaultAllowUserRegister),
		AllowEnterpriseRegister:     GetBool(items, SettingAllowEnterpriseRegisterKey, DefaultAllowEnterpriseRegister),
		MailActivationEnabled:       GetBool(items, SettingMailActivationEnabledKey, DefaultMailActivationEnabled),
		OrgPlanTypes:                getOrgPlanTypesSetting(items),
		HomePageAnnouncementEnabled: homePageAnnouncementEnabled,
		HomePageAnnouncementContent: homePageAnnouncementContent,
		AppsPageAnnouncementEnabled: appsPageAnnouncementEnabled,
		AppsPageAnnouncementContent: appsPageAnnouncementContent,
		// Keep deprecated fields for old clients. It follows app-page settings now.
		PageAnnouncementEnabled:   appsPageAnnouncementEnabled,
		PageAnnouncementContent:   appsPageAnnouncementContent,
		SMTPSenderName:            GetString(items, SettingSMTPSenderNameKey, ""),
		SMTPSenderEmail:           GetString(items, SettingSMTPSenderEmailKey, ""),
		SMTPHost:                  GetString(items, SettingSMTPHostKey, ""),
		SMTPPort:                  smtpPort,
		SMTPUsername:              GetString(items, SettingSMTPUsernameKey, ""),
		SMTPConnTTLSeconds:        smtpConnTTLSeconds,
		SMTPForceSSL:              GetBool(items, SettingSMTPForceSSLKey, DefaultSMTPForceSSL),
		SMTPPasswordConfigured:    smtpPasswordConfigured,
		RegisterEmailCodeTemplate: GetString(items, SettingRegisterEmailCodeTemplateKey, DefaultRegisterEmailCodeTemplate),
		ServiceStatusOverall:      normalizeServiceStatusOverall(GetString(items, SettingServiceStatusOverallKey, DefaultServiceStatusOverall)),
		ServiceStatusMessage:      serviceStatusMessage,
		ServiceStatusAnnouncement: GetString(items, SettingServiceStatusAnnouncementKey, DefaultServiceStatusAnnouncement),
		ServiceStatusComponents:   getServiceStatusComponentsSetting(items),
		ServiceStatusIncidents:    getServiceStatusIncidentsSetting(items),
		ServiceStatusUpdatedAt:    buildServiceStatusUpdatedAt(GetString(items, SettingServiceStatusUpdatedAtKey, "")),
		SSOEnabled:                GetBool(items, SettingSSOEnabledKey, DefaultSSOEnabled),
		SSODisplayName:            GetString(items, SettingSSODisplayNameKey, DefaultSSODisplayName),
		SSOIssuer:                 GetString(items, SettingSSOIssuerKey, ""),
		SSOAuthorizeEndpoint:      GetString(items, SettingSSOAuthorizeEndpointKey, ""),
		SSOTokenEndpoint:          GetString(items, SettingSSOTokenEndpointKey, ""),
		SSOUserinfoEndpoint:       GetString(items, SettingSSOUserinfoEndpointKey, ""),
		SSOJWKSURI:                GetString(items, SettingSSOJWKSURIKey, ""),
		SSOClientID:               GetString(items, SettingSSOClientIDKey, ""),
		SSOScopes:                 GetString(items, SettingSSOScopesKey, DefaultSSOScopes),
		SSORedirectURI:            GetString(items, SettingSSORedirectURIKey, ""),
		SSOClientSecretConfigured: strings.TrimSpace(GetString(items, SettingSSOClientSecretKey, "")) != "",
	}
}

// RefreshServiceStatusHeartbeat refreshes the service-status timestamp if the heartbeat
// interval has elapsed, returning the (possibly updated) settings map.
func (s *Service) RefreshServiceStatusHeartbeat(items map[string]models.SystemSetting) map[string]models.SystemSetting {
	now := time.Now()
	current := GetString(items, SettingServiceStatusUpdatedAtKey, "")
	if !isServiceStatusHeartbeatDue(current, now) {
		return items
	}

	refreshedAt := now.Format(time.RFC3339)
	setting := models.SystemSetting{
		SettingKey:   SettingServiceStatusUpdatedAtKey,
		SettingValue: refreshedAt,
		ValueType:    "string",
		Description:  SettingDescription(SettingServiceStatusUpdatedAtKey),
	}
	if err := s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"setting_value": refreshedAt,
			"value_type":    "string",
			"description":   SettingDescription(SettingServiceStatusUpdatedAtKey),
			"updated_at":    now,
		}),
	}).Create(&setting).Error; err != nil {
		return items
	}

	if items == nil {
		items = map[string]models.SystemSetting{}
	}
	items[SettingServiceStatusUpdatedAtKey] = setting
	return items
}

// UpdateSettings validates and persists the supplied settings update, returning the
// resulting settings response. Client errors are *ValidationError; other errors are
// internal failures.
func (s *Service) UpdateSettings(req UpdateSettingsRequest, userUUID uuid.UUID) (SettingsResponse, error) {
	if !s.HasSettingsTable() {
		if err := s.DB.AutoMigrate(&models.SystemSetting{}); err != nil {
			return SettingsResponse{}, errors.New("failed to initialize system settings table")
		}
	}

	existingItems, err := s.ListSettings()
	if err != nil {
		return SettingsResponse{}, errors.New("failed to load system settings")
	}

	type settingUpdate struct {
		Value     string
		ValueType string
	}
	updates := map[string]settingUpdate{}
	if req.SiteName != nil {
		siteName := strings.TrimSpace(*req.SiteName)
		if siteName == "" {
			return SettingsResponse{}, &ValidationError{Message: "site_name required"}
		}
		if len([]rune(siteName)) > MaxSiteNameLength {
			return SettingsResponse{}, &ValidationError{Message: "site_name too long"}
		}
		updates[SettingSiteNameKey] = settingUpdate{Value: siteName, ValueType: "string"}
	}
	if req.AllowUserRegister != nil {
		updates[SettingAllowUserRegisterKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AllowUserRegister),
			ValueType: "bool",
		}
	}
	if req.AllowEnterpriseRegister != nil {
		updates[SettingAllowEnterpriseRegisterKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AllowEnterpriseRegister),
			ValueType: "bool",
		}
	}
	if req.MailActivationEnabled != nil {
		updates[SettingMailActivationEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.MailActivationEnabled),
			ValueType: "bool",
		}
	}
	if req.OrgPlanTypes != nil {
		normalized := NormalizeOrgPlanTypes(*req.OrgPlanTypes)
		if len(normalized) == 0 {
			return SettingsResponse{}, &ValidationError{Message: "org_plan_types invalid"}
		}
		payload, err := json.Marshal(normalized)
		if err != nil {
			return SettingsResponse{}, errors.New("failed to encode org_plan_types")
		}
		updates[SettingOrgPlanTypesKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
	}
	// Legacy announcement fields: keep compatibility for older clients.
	if req.PageAnnouncementEnabled != nil {
		value := strconv.FormatBool(*req.PageAnnouncementEnabled)
		updates[SettingPageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
		updates[SettingHomePageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
		updates[SettingAppsPageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
	}
	if req.PageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.PageAnnouncementContent)
		if len([]rune(content)) > MaxPageAnnouncementContentLength {
			return SettingsResponse{}, &ValidationError{Message: "page_announcement_content too long"}
		}
		updates[SettingPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
		updates[SettingHomePageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
		updates[SettingAppsPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.HomePageAnnouncementEnabled != nil {
		updates[SettingHomePageAnnouncementEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.HomePageAnnouncementEnabled),
			ValueType: "bool",
		}
	}
	if req.HomePageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.HomePageAnnouncementContent)
		if len([]rune(content)) > MaxPageAnnouncementContentLength {
			return SettingsResponse{}, &ValidationError{Message: "home_page_announcement_content too long"}
		}
		updates[SettingHomePageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.AppsPageAnnouncementEnabled != nil {
		updates[SettingAppsPageAnnouncementEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AppsPageAnnouncementEnabled),
			ValueType: "bool",
		}
	}
	if req.AppsPageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.AppsPageAnnouncementContent)
		if len([]rune(content)) > MaxPageAnnouncementContentLength {
			return SettingsResponse{}, &ValidationError{Message: "apps_page_announcement_content too long"}
		}
		updates[SettingAppsPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.RegisterEmailCodeTemplate != nil {
		content := strings.TrimSpace(*req.RegisterEmailCodeTemplate)
		if content == "" {
			content = DefaultRegisterEmailCodeTemplate
		}
		if len([]rune(content)) > MaxRegisterEmailCodeTemplateLength {
			return SettingsResponse{}, &ValidationError{Message: "register_email_code_template too long"}
		}
		updates[SettingRegisterEmailCodeTemplateKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	serviceStatusUpdated := false
	if req.ServiceStatusOverall != nil {
		updates[SettingServiceStatusOverallKey] = settingUpdate{
			Value:     normalizeServiceStatusOverall(*req.ServiceStatusOverall),
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusMessage != nil {
		messageValue, msgErr := validateServiceStatusMessage(*req.ServiceStatusMessage)
		if msgErr != nil {
			return SettingsResponse{}, &ValidationError{Message: msgErr.Error()}
		}
		updates[SettingServiceStatusMessageKey] = settingUpdate{
			Value:     messageValue,
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusAnnouncement != nil {
		announcementValue, annErr := validateServiceStatusAnnouncement(*req.ServiceStatusAnnouncement)
		if annErr != nil {
			return SettingsResponse{}, &ValidationError{Message: annErr.Error()}
		}
		updates[SettingServiceStatusAnnouncementKey] = settingUpdate{
			Value:     announcementValue,
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusComponents != nil {
		components, compErr := validateServiceStatusComponents(*req.ServiceStatusComponents)
		if compErr != nil {
			return SettingsResponse{}, &ValidationError{Message: compErr.Error()}
		}
		payload, marshalErr := json.Marshal(components)
		if marshalErr != nil {
			return SettingsResponse{}, errors.New("failed to encode service_status_components")
		}
		updates[SettingServiceStatusComponentsKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusIncidents != nil {
		incidents, incidentErr := validateServiceStatusIncidents(*req.ServiceStatusIncidents)
		if incidentErr != nil {
			return SettingsResponse{}, &ValidationError{Message: incidentErr.Error()}
		}
		payload, marshalErr := json.Marshal(incidents)
		if marshalErr != nil {
			return SettingsResponse{}, errors.New("failed to encode service_status_incidents")
		}
		updates[SettingServiceStatusIncidentsKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
		serviceStatusUpdated = true
	}
	if serviceStatusUpdated {
		updates[SettingServiceStatusUpdatedAtKey] = settingUpdate{
			Value:     time.Now().Format(time.RFC3339),
			ValueType: "string",
		}
	}
	hasSMTPUpdate := req.SMTPSenderName != nil ||
		req.SMTPSenderEmail != nil ||
		req.SMTPHost != nil ||
		req.SMTPPort != nil ||
		req.SMTPUsername != nil ||
		req.SMTPPassword != nil ||
		req.SMTPConnTTLSeconds != nil ||
		req.SMTPForceSSL != nil
	if hasSMTPUpdate {
		smtpCfg := s.SMTPConfigFromSettings(existingItems)
		password, configured, passwordErr := s.SMTPPasswordFromSettings(existingItems)
		if passwordErr != nil {
			return SettingsResponse{}, &ValidationError{Message: "failed to load smtp password"}
		}
		smtpCfg.Password = password
		if req.SMTPSenderName != nil {
			smtpCfg.SenderName = strings.TrimSpace(*req.SMTPSenderName)
		}
		if req.SMTPSenderEmail != nil {
			smtpCfg.SenderEmail = strings.TrimSpace(*req.SMTPSenderEmail)
		}
		if req.SMTPHost != nil {
			smtpCfg.Host = strings.TrimSpace(*req.SMTPHost)
		}
		if req.SMTPPort != nil {
			smtpCfg.Port = *req.SMTPPort
		}
		if req.SMTPUsername != nil {
			smtpCfg.Username = strings.TrimSpace(*req.SMTPUsername)
		}
		if req.SMTPConnTTLSeconds != nil {
			smtpCfg.ConnTTLSeconds = *req.SMTPConnTTLSeconds
		}
		if req.SMTPForceSSL != nil {
			smtpCfg.ForceSSL = *req.SMTPForceSSL
		}
		passwordUpdated := false
		if req.SMTPPassword != nil {
			nextPassword := strings.TrimSpace(*req.SMTPPassword)
			if nextPassword != "" {
				smtpCfg.Password = nextPassword
				passwordUpdated = true
			}
		}
		passwordRequired := !configured && strings.TrimSpace(smtpCfg.Password) == ""
		if err := ValidateSMTPConfig(smtpCfg, passwordRequired); err != nil {
			return SettingsResponse{}, &ValidationError{Message: err.Error()}
		}
		updates[SettingSMTPSenderNameKey] = settingUpdate{Value: smtpCfg.SenderName, ValueType: "string"}
		updates[SettingSMTPSenderEmailKey] = settingUpdate{Value: smtpCfg.SenderEmail, ValueType: "string"}
		updates[SettingSMTPHostKey] = settingUpdate{Value: smtpCfg.Host, ValueType: "string"}
		updates[SettingSMTPPortKey] = settingUpdate{Value: strconv.Itoa(smtpCfg.Port), ValueType: "int"}
		updates[SettingSMTPUsernameKey] = settingUpdate{Value: smtpCfg.Username, ValueType: "string"}
		updates[SettingSMTPConnTTLSecondsKey] = settingUpdate{Value: strconv.Itoa(smtpCfg.ConnTTLSeconds), ValueType: "int"}
		updates[SettingSMTPForceSSLKey] = settingUpdate{Value: strconv.FormatBool(smtpCfg.ForceSSL), ValueType: "bool"}
		if passwordUpdated {
			updates[SettingSMTPPasswordKey] = settingUpdate{Value: smtpCfg.Password, ValueType: "string"}
		}
	}
	if req.SSOEnabled != nil {
		updates[SettingSSOEnabledKey] = settingUpdate{Value: strconv.FormatBool(*req.SSOEnabled), ValueType: "bool"}
	}
	if req.SSODisplayName != nil {
		name := strings.TrimSpace(*req.SSODisplayName)
		if name == "" {
			name = DefaultSSODisplayName
		}
		if len([]rune(name)) > MaxSiteNameLength {
			return SettingsResponse{}, &ValidationError{Message: "sso_display_name too long"}
		}
		updates[SettingSSODisplayNameKey] = settingUpdate{Value: name, ValueType: "string"}
	}
	ssoEndpointFields := []struct {
		key   string
		value *string
	}{
		{SettingSSOIssuerKey, req.SSOIssuer},
		{SettingSSOAuthorizeEndpointKey, req.SSOAuthorizeEndpoint},
		{SettingSSOTokenEndpointKey, req.SSOTokenEndpoint},
		{SettingSSOUserinfoEndpointKey, req.SSOUserinfoEndpoint},
		{SettingSSOJWKSURIKey, req.SSOJWKSURI},
		{SettingSSOClientIDKey, req.SSOClientID},
		{SettingSSORedirectURIKey, req.SSORedirectURI},
	}
	for _, f := range ssoEndpointFields {
		if f.value == nil {
			continue
		}
		updates[f.key] = settingUpdate{Value: strings.TrimSpace(*f.value), ValueType: "string"}
	}
	if req.SSOScopes != nil {
		scopes := strings.Join(strings.Fields(strings.TrimSpace(*req.SSOScopes)), " ")
		if scopes == "" {
			scopes = DefaultSSOScopes
		}
		updates[SettingSSOScopesKey] = settingUpdate{Value: scopes, ValueType: "string"}
	}
	if req.SSOClientSecret != nil {
		// Only overwrite when a non-empty value is sent (mirrors SMTP password).
		if secret := strings.TrimSpace(*req.SSOClientSecret); secret != "" {
			updates[SettingSSOClientSecretKey] = settingUpdate{Value: secret, ValueType: "string"}
		}
	}

	if len(updates) == 0 {
		return SettingsResponse{}, &ValidationError{Message: "no settings to update"}
	}

	now := time.Now()
	for key, update := range updates {
		updatedBy := userUUID
		item := models.SystemSetting{
			SettingKey:   key,
			SettingValue: update.Value,
			ValueType:    update.ValueType,
			Description:  SettingDescription(key),
			UpdatedBy:    &updatedBy,
		}
		if err := s.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"setting_value": update.Value,
				"value_type":    update.ValueType,
				"description":   SettingDescription(key),
				"updated_by":    userUUID,
				"updated_at":    now,
			}),
		}).Create(&item).Error; err != nil {
			return SettingsResponse{}, errors.New("failed to save system settings")
		}
	}

	items, err := s.ListSettings()
	if err != nil {
		return SettingsResponse{}, errors.New("failed to load system settings")
	}
	return BuildSettingsResponse(items), nil
}

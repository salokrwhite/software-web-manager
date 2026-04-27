package handlers

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

const (
	systemSettingSiteNameKey                    = "site_name"
	systemSettingAllowUserRegisterKey           = "allow_user_register"
	systemSettingAllowEnterpriseRegisterKey     = "allow_enterprise_register"
	systemSettingMailActivationEnabledKey       = "mail_activation_enabled"
	systemSettingOrgPlanTypesKey                = "org_plan_types"
	systemSettingHomePageAnnouncementEnabledKey = "home_page_announcement_enabled"
	systemSettingHomePageAnnouncementContentKey = "home_page_announcement_content"
	systemSettingAppsPageAnnouncementEnabledKey = "apps_page_announcement_enabled"
	systemSettingAppsPageAnnouncementContentKey = "apps_page_announcement_content"
	// Deprecated: kept for backward compatibility.
	systemSettingPageAnnouncementEnabledKey = "page_announcement_enabled"
	// Deprecated: kept for backward compatibility.
	systemSettingPageAnnouncementContentKey   = "page_announcement_content"
	systemSettingSMTPSenderNameKey            = "smtp_sender_name"
	systemSettingSMTPSenderEmailKey           = "smtp_sender_email"
	systemSettingSMTPHostKey                  = "smtp_host"
	systemSettingSMTPPortKey                  = "smtp_port"
	systemSettingSMTPUsernameKey              = "smtp_username"
	systemSettingSMTPPasswordKey              = "smtp_password"
	systemSettingSMTPConnTTLSecondsKey        = "smtp_conn_ttl_seconds"
	systemSettingSMTPForceSSLKey              = "smtp_force_ssl"
	systemSettingRegisterEmailCodeTemplateKey = "register_email_code_template"
	systemSettingServiceStatusOverallKey      = "service_status_overall_status"
	systemSettingServiceStatusMessageKey      = "service_status_overall_message"
	systemSettingServiceStatusAnnouncementKey = "service_status_announcement"
	systemSettingServiceStatusComponentsKey   = "service_status_components"
	systemSettingServiceStatusIncidentsKey    = "service_status_incidents"
	systemSettingServiceStatusUpdatedAtKey    = "service_status_updated_at"
	defaultSiteName                           = "SWM 软件版本管理平台"
	defaultAllowUserRegister                  = true
	defaultAllowEnterpriseRegister            = true
	defaultMailActivationEnabled              = false
	defaultHomePageAnnouncementEnabled        = false
	defaultHomePageAnnouncementContent        = ""
	defaultAppsPageAnnouncementEnabled        = false
	defaultAppsPageAnnouncementContent        = ""
	// Deprecated: kept for backward compatibility.
	defaultPageAnnouncementEnabled = false
	// Deprecated: kept for backward compatibility.
	defaultPageAnnouncementContent     = ""
	defaultSMTPPort                    = 465
	defaultSMTPConnTTLSeconds          = 15
	defaultSMTPForceSSL                = true
	defaultRegisterEmailCodeTemplate   = "您好，\n\n您的注册验证码是 {{code}}，有效期 {{minutes}} 分钟。\n\n如果这不是您的操作，请忽略此邮件。\n\n{{site_name}}"
	defaultServiceStatusOverall        = "operational"
	defaultServiceStatusMessage        = "所有系统正常运行"
	defaultServiceStatusAnnouncement   = ""
	maxSiteNameLength                  = 100
	maxPageAnnouncementContentLength   = 500
	maxRegisterEmailCodeTemplateLength = 2000
	maxServiceStatusMessageLength      = 300
	maxServiceStatusAnnouncementLength = 1000
	maxServiceStatusItemsCount         = 20
	maxServiceStatusTitleLength        = 100
	maxServiceStatusDescriptionLength  = 500
	maxServiceStatusTimeLength         = 64
	serviceStatusHeartbeatInterval     = 5 * time.Minute
	serviceStatusTimeLayout            = "2006-01-02T15:04:05Z07:00"
)

type serviceStatusComponent struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

type serviceStatusIncident struct {
	Title       string `json:"title"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Description string `json:"description,omitempty"`
}

type systemSettingsResponse struct {
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
	ServiceStatusComponents   []serviceStatusComponent `json:"service_status_components"`
	ServiceStatusIncidents    []serviceStatusIncident  `json:"service_status_incidents"`
	ServiceStatusUpdatedAt    string                   `json:"service_status_updated_at"`
}

type updateSystemSettingsRequest struct {
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
	ServiceStatusComponents   *[]serviceStatusComponent `json:"service_status_components"`
	ServiceStatusIncidents    *[]serviceStatusIncident  `json:"service_status_incidents"`
}

type testSMTPRequest struct {
	ToEmail            string  `json:"to_email" binding:"required,email"`
	Subject            *string `json:"subject"`
	Body               *string `json:"body"`
	SMTPSenderName     *string `json:"smtp_sender_name"`
	SMTPSenderEmail    *string `json:"smtp_sender_email"`
	SMTPHost           *string `json:"smtp_host"`
	SMTPPort           *int    `json:"smtp_port"`
	SMTPUsername       *string `json:"smtp_username"`
	SMTPPassword       *string `json:"smtp_password"`
	SMTPConnTTLSeconds *int    `json:"smtp_conn_ttl_seconds"`
	SMTPForceSSL       *bool   `json:"smtp_force_ssl"`
}

type smtpConfig struct {
	SenderName     string
	SenderEmail    string
	Host           string
	Port           int
	Username       string
	Password       string
	ConnTTLSeconds int
	ForceSSL       bool
}

type smtpLoginAuth struct {
	username string
	password string
	step     int
}

func newSMTPLoginAuth(username, password string) smtp.Auth {
	return &smtpLoginAuth{username: username, password: password}
}

func (a *smtpLoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	a.step = 0
	return "LOGIN", []byte{}, nil
}

func (a *smtpLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	a.step++
	if a.step == 1 {
		return []byte(a.username), nil
	}
	if a.step == 2 {
		return []byte(a.password), nil
	}
	return nil, nil
}

func smtpAuth(client *smtp.Client, cfg smtpConfig) error {
	if strings.TrimSpace(cfg.Username) == "" {
		return nil
	}
	hasAuthMethods, methodsText := client.Extension("AUTH")
	methods := strings.ToUpper(strings.TrimSpace(methodsText))

	usePlain := !hasAuthMethods || strings.Contains(methods, "PLAIN")
	if usePlain {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
		return nil
	}
	if strings.Contains(methods, "LOGIN") {
		if err := client.Auth(newSMTPLoginAuth(cfg.Username, cfg.Password)); err != nil {
			return err
		}
		return nil
	}
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return client.Auth(auth)
}

func systemSettingDescription(key string) string {
	switch key {
	case systemSettingSiteNameKey:
		return "站点名称"
	case systemSettingAllowUserRegisterKey:
		return "允许新用户注册"
	case systemSettingAllowEnterpriseRegisterKey:
		return "允许企业用户注册"
	case systemSettingMailActivationEnabledKey:
		return "邮件激活"
	case systemSettingOrgPlanTypesKey:
		return "企业套餐类型"
	case systemSettingHomePageAnnouncementEnabledKey:
		return "首页公告启用"
	case systemSettingHomePageAnnouncementContentKey:
		return "首页公告内容"
	case systemSettingAppsPageAnnouncementEnabledKey:
		return "应用管理页面公告启用"
	case systemSettingAppsPageAnnouncementContentKey:
		return "应用管理页面公告内容"
	case systemSettingPageAnnouncementEnabledKey:
		return "页面公告启用"
	case systemSettingPageAnnouncementContentKey:
		return "页面公告内容"
	case systemSettingSMTPSenderNameKey:
		return "SMTP发件人名"
	case systemSettingSMTPSenderEmailKey:
		return "SMTP发件人邮箱"
	case systemSettingSMTPHostKey:
		return "SMTP服务器"
	case systemSettingSMTPPortKey:
		return "SMTP端口"
	case systemSettingSMTPUsernameKey:
		return "SMTP用户名"
	case systemSettingSMTPPasswordKey:
		return "SMTP密码"
	case systemSettingSMTPConnTTLSecondsKey:
		return "SMTP连接有效期(秒)"
	case systemSettingSMTPForceSSLKey:
		return "SMTP强制SSL"
	case systemSettingRegisterEmailCodeTemplateKey:
		return "注册邮箱验证码模板"
	case systemSettingServiceStatusOverallKey:
		return "服务状态总览状态"
	case systemSettingServiceStatusMessageKey:
		return "服务状态总览说明"
	case systemSettingServiceStatusAnnouncementKey:
		return "服务状态公告"
	case systemSettingServiceStatusComponentsKey:
		return "服务状态组件列表"
	case systemSettingServiceStatusIncidentsKey:
		return "服务状态事件列表"
	case systemSettingServiceStatusUpdatedAtKey:
		return "服务状态更新时间"
	default:
		return ""
	}
}

func normalizeOrgPlanTypes(values []string) []string {
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

func defaultServiceStatusComponents() []serviceStatusComponent {
	return []serviceStatusComponent{
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
		return defaultServiceStatusOverall
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
		trimmed = defaultServiceStatusMessage
	}
	if len([]rune(trimmed)) > maxServiceStatusMessageLength {
		return "", errors.New("service_status_overall_message too long")
	}
	return trimmed, nil
}

func validateServiceStatusAnnouncement(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len([]rune(trimmed)) > maxServiceStatusAnnouncementLength {
		return "", errors.New("service_status_announcement too long")
	}
	return trimmed, nil
}

func validateServiceStatusComponents(values []serviceStatusComponent) ([]serviceStatusComponent, error) {
	if len(values) > maxServiceStatusItemsCount {
		return nil, errors.New("service_status_components invalid")
	}
	out := make([]serviceStatusComponent, 0, len(values))
	for i := range values {
		name := strings.TrimSpace(values[i].Name)
		if name == "" {
			continue
		}
		if len([]rune(name)) > maxServiceStatusTitleLength {
			return nil, errors.New("service_status_components invalid")
		}
		description := strings.TrimSpace(values[i].Description)
		if len([]rune(description)) > maxServiceStatusDescriptionLength {
			return nil, errors.New("service_status_components invalid")
		}
		out = append(out, serviceStatusComponent{
			Name:        name,
			Status:      normalizeServiceComponentStatus(values[i].Status),
			Description: description,
		})
	}
	return out, nil
}

func validateServiceStatusIncidents(values []serviceStatusIncident) ([]serviceStatusIncident, error) {
	if len(values) > maxServiceStatusItemsCount {
		return nil, errors.New("service_status_incidents invalid")
	}
	out := make([]serviceStatusIncident, 0, len(values))
	for i := range values {
		title := strings.TrimSpace(values[i].Title)
		if title == "" {
			continue
		}
		if len([]rune(title)) > maxServiceStatusTitleLength {
			return nil, errors.New("service_status_incidents invalid")
		}
		description := strings.TrimSpace(values[i].Description)
		if len([]rune(description)) > maxServiceStatusDescriptionLength {
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
		if len([]rune(startedAt)) > maxServiceStatusTimeLength || len([]rune(updatedAt)) > maxServiceStatusTimeLength {
			return nil, errors.New("service_status_incidents invalid")
		}
		out = append(out, serviceStatusIncident{
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

func getServiceStatusComponentsSetting(items map[string]models.SystemSetting) []serviceStatusComponent {
	item, ok := items[systemSettingServiceStatusComponentsKey]
	if !ok || strings.TrimSpace(item.SettingValue) == "" {
		return defaultServiceStatusComponents()
	}
	var values []serviceStatusComponent
	if err := json.Unmarshal([]byte(item.SettingValue), &values); err != nil {
		return defaultServiceStatusComponents()
	}
	normalized, err := validateServiceStatusComponents(values)
	if err != nil || len(normalized) == 0 {
		return defaultServiceStatusComponents()
	}
	return normalized
}

func getServiceStatusIncidentsSetting(items map[string]models.SystemSetting) []serviceStatusIncident {
	item, ok := items[systemSettingServiceStatusIncidentsKey]
	if !ok || strings.TrimSpace(item.SettingValue) == "" {
		return []serviceStatusIncident{}
	}
	var values []serviceStatusIncident
	if err := json.Unmarshal([]byte(item.SettingValue), &values); err != nil {
		return []serviceStatusIncident{}
	}
	normalized, err := validateServiceStatusIncidents(values)
	if err != nil {
		return []serviceStatusIncident{}
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

func getOrgPlanTypesSetting(items map[string]models.SystemSetting) []string {
	item, ok := items[systemSettingOrgPlanTypesKey]
	if !ok {
		return normalizeOrgPlanTypes(nil)
	}
	raw := strings.TrimSpace(item.SettingValue)
	if raw == "" {
		return normalizeOrgPlanTypes(nil)
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return normalizeOrgPlanTypes(values)
	}
	parts := strings.Split(raw, ",")
	return normalizeOrgPlanTypes(parts)
}

func normalizeOrgPlanValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isAllowedOrgPlan(value string, planTypes []string) bool {
	normalized := normalizeOrgPlanValue(value)
	for _, item := range planTypes {
		if normalized == normalizeOrgPlanValue(item) {
			return true
		}
	}
	return false
}

func getBoolSetting(items map[string]models.SystemSetting, key string, defaultValue bool) bool {
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

func getStringSetting(items map[string]models.SystemSetting, key string, defaultValue string) string {
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

func getIntSetting(items map[string]models.SystemSetting, key string, defaultValue int) int {
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

func validateSMTPPort(port int) bool {
	return port >= 1 && port <= 65535
}

func validateSMTPConnTTLSeconds(ttl int) bool {
	return ttl >= 1 && ttl <= 86400
}

func parseSenderAddress(senderName, senderEmail string) string {
	if senderEmail == "" {
		return ""
	}
	addr := mail.Address{Name: senderName, Address: senderEmail}
	return addr.String()
}

func sanitizeSMTPError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "smtp test failed"
	}
	if len(text) > 240 {
		return "smtp test failed"
	}
	return text
}

func (h *Handler) getSMTPConfigFromSettings(items map[string]models.SystemSetting) smtpConfig {
	return smtpConfig{
		SenderName:     getStringSetting(items, systemSettingSMTPSenderNameKey, ""),
		SenderEmail:    getStringSetting(items, systemSettingSMTPSenderEmailKey, ""),
		Host:           getStringSetting(items, systemSettingSMTPHostKey, ""),
		Port:           getIntSetting(items, systemSettingSMTPPortKey, defaultSMTPPort),
		Username:       getStringSetting(items, systemSettingSMTPUsernameKey, ""),
		ConnTTLSeconds: getIntSetting(items, systemSettingSMTPConnTTLSecondsKey, defaultSMTPConnTTLSeconds),
		ForceSSL:       getBoolSetting(items, systemSettingSMTPForceSSLKey, defaultSMTPForceSSL),
	}
}

func (h *Handler) getSMTPPasswordFromSettings(items map[string]models.SystemSetting) (string, bool, error) {
	password := getStringSetting(items, systemSettingSMTPPasswordKey, "")
	if password == "" {
		return "", false, nil
	}
	return password, true, nil
}

func validateSMTPConfig(cfg smtpConfig, passwordRequired bool) error {
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("smtp_host required")
	}
	if !validateSMTPPort(cfg.Port) {
		return errors.New("smtp_port invalid")
	}
	if !validateSMTPConnTTLSeconds(cfg.ConnTTLSeconds) {
		return errors.New("smtp_conn_ttl_seconds invalid")
	}
	if strings.TrimSpace(cfg.SenderEmail) == "" {
		return errors.New("smtp_sender_email required")
	}
	if _, err := mail.ParseAddress(cfg.SenderEmail); err != nil {
		return errors.New("smtp_sender_email invalid")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New("smtp_username required")
	}
	if passwordRequired && strings.TrimSpace(cfg.Password) == "" {
		return errors.New("smtp_password required")
	}
	return nil
}

func sendSMTPMailOnce(cfg smtpConfig, toEmail, subject, body string) error {
	timeout := time.Duration(cfg.ConnTTLSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(defaultSMTPConnTTLSeconds) * time.Second
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dialer := &net.Dialer{Timeout: timeout}
	var client *smtp.Client
	var err error
	if cfg.ForceSSL {
		tlsConn, tlsErr := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if tlsErr != nil {
			return fmt.Errorf("dial smtp server failed")
		}
		client, err = smtp.NewClient(tlsConn, cfg.Host)
		if err != nil {
			return fmt.Errorf("create smtp ssl client failed")
		}
	} else {
		conn, dialErr := dialer.Dial("tcp", addr)
		if dialErr != nil {
			return fmt.Errorf("dial smtp server failed")
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("create smtp client failed")
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("smtp starttls failed")
			}
		}
	}
	defer func() {
		_ = client.Quit()
		_ = client.Close()
	}()

	if err := smtpAuth(client, cfg); err != nil {
		return fmt.Errorf("smtp auth failed: %v", err)
	}
	if err := client.Mail(cfg.SenderEmail); err != nil {
		return fmt.Errorf("smtp set sender failed")
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("smtp set recipient failed")
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data command failed")
	}
	from := parseSenderAddress(cfg.SenderName, cfg.SenderEmail)
	if from == "" {
		from = cfg.SenderEmail
	}
	content := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, toEmail, subject, body)
	if _, err := writer.Write([]byte(content)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write message failed")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp finalize message failed")
	}
	return nil
}

func shouldRetrySMTPError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if text == "" {
		return false
	}
	if strings.Contains(text, "system busy") {
		return true
	}
	if strings.Contains(text, "too many") {
		return true
	}
	return false
}

func sendSMTPMail(cfg smtpConfig, toEmail, subject, body string) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		lastErr = sendSMTPMailOnce(cfg, toEmail, subject, body)
		if lastErr == nil {
			return nil
		}
		if !shouldRetrySMTPError(lastErr) || i == 2 {
			return lastErr
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return lastErr
}

func (h *Handler) listSystemSettings() (map[string]models.SystemSetting, error) {
	var items []models.SystemSetting
	if err := h.DB.Order("created_at asc").Find(&items).Error; err != nil {
		return nil, err
	}
	out := make(map[string]models.SystemSetting, len(items))
	for i := range items {
		item := items[i]
		out[item.SettingKey] = item
	}
	return out, nil
}

func (h *Handler) refreshServiceStatusHeartbeat(items map[string]models.SystemSetting) map[string]models.SystemSetting {
	now := time.Now()
	current := getStringSetting(items, systemSettingServiceStatusUpdatedAtKey, "")
	if !isServiceStatusHeartbeatDue(current, now) {
		return items
	}

	refreshedAt := now.Format(time.RFC3339)
	setting := models.SystemSetting{
		SettingKey:   systemSettingServiceStatusUpdatedAtKey,
		SettingValue: refreshedAt,
		ValueType:    "string",
		Description:  systemSettingDescription(systemSettingServiceStatusUpdatedAtKey),
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"setting_value": refreshedAt,
			"value_type":    "string",
			"description":   systemSettingDescription(systemSettingServiceStatusUpdatedAtKey),
			"updated_at":    now,
		}),
	}).Create(&setting).Error; err != nil {
		return items
	}

	if items == nil {
		items = map[string]models.SystemSetting{}
	}
	items[systemSettingServiceStatusUpdatedAtKey] = setting
	return items
}

func buildSystemSettingsResponse(items map[string]models.SystemSetting) systemSettingsResponse {
	siteName := defaultSiteName
	if item, ok := items[systemSettingSiteNameKey]; ok {
		value := strings.TrimSpace(item.SettingValue)
		if value != "" {
			siteName = value
		}
	}
	smtpPort := getIntSetting(items, systemSettingSMTPPortKey, defaultSMTPPort)
	if !validateSMTPPort(smtpPort) {
		smtpPort = defaultSMTPPort
	}
	smtpConnTTLSeconds := getIntSetting(items, systemSettingSMTPConnTTLSecondsKey, defaultSMTPConnTTLSeconds)
	if !validateSMTPConnTTLSeconds(smtpConnTTLSeconds) {
		smtpConnTTLSeconds = defaultSMTPConnTTLSeconds
	}
	smtpPasswordConfigured := strings.TrimSpace(getStringSetting(items, systemSettingSMTPPasswordKey, "")) != ""
	legacyPageAnnouncementEnabled := getBoolSetting(items, systemSettingPageAnnouncementEnabledKey, defaultPageAnnouncementEnabled)
	legacyPageAnnouncementContent := getStringSetting(items, systemSettingPageAnnouncementContentKey, defaultPageAnnouncementContent)
	homePageAnnouncementEnabled := getBoolSetting(items, systemSettingHomePageAnnouncementEnabledKey, legacyPageAnnouncementEnabled)
	homePageAnnouncementContent := getStringSetting(items, systemSettingHomePageAnnouncementContentKey, legacyPageAnnouncementContent)
	if strings.TrimSpace(homePageAnnouncementContent) == "" {
		homePageAnnouncementContent = defaultHomePageAnnouncementContent
	}
	appsPageAnnouncementEnabled := getBoolSetting(items, systemSettingAppsPageAnnouncementEnabledKey, legacyPageAnnouncementEnabled)
	appsPageAnnouncementContent := getStringSetting(items, systemSettingAppsPageAnnouncementContentKey, legacyPageAnnouncementContent)
	if strings.TrimSpace(appsPageAnnouncementContent) == "" {
		appsPageAnnouncementContent = defaultAppsPageAnnouncementContent
	}
	serviceStatusMessage := getStringSetting(items, systemSettingServiceStatusMessageKey, defaultServiceStatusMessage)
	if strings.TrimSpace(serviceStatusMessage) == "" {
		serviceStatusMessage = defaultServiceStatusMessage
	}
	return systemSettingsResponse{
		SiteName:                    siteName,
		AllowUserRegister:           getBoolSetting(items, systemSettingAllowUserRegisterKey, defaultAllowUserRegister),
		AllowEnterpriseRegister:     getBoolSetting(items, systemSettingAllowEnterpriseRegisterKey, defaultAllowEnterpriseRegister),
		MailActivationEnabled:       getBoolSetting(items, systemSettingMailActivationEnabledKey, defaultMailActivationEnabled),
		OrgPlanTypes:                getOrgPlanTypesSetting(items),
		HomePageAnnouncementEnabled: homePageAnnouncementEnabled,
		HomePageAnnouncementContent: homePageAnnouncementContent,
		AppsPageAnnouncementEnabled: appsPageAnnouncementEnabled,
		AppsPageAnnouncementContent: appsPageAnnouncementContent,
		// Keep deprecated fields for old clients. It follows app-page settings now.
		PageAnnouncementEnabled:   appsPageAnnouncementEnabled,
		PageAnnouncementContent:   appsPageAnnouncementContent,
		SMTPSenderName:            getStringSetting(items, systemSettingSMTPSenderNameKey, ""),
		SMTPSenderEmail:           getStringSetting(items, systemSettingSMTPSenderEmailKey, ""),
		SMTPHost:                  getStringSetting(items, systemSettingSMTPHostKey, ""),
		SMTPPort:                  smtpPort,
		SMTPUsername:              getStringSetting(items, systemSettingSMTPUsernameKey, ""),
		SMTPConnTTLSeconds:        smtpConnTTLSeconds,
		SMTPForceSSL:              getBoolSetting(items, systemSettingSMTPForceSSLKey, defaultSMTPForceSSL),
		SMTPPasswordConfigured:    smtpPasswordConfigured,
		RegisterEmailCodeTemplate: getStringSetting(items, systemSettingRegisterEmailCodeTemplateKey, defaultRegisterEmailCodeTemplate),
		ServiceStatusOverall:      normalizeServiceStatusOverall(getStringSetting(items, systemSettingServiceStatusOverallKey, defaultServiceStatusOverall)),
		ServiceStatusMessage:      serviceStatusMessage,
		ServiceStatusAnnouncement: getStringSetting(items, systemSettingServiceStatusAnnouncementKey, defaultServiceStatusAnnouncement),
		ServiceStatusComponents:   getServiceStatusComponentsSetting(items),
		ServiceStatusIncidents:    getServiceStatusIncidentsSetting(items),
		ServiceStatusUpdatedAt:    buildServiceStatusUpdatedAt(getStringSetting(items, systemSettingServiceStatusUpdatedAtKey, "")),
	}
}

func (h *Handler) allowUserRegisterEnabled() (bool, error) {
	if !h.hasSystemSettingsTable() {
		return defaultAllowUserRegister, nil
	}
	items, err := h.listSystemSettings()
	if err != nil {
		return false, err
	}
	return getBoolSetting(items, systemSettingAllowUserRegisterKey, defaultAllowUserRegister), nil
}

func (h *Handler) allowEnterpriseRegisterEnabled() (bool, error) {
	if !h.hasSystemSettingsTable() {
		return defaultAllowEnterpriseRegister, nil
	}
	items, err := h.listSystemSettings()
	if err != nil {
		return false, err
	}
	return getBoolSetting(items, systemSettingAllowEnterpriseRegisterKey, defaultAllowEnterpriseRegister), nil
}

func (h *Handler) getOrgPlanTypes() ([]string, error) {
	if !h.hasSystemSettingsTable() {
		return normalizeOrgPlanTypes(nil), nil
	}
	items, err := h.listSystemSettings()
	if err != nil {
		return nil, err
	}
	return getOrgPlanTypesSetting(items), nil
}

func (h *Handler) GetSystemSettings(c *gin.Context) {
	if !h.hasSystemSettingsTable() {
		c.JSON(http.StatusOK, buildSystemSettingsResponse(nil))
		return
	}
	items, err := h.listSystemSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	items = h.refreshServiceStatusHeartbeat(items)
	c.JSON(http.StatusOK, buildSystemSettingsResponse(items))
}

func (h *Handler) GetPublicSettings(c *gin.Context) {
	if !h.hasSystemSettingsTable() {
		c.JSON(http.StatusOK, buildSystemSettingsResponse(nil))
		return
	}
	items, err := h.listSystemSettings()
	if err != nil {
		c.JSON(http.StatusOK, buildSystemSettingsResponse(nil))
		return
	}
	items = h.refreshServiceStatusHeartbeat(items)
	c.JSON(http.StatusOK, buildSystemSettingsResponse(items))
}

func (h *Handler) UpdateSystemSettings(c *gin.Context) {
	if !h.hasSystemSettingsTable() {
		if err := h.DB.AutoMigrate(&models.SystemSetting{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize system settings table"})
			return
		}
	}

	var req updateSystemSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existingItems, err := h.listSystemSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}

	type settingUpdate struct {
		Value     string
		ValueType string
	}
	updates := map[string]settingUpdate{}
	if req.SiteName != nil {
		siteName := strings.TrimSpace(*req.SiteName)
		if siteName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "site_name required"})
			return
		}
		if len([]rune(siteName)) > maxSiteNameLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "site_name too long"})
			return
		}
		updates[systemSettingSiteNameKey] = settingUpdate{Value: siteName, ValueType: "string"}
	}
	if req.AllowUserRegister != nil {
		updates[systemSettingAllowUserRegisterKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AllowUserRegister),
			ValueType: "bool",
		}
	}
	if req.AllowEnterpriseRegister != nil {
		updates[systemSettingAllowEnterpriseRegisterKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AllowEnterpriseRegister),
			ValueType: "bool",
		}
	}
	if req.MailActivationEnabled != nil {
		updates[systemSettingMailActivationEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.MailActivationEnabled),
			ValueType: "bool",
		}
	}
	if req.OrgPlanTypes != nil {
		normalized := normalizeOrgPlanTypes(*req.OrgPlanTypes)
		if len(normalized) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "org_plan_types invalid"})
			return
		}
		payload, err := json.Marshal(normalized)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode org_plan_types"})
			return
		}
		updates[systemSettingOrgPlanTypesKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
	}
	// Legacy announcement fields: keep compatibility for older clients.
	if req.PageAnnouncementEnabled != nil {
		value := strconv.FormatBool(*req.PageAnnouncementEnabled)
		updates[systemSettingPageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
		updates[systemSettingHomePageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
		updates[systemSettingAppsPageAnnouncementEnabledKey] = settingUpdate{
			Value:     value,
			ValueType: "bool",
		}
	}
	if req.PageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.PageAnnouncementContent)
		if len([]rune(content)) > maxPageAnnouncementContentLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page_announcement_content too long"})
			return
		}
		updates[systemSettingPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
		updates[systemSettingHomePageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
		updates[systemSettingAppsPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.HomePageAnnouncementEnabled != nil {
		updates[systemSettingHomePageAnnouncementEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.HomePageAnnouncementEnabled),
			ValueType: "bool",
		}
	}
	if req.HomePageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.HomePageAnnouncementContent)
		if len([]rune(content)) > maxPageAnnouncementContentLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "home_page_announcement_content too long"})
			return
		}
		updates[systemSettingHomePageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.AppsPageAnnouncementEnabled != nil {
		updates[systemSettingAppsPageAnnouncementEnabledKey] = settingUpdate{
			Value:     strconv.FormatBool(*req.AppsPageAnnouncementEnabled),
			ValueType: "bool",
		}
	}
	if req.AppsPageAnnouncementContent != nil {
		content := strings.TrimSpace(*req.AppsPageAnnouncementContent)
		if len([]rune(content)) > maxPageAnnouncementContentLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "apps_page_announcement_content too long"})
			return
		}
		updates[systemSettingAppsPageAnnouncementContentKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	if req.RegisterEmailCodeTemplate != nil {
		content := strings.TrimSpace(*req.RegisterEmailCodeTemplate)
		if content == "" {
			content = defaultRegisterEmailCodeTemplate
		}
		if len([]rune(content)) > maxRegisterEmailCodeTemplateLength {
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_email_code_template too long"})
			return
		}
		updates[systemSettingRegisterEmailCodeTemplateKey] = settingUpdate{
			Value:     content,
			ValueType: "string",
		}
	}
	serviceStatusUpdated := false
	if req.ServiceStatusOverall != nil {
		updates[systemSettingServiceStatusOverallKey] = settingUpdate{
			Value:     normalizeServiceStatusOverall(*req.ServiceStatusOverall),
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusMessage != nil {
		messageValue, msgErr := validateServiceStatusMessage(*req.ServiceStatusMessage)
		if msgErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": msgErr.Error()})
			return
		}
		updates[systemSettingServiceStatusMessageKey] = settingUpdate{
			Value:     messageValue,
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusAnnouncement != nil {
		announcementValue, annErr := validateServiceStatusAnnouncement(*req.ServiceStatusAnnouncement)
		if annErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": annErr.Error()})
			return
		}
		updates[systemSettingServiceStatusAnnouncementKey] = settingUpdate{
			Value:     announcementValue,
			ValueType: "string",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusComponents != nil {
		components, compErr := validateServiceStatusComponents(*req.ServiceStatusComponents)
		if compErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": compErr.Error()})
			return
		}
		payload, marshalErr := json.Marshal(components)
		if marshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode service_status_components"})
			return
		}
		updates[systemSettingServiceStatusComponentsKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
		serviceStatusUpdated = true
	}
	if req.ServiceStatusIncidents != nil {
		incidents, incidentErr := validateServiceStatusIncidents(*req.ServiceStatusIncidents)
		if incidentErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": incidentErr.Error()})
			return
		}
		payload, marshalErr := json.Marshal(incidents)
		if marshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode service_status_incidents"})
			return
		}
		updates[systemSettingServiceStatusIncidentsKey] = settingUpdate{
			Value:     string(payload),
			ValueType: "json",
		}
		serviceStatusUpdated = true
	}
	if serviceStatusUpdated {
		updates[systemSettingServiceStatusUpdatedAtKey] = settingUpdate{
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
		smtpCfg := h.getSMTPConfigFromSettings(existingItems)
		password, configured, passwordErr := h.getSMTPPasswordFromSettings(existingItems)
		if passwordErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to load smtp password"})
			return
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
		if err := validateSMTPConfig(smtpCfg, passwordRequired); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates[systemSettingSMTPSenderNameKey] = settingUpdate{Value: smtpCfg.SenderName, ValueType: "string"}
		updates[systemSettingSMTPSenderEmailKey] = settingUpdate{Value: smtpCfg.SenderEmail, ValueType: "string"}
		updates[systemSettingSMTPHostKey] = settingUpdate{Value: smtpCfg.Host, ValueType: "string"}
		updates[systemSettingSMTPPortKey] = settingUpdate{Value: strconv.Itoa(smtpCfg.Port), ValueType: "int"}
		updates[systemSettingSMTPUsernameKey] = settingUpdate{Value: smtpCfg.Username, ValueType: "string"}
		updates[systemSettingSMTPConnTTLSecondsKey] = settingUpdate{Value: strconv.Itoa(smtpCfg.ConnTTLSeconds), ValueType: "int"}
		updates[systemSettingSMTPForceSSLKey] = settingUpdate{Value: strconv.FormatBool(smtpCfg.ForceSSL), ValueType: "bool"}
		if passwordUpdated {
			updates[systemSettingSMTPPasswordKey] = settingUpdate{Value: smtpCfg.Password, ValueType: "string"}
		}
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no settings to update"})
		return
	}

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

	now := time.Now()
	for key, update := range updates {
		updatedBy := userUUID
		item := models.SystemSetting{
			SettingKey:   key,
			SettingValue: update.Value,
			ValueType:    update.ValueType,
			Description:  systemSettingDescription(key),
			UpdatedBy:    &updatedBy,
		}
		if err := h.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"setting_value": update.Value,
				"value_type":    update.ValueType,
				"description":   systemSettingDescription(key),
				"updated_by":    userUUID,
				"updated_at":    now,
			}),
		}).Create(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save system settings"})
			return
		}
	}

	items, err := h.listSystemSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	c.JSON(http.StatusOK, buildSystemSettingsResponse(items))
}

func (h *Handler) TestSystemSMTP(c *gin.Context) {
	var req testSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.hasSystemSettingsTable() {
		if err := h.DB.AutoMigrate(&models.SystemSetting{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize system settings table"})
			return
		}
	}
	items, err := h.listSystemSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	cfg := h.getSMTPConfigFromSettings(items)
	password, configured, passwordErr := h.getSMTPPasswordFromSettings(items)
	if passwordErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to load smtp password"})
		return
	}
	cfg.Password = password

	if req.SMTPSenderName != nil {
		cfg.SenderName = strings.TrimSpace(*req.SMTPSenderName)
	}
	if req.SMTPSenderEmail != nil {
		cfg.SenderEmail = strings.TrimSpace(*req.SMTPSenderEmail)
	}
	if req.SMTPHost != nil {
		cfg.Host = strings.TrimSpace(*req.SMTPHost)
	}
	if req.SMTPPort != nil {
		cfg.Port = *req.SMTPPort
	}
	if req.SMTPUsername != nil {
		cfg.Username = strings.TrimSpace(*req.SMTPUsername)
	}
	if req.SMTPConnTTLSeconds != nil {
		cfg.ConnTTLSeconds = *req.SMTPConnTTLSeconds
	}
	if req.SMTPForceSSL != nil {
		cfg.ForceSSL = *req.SMTPForceSSL
	}
	if req.SMTPPassword != nil && strings.TrimSpace(*req.SMTPPassword) != "" {
		cfg.Password = strings.TrimSpace(*req.SMTPPassword)
		configured = true
	}
	if err := validateSMTPConfig(cfg, !configured); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subject := "SMTP 测试邮件"
	if req.Subject != nil && strings.TrimSpace(*req.Subject) != "" {
		subject = strings.TrimSpace(*req.Subject)
	}
	body := "这是一封来自系统设置的测试邮件。"
	if req.Body != nil && strings.TrimSpace(*req.Body) != "" {
		body = strings.TrimSpace(*req.Body)
	}
	if err := sendSMTPMail(cfg, strings.TrimSpace(req.ToEmail), subject, body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": sanitizeSMTPError(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

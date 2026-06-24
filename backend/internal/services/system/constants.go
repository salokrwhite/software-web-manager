package system

import "time"

// Setting keys persisted in the system_settings table.
const (
	SettingSiteNameKey                    = "site_name"
	SettingAllowUserRegisterKey           = "allow_user_register"
	SettingAllowEnterpriseRegisterKey     = "allow_enterprise_register"
	SettingMailActivationEnabledKey       = "mail_activation_enabled"
	SettingOrgPlanTypesKey                = "org_plan_types"
	SettingHomePageAnnouncementEnabledKey = "home_page_announcement_enabled"
	SettingHomePageAnnouncementContentKey = "home_page_announcement_content"
	SettingAppsPageAnnouncementEnabledKey = "apps_page_announcement_enabled"
	SettingAppsPageAnnouncementContentKey = "apps_page_announcement_content"
	// Deprecated: kept for backward compatibility.
	SettingPageAnnouncementEnabledKey = "page_announcement_enabled"
	// Deprecated: kept for backward compatibility.
	SettingPageAnnouncementContentKey   = "page_announcement_content"
	SettingSMTPSenderNameKey            = "smtp_sender_name"
	SettingSMTPSenderEmailKey           = "smtp_sender_email"
	SettingSMTPHostKey                  = "smtp_host"
	SettingSMTPPortKey                  = "smtp_port"
	SettingSMTPUsernameKey              = "smtp_username"
	SettingSMTPPasswordKey              = "smtp_password"
	SettingSMTPConnTTLSecondsKey        = "smtp_conn_ttl_seconds"
	SettingSMTPForceSSLKey              = "smtp_force_ssl"
	SettingRegisterEmailCodeTemplateKey = "register_email_code_template"
	SettingServiceStatusOverallKey      = "service_status_overall_status"
	SettingServiceStatusMessageKey      = "service_status_overall_message"
	SettingServiceStatusAnnouncementKey = "service_status_announcement"
	SettingServiceStatusComponentsKey   = "service_status_components"
	SettingServiceStatusIncidentsKey    = "service_status_incidents"
	SettingServiceStatusUpdatedAtKey    = "service_status_updated_at"
	SettingSSOEnabledKey                = "sso_enabled"
	SettingSSODisplayNameKey            = "sso_display_name"
	SettingSSOIssuerKey                 = "sso_issuer"
	SettingSSOAuthorizeEndpointKey      = "sso_authorize_endpoint"
	SettingSSOTokenEndpointKey          = "sso_token_endpoint"
	SettingSSOUserinfoEndpointKey       = "sso_userinfo_endpoint"
	SettingSSOJWKSURIKey                = "sso_jwks_uri"
	SettingSSOClientIDKey               = "sso_client_id"
	SettingSSOClientSecretKey           = "sso_client_secret"
	SettingSSOScopesKey                 = "sso_scopes"
	SettingSSORedirectURIKey            = "sso_redirect_uri"
)

// Default values applied when a setting is missing or blank.
const (
	DefaultSiteName                = "SWM 软件版本管理平台"
	DefaultAllowUserRegister       = true
	DefaultAllowEnterpriseRegister = true
	DefaultMailActivationEnabled   = false
	DefaultHomePageAnnouncementEnabled = false
	DefaultHomePageAnnouncementContent = ""
	DefaultAppsPageAnnouncementEnabled = false
	DefaultAppsPageAnnouncementContent = ""
	// Deprecated: kept for backward compatibility.
	DefaultPageAnnouncementEnabled = false
	// Deprecated: kept for backward compatibility.
	DefaultPageAnnouncementContent     = ""
	DefaultSMTPPort                    = 465
	DefaultSMTPConnTTLSeconds          = 15
	DefaultSMTPForceSSL                = true
	DefaultRegisterEmailCodeTemplate   = "您好，\n\n您的注册验证码是 {{code}}，有效期 {{minutes}} 分钟。\n\n如果这不是您的操作，请忽略此邮件。\n\n{{site_name}}"
	DefaultServiceStatusOverall        = "operational"
	DefaultServiceStatusMessage        = "所有系统正常运行"
	DefaultServiceStatusAnnouncement   = ""
	DefaultSSOEnabled                  = false
	DefaultSSODisplayName              = "SSO 单点登录"
	DefaultSSOScopes                   = "openid email profile"
)

// Validation bounds.
const (
	MaxSiteNameLength                  = 100
	MaxPageAnnouncementContentLength   = 500
	MaxRegisterEmailCodeTemplateLength = 2000
	MaxServiceStatusMessageLength      = 300
	MaxServiceStatusAnnouncementLength = 1000
	MaxServiceStatusItemsCount         = 20
	MaxServiceStatusTitleLength        = 100
	MaxServiceStatusDescriptionLength  = 500
	MaxServiceStatusTimeLength         = 64
)

const (
	serviceStatusHeartbeatInterval = 5 * time.Minute
	serviceStatusTimeLayout        = "2006-01-02T15:04:05Z07:00"
)

// SettingDescription returns the human-readable label stored alongside a setting.
func SettingDescription(key string) string {
	switch key {
	case SettingSiteNameKey:
		return "站点名称"
	case SettingAllowUserRegisterKey:
		return "允许新用户注册"
	case SettingAllowEnterpriseRegisterKey:
		return "允许企业用户注册"
	case SettingMailActivationEnabledKey:
		return "邮件激活"
	case SettingOrgPlanTypesKey:
		return "企业套餐类型"
	case SettingHomePageAnnouncementEnabledKey:
		return "首页公告启用"
	case SettingHomePageAnnouncementContentKey:
		return "首页公告内容"
	case SettingAppsPageAnnouncementEnabledKey:
		return "应用管理页面公告启用"
	case SettingAppsPageAnnouncementContentKey:
		return "应用管理页面公告内容"
	case SettingPageAnnouncementEnabledKey:
		return "页面公告启用"
	case SettingPageAnnouncementContentKey:
		return "页面公告内容"
	case SettingSMTPSenderNameKey:
		return "SMTP发件人名"
	case SettingSMTPSenderEmailKey:
		return "SMTP发件人邮箱"
	case SettingSMTPHostKey:
		return "SMTP服务器"
	case SettingSMTPPortKey:
		return "SMTP端口"
	case SettingSMTPUsernameKey:
		return "SMTP用户名"
	case SettingSMTPPasswordKey:
		return "SMTP密码"
	case SettingSMTPConnTTLSecondsKey:
		return "SMTP连接有效期(秒)"
	case SettingSMTPForceSSLKey:
		return "SMTP强制SSL"
	case SettingRegisterEmailCodeTemplateKey:
		return "注册邮箱验证码模板"
	case SettingServiceStatusOverallKey:
		return "服务状态总览状态"
	case SettingServiceStatusMessageKey:
		return "服务状态总览说明"
	case SettingServiceStatusAnnouncementKey:
		return "服务状态公告"
	case SettingServiceStatusComponentsKey:
		return "服务状态组件列表"
	case SettingServiceStatusIncidentsKey:
		return "服务状态事件列表"
	case SettingServiceStatusUpdatedAtKey:
		return "服务状态更新时间"
	case SettingSSOEnabledKey:
		return "SSO单点登录启用"
	case SettingSSODisplayNameKey:
		return "SSO登录按钮文案"
	case SettingSSOIssuerKey:
		return "SSO Issuer"
	case SettingSSOAuthorizeEndpointKey:
		return "SSO授权端点"
	case SettingSSOTokenEndpointKey:
		return "SSO Token端点"
	case SettingSSOUserinfoEndpointKey:
		return "SSO UserInfo端点"
	case SettingSSOJWKSURIKey:
		return "SSO JWKS地址"
	case SettingSSOClientIDKey:
		return "SSO Client ID"
	case SettingSSOClientSecretKey:
		return "SSO Client Secret"
	case SettingSSOScopesKey:
		return "SSO Scopes"
	case SettingSSORedirectURIKey:
		return "SSO回调地址"
	default:
		return ""
	}
}

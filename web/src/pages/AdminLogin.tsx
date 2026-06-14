import { Button, Card, Divider, Form, Input, message, Typography, Checkbox, Grid } from 'antd'
import { useState } from 'react'
import { useLocation, useNavigate, Link as RouterLink } from 'react-router-dom'
import { useTranslation, initReactI18next } from 'react-i18next'
import {
  SafetyOutlined,
  LockOutlined,
  MailOutlined,
  SecurityScanOutlined,
  KeyOutlined,
  SafetyCertificateOutlined,
  GlobalOutlined,
  LoginOutlined
} from '@ant-design/icons'
import i18next from 'i18next'
import api, { getErrorMessage, storeTokens } from '../api/client'
import { getSafeRedirectPath } from '../utils/redirect'
import { useSiteName } from '../utils/siteName'
import { useSSOConfig, startSSOLogin } from '../utils/ssoConfig'

const { Title, Text, Paragraph } = Typography

type AdminLoginLanguage = 'zh' | 'en'

const ADMIN_LOGIN_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialAdminLoginLanguage = (): AdminLoginLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(ADMIN_LOGIN_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const adminLoginResources = {
  zh: {
    translation: {
      adminLoginPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        loginSuccess: '登录成功',
        pendingReview: '账号待审核，请联系系统管理员',
        accountDisabled: '账号已停用，请联系系统管理员',
        loginFailed: '登录失败',
        brandDescription: '仅供企业管理员使用，拥有企业管理权限',
        backToNormalLogin: '返回普通登录',
        title: '管理员登录',
        subtitle: '企业管理员专用入口',
        warningNotice: '此入口仅限企业管理员使用。普通员工请使用普通登录入口。',
        emailRequired: '请输入管理员邮箱',
        emailInvalid: '请输入有效的邮箱地址',
        emailPlaceholder: '管理员邮箱',
        passwordRequired: '请输入密码',
        passwordPlaceholder: '密码',
        otpPlaceholder: '双因素验证码 (2FA)',
        rememberMe: '记住我',
        forgotPassword: '忘记密码？',
        submitButton: '管理员登录',
        supportPrefix: '需要帮助？请联系 ',
        supportLink: '企业支持',
        features: [
          { title: '系统管理员权限', desc: '管理所有企业组织与账号' },
          { title: '企业账号管理', desc: '创建和管理企业管理员账号' },
          { title: '全局审计', desc: '查看全系统操作日志' }
        ]
      }
    }
  },
  en: {
    translation: {
      adminLoginPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        loginSuccess: 'Login successful',
        pendingReview: 'Account pending review. Please contact the system administrator.',
        accountDisabled: 'Account has been disabled. Please contact the system administrator.',
        loginFailed: 'Login failed',
        brandDescription: 'For enterprise administrators only, with organization-level management permissions.',
        backToNormalLogin: 'Back to User Login',
        title: 'Admin Login',
        subtitle: 'Enterprise administrator access',
        warningNotice: 'This entry is for enterprise administrators only. Regular users should use the standard login entry.',
        emailRequired: 'Please enter admin email',
        emailInvalid: 'Please enter a valid email address',
        emailPlaceholder: 'Admin email',
        passwordRequired: 'Please enter your password',
        passwordPlaceholder: 'Password',
        otpPlaceholder: 'Two-Factor Code (2FA)',
        rememberMe: 'Remember me',
        forgotPassword: 'Forgot password?',
        submitButton: 'Admin Sign In',
        supportPrefix: 'Need help? Contact ',
        supportLink: 'Enterprise Support',
        features: [
          { title: 'System Admin Privileges', desc: 'Manage all enterprise organizations and accounts' },
          { title: 'Enterprise Account Management', desc: 'Create and manage enterprise administrator accounts' },
          { title: 'Global Audit', desc: 'Review system-wide operation logs' }
        ]
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: adminLoginResources,
    lng: getInitialAdminLoginLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', adminLoginResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', adminLoginResources.en.translation, true, true)
}

export default function AdminLogin({ redirectTo = '/system/orgs' }: { redirectTo?: string }) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const siteName = useSiteName()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const params = new URLSearchParams(location.search)
  const redirectParam = params.get('redirect')
  const safeRedirect = getSafeRedirectPath(redirectParam || redirectTo, '/system/orgs')
  const sso = useSSOConfig()
  const [ssoLoading, setSsoLoading] = useState(false)
  const currentLanguage: AdminLoginLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh'
    ? t('adminLoginPage.switchToEnglish')
    : t('adminLoginPage.switchToChinese')

  const handleSwitchLanguage = () => {
    const nextLanguage: AdminLoginLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(ADMIN_LOGIN_LANGUAGE_STORAGE_KEY, nextLanguage)
    void i18n.changeLanguage(nextLanguage).finally(() => {
      window.location.reload()
    })
  }

  const onLogin = async (values: any) => {
    try {
      const res = await api.post('/api/auth/admin-login', values)
      storeTokens(res.data.tokens)
      if (res.data.org_id) {
        sessionStorage.setItem('org_id', res.data.org_id)
      } else {
        sessionStorage.removeItem('org_id')
      }
      if (res.data.role) {
        sessionStorage.setItem('role', res.data.role)
      } else {
        sessionStorage.removeItem('role')
      }
      if (res.data.user?.email) {
        sessionStorage.setItem('user_email', res.data.user.email)
      } else {
        sessionStorage.removeItem('user_email')
      }
      if (res.data.org_type) {
        sessionStorage.setItem('org_type', res.data.org_type)
      } else {
        sessionStorage.removeItem('org_type')
      }
      sessionStorage.removeItem('impersonating')
      sessionStorage.removeItem('impersonation_org_id')
      sessionStorage.removeItem('system_backup_access_token')
      sessionStorage.removeItem('system_backup_refresh_token')
      sessionStorage.removeItem('system_backup_org_id')
      sessionStorage.removeItem('system_backup_role')
      const systemRole = (res.data.system_role || '').toLowerCase()
      if (systemRole) {
        sessionStorage.setItem('system_role', systemRole)
      } else {
        sessionStorage.removeItem('system_role')
      }
      message.success(t('adminLoginPage.loginSuccess'))
      if (systemRole === 'system_admin') {
        navigate(safeRedirect)
        return
      }
      if (safeRedirect.startsWith('/system')) {
        navigate('/dashboard')
        return
      }
      navigate(safeRedirect)
    } catch (err: any) {
      const code = err?.response?.data?.code
      if (code === 'user_pending' || code === 'org_pending') {
        message.info(t('adminLoginPage.pendingReview'))
        navigate('/pending')
        return
      }
      if (code === 'user_disabled' || code === 'org_disabled') {
        message.error(t('adminLoginPage.accountDisabled'))
        return
      }
      message.error(getErrorMessage(err, t('adminLoginPage.loginFailed')))
    }
  }

  const onSSOLogin = async () => {
    setSsoLoading(true)
    try {
      await startSSOLogin(safeRedirect)
    } catch {
      message.error('无法发起 SSO 登录，请稍后再试')
      setSsoLoading(false)
    }
  }

  const featureItems = t('adminLoginPage.features', { returnObjects: true }) as unknown as Array<{
    title: string
    desc: string
  }>
  const featureIcons = [<SecurityScanOutlined />, <KeyOutlined />, <SafetyCertificateOutlined />]
  const features = featureItems.map((item, index) => ({
    icon: featureIcons[index] ?? <SafetyCertificateOutlined />,
    title: item.title,
    desc: item.desc
  }))

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: isMobile ? 'column' : 'row',
        minHeight: '100vh',
        background: isMobile ? '#fff' : '#f0f2f5'
      }}
    >
      {!isMobile && (
        <div
          style={{
            flex: 1,
            background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
            padding: '60px',
            color: '#fff',
            position: 'relative',
            overflow: 'hidden'
          }}
        >
          <div
            style={{
              position: 'absolute',
              top: -100,
              right: -100,
              width: 400,
              height: 400,
              background: 'rgba(255,255,255,0.05)',
              borderRadius: '50%'
            }}
          />
          <div
            style={{
              position: 'absolute',
              bottom: -150,
              left: -150,
              width: 500,
              height: 500,
              background: 'rgba(255,255,255,0.03)',
              borderRadius: '50%'
            }}
          />
          <div
            style={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              width: 600,
              height: 600,
              background: 'radial-gradient(circle, rgba(24,144,255,0.1) 0%, transparent 70%)'
            }}
          />

          <div style={{ textAlign: 'center', zIndex: 1, maxWidth: 480, width: '100%' }}>
            <div
              style={{
                width: 80,
                height: 80,
                background: 'rgba(255,255,255,0.1)',
                borderRadius: 20,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 32px',
                backdropFilter: 'blur(10px)',
                border: '1px solid rgba(255,255,255,0.2)'
              }}
            >
              <SafetyOutlined style={{ fontSize: 40, color: '#fff' }} />
            </div>

            <Title level={2} style={{ color: '#fff', marginBottom: 16, fontSize: 36 }}>
              {siteName}
            </Title>
            <Paragraph style={{ color: 'rgba(255,255,255,0.7)', fontSize: 16, marginBottom: 48 }}>
              {t('adminLoginPage.brandDescription')}
            </Paragraph>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 20, textAlign: 'left' }}>
              {features.map((item, index) => (
                <div key={index} style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                  <div
                    style={{
                      width: 48,
                      height: 48,
                      background: 'rgba(255,255,255,0.1)',
                      borderRadius: 12,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 20,
                      flexShrink: 0,
                      border: '1px solid rgba(255,255,255,0.15)'
                    }}
                  >
                    {item.icon}
                  </div>
                  <div>
                    <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 4, color: '#fff' }}>{item.title}</div>
                    <div style={{ fontSize: 13, color: 'rgba(255,255,255,0.6)' }}>{item.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div style={{ position: 'absolute', bottom: 40, color: 'rgba(255,255,255,0.5)', fontSize: 13 }}>
            © 2024 {siteName}. All rights reserved.
          </div>
        </div>
      )}

      <div
        style={{
          width: isMobile ? '100%' : 480,
          background: '#fff',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: isMobile ? 'flex-start' : 'center',
          padding: isMobile ? '28px 16px 36px' : '60px',
          boxShadow: isMobile ? 'none' : '-4px 0 20px rgba(0,0,0,0.05)'
        }}
      >
        <div style={{ maxWidth: 360, margin: '0 auto', width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: isMobile ? 12 : 16 }}>
            <Button icon={<GlobalOutlined />} onClick={handleSwitchLanguage}>
              {languageSwitchLabel}
            </Button>
          </div>

          <div style={{ marginBottom: isMobile ? 24 : 40 }}>
            <RouterLink to="/login" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('adminLoginPage.backToNormalLogin')}
            </RouterLink>
          </div>

          <div style={{ textAlign: 'center', marginBottom: isMobile ? 28 : 40 }}>
            <Title level={3} style={{ marginBottom: 8 }}>{t('adminLoginPage.title')}</Title>
            <Text type="secondary">{t('adminLoginPage.subtitle')}</Text>
          </div>

          <Card style={{ marginBottom: 24, background: '#fff2f0', border: '1px solid #ffccc7' }}>
            <Text type="warning" style={{ fontSize: isMobile ? 12 : 13 }}>
              <SafetyOutlined style={{ marginRight: 8 }} />
              {t('adminLoginPage.warningNotice')}
            </Text>
          </Card>

          <Form layout="vertical" onFinish={onLogin} size={isMobile ? 'middle' : 'large'}>
            <Form.Item
              name="email"
              rules={[
                { required: true, message: t('adminLoginPage.emailRequired') },
                { type: 'email', message: t('adminLoginPage.emailInvalid') }
              ]}
            >
              <Input
                prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                placeholder={t('adminLoginPage.emailPlaceholder')}
              />
            </Form.Item>
            <Form.Item
              name="password"
              rules={[{ required: true, message: t('adminLoginPage.passwordRequired') }]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                placeholder={t('adminLoginPage.passwordPlaceholder')}
              />
            </Form.Item>
            <Form.Item name="otp_code">
              <Input
                prefix={<SafetyOutlined style={{ color: '#bfbfbf' }} />}
                placeholder={t('adminLoginPage.otpPlaceholder')}
                maxLength={6}
              />
            </Form.Item>
            <Form.Item>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Checkbox>{t('adminLoginPage.rememberMe')}</Checkbox>
                <a style={{ color: '#1890ff', fontSize: 13 }}>{t('adminLoginPage.forgotPassword')}</a>
              </div>
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size={isMobile ? 'middle' : 'large'}
              style={{ height: isMobile ? 42 : 44, background: '#1a1a2e' }}
            >
              {t('adminLoginPage.submitButton')}
            </Button>
          </Form>

          {sso.enabled && (
            <>
              <Divider plain style={{ color: '#bfbfbf', fontSize: 12 }}>或</Divider>
              <Button
                block
                size={isMobile ? 'middle' : 'large'}
                icon={<LoginOutlined />}
                loading={ssoLoading}
                onClick={onSSOLogin}
                style={{ height: isMobile ? 42 : 44 }}
              >
                {sso.displayName}
              </Button>
            </>
          )}

          <div style={{ marginTop: isMobile ? 28 : 40, textAlign: 'center' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('adminLoginPage.supportPrefix')}
              <a>{t('adminLoginPage.supportLink')}</a>
            </Text>
          </div>
        </div>
      </div>
    </div>
  )
}

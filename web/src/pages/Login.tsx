import { Button, Divider, Form, Input, message, Typography, Checkbox, Grid } from 'antd'
import { useState } from 'react'
import { useLocation, useNavigate, Link as RouterLink } from 'react-router-dom'
import { useTranslation, initReactI18next } from 'react-i18next'
import {
  RocketOutlined,
  SafetyOutlined,
  GlobalOutlined,
  TeamOutlined,
  CheckCircleOutlined,
  LockOutlined,
  MailOutlined,
  LoginOutlined
} from '@ant-design/icons'
import i18next from 'i18next'
import api, { getErrorMessage, storeTokens } from '../api/client'
import { getSafeRedirectPath } from '../utils/redirect'
import { useSiteName } from '../utils/siteName'
import { useSSOConfig, startSSOLogin } from '../utils/ssoConfig'

const { Title, Text, Paragraph } = Typography

type LoginLanguage = 'zh' | 'en'

const LOGIN_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialLoginLanguage = (): LoginLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(LOGIN_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const loginResources = {
  zh: {
    translation: {
      loginPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        pendingReview: '账号待审核，请联系系统管理员',
        accountDisabled: '账号已停用，请联系系统管理员',
        adminLoginRequired: '请使用管理员入口登录',
        loginFailed: '登录失败',
        title: '欢迎访问',
        subtitle: '登录以继续',
        brandDescription: '企业级应用版本管理解决方案，助力您的软件交付流程',
        emailRequired: '请输入邮箱',
        emailInvalid: '请输入有效的邮箱地址',
        emailPlaceholder: '企业邮箱',
        passwordRequired: '请输入密码',
        passwordPlaceholder: '密码',
        rememberMe: '记住我',
        forgotPassword: '忘记密码？',
        submit: '登 录',
        registerLink: '普通用户注册',
        adminLoginLink: '系统管理员登录',
        enterpriseRegisterLink: '企业注册',
        supportPrefix: '需要帮助？请联系 ',
        supportLink: '企业支持',
        features: [
          { title: '安全可靠', desc: '企业级数据加密保护' },
          { title: '全球分发', desc: 'CDN 加速全球访问' },
          { title: '团队协作', desc: '多角色权限管理' },
          { title: '版本管理', desc: '完整的版本生命周期' }
        ]
      }
    }
  },
  en: {
    translation: {
      loginPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        pendingReview: 'Account pending review. Please contact the system administrator.',
        accountDisabled: 'Account has been disabled. Please contact the system administrator.',
        adminLoginRequired: 'Please use the admin login entry.',
        loginFailed: 'Login failed',
        title: 'Welcome Back',
        subtitle: 'Sign in to continue',
        brandDescription: 'Enterprise-grade application version management to streamline your software delivery process.',
        emailRequired: 'Please enter your email',
        emailInvalid: 'Please enter a valid email address',
        emailPlaceholder: 'Work email',
        passwordRequired: 'Please enter your password',
        passwordPlaceholder: 'Password',
        rememberMe: 'Remember me',
        forgotPassword: 'Forgot password?',
        submit: 'Sign In',
        registerLink: 'User Registration',
        adminLoginLink: 'System Admin Login',
        enterpriseRegisterLink: 'Enterprise Registration',
        supportPrefix: 'Need help? Contact ',
        supportLink: 'Enterprise Support',
        features: [
          { title: 'Secure & Reliable', desc: 'Enterprise-grade data encryption protection' },
          { title: 'Global Delivery', desc: 'CDN-accelerated global access' },
          { title: 'Team Collaboration', desc: 'Multi-role permission management' },
          { title: 'Version Management', desc: 'Complete version lifecycle management' }
        ]
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: loginResources,
    lng: getInitialLoginLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', loginResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', loginResources.en.translation, true, true)
}

export default function Login({ redirectTo = '/dashboard' }: { redirectTo?: string }) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const siteName = useSiteName()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const params = new URLSearchParams(location.search)
  const redirectParam = params.get('redirect')
  const safeRedirect = getSafeRedirectPath(redirectParam || redirectTo, '/dashboard')
  const sso = useSSOConfig()
  const [ssoLoading, setSsoLoading] = useState(false)
  const currentLanguage: LoginLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh'
    ? t('loginPage.switchToEnglish')
    : t('loginPage.switchToChinese')

  const handleSwitchLanguage = () => {
    const nextLanguage: LoginLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(LOGIN_LANGUAGE_STORAGE_KEY, nextLanguage)
    void i18n.changeLanguage(nextLanguage).finally(() => {
      window.location.reload()
    })
  }

  const onLogin = async (values: any) => {
    try {
      const res = await api.post('/api/auth/login', values)
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
      if (res.data.system_role) {
        sessionStorage.setItem('system_role', res.data.system_role)
      } else {
        sessionStorage.removeItem('system_role')
      }
      try {
        const orgRes = await api.get('/api/orgs')
        const items = orgRes?.data?.items || []
        if (items.length > 1) {
          navigate(`/org-select?redirect=${encodeURIComponent(safeRedirect)}`)
          return
        }
      } catch {
        // ignore org list errors and fallback to default redirect
      }
      navigate(safeRedirect)
    } catch (err: any) {
      const code = err?.response?.data?.code
      if (code === 'user_pending' || code === 'org_pending') {
        message.info(t('loginPage.pendingReview'))
        navigate('/pending')
        return
      }
      if (code === 'user_disabled' || code === 'org_disabled') {
        message.error(t('loginPage.accountDisabled'))
        return
      }
      if (code === 'admin_login_required') {
        message.info(t('loginPage.adminLoginRequired'))
        navigate(`/admin-login?redirect=${encodeURIComponent(safeRedirect)}`)
        return
      }
      message.error(getErrorMessage(err, t('loginPage.loginFailed')))
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

  const featureItems = t('loginPage.features', { returnObjects: true }) as unknown as Array<{
    title: string
    desc: string
  }>
  const featureIcons = [<SafetyOutlined />, <GlobalOutlined />, <TeamOutlined />, <CheckCircleOutlined />]
  const features = featureItems.map((item, index) => ({
    icon: featureIcons[index] ?? <CheckCircleOutlined />,
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
            background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
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
              background: 'rgba(255,255,255,0.1)',
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
              background: 'rgba(255,255,255,0.08)',
              borderRadius: '50%'
            }}
          />

          <div style={{ textAlign: 'center', zIndex: 1, maxWidth: 480, width: '100%' }}>
            <div
              style={{
                width: 80,
                height: 80,
                background: 'rgba(255,255,255,0.2)',
                borderRadius: 20,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 32px',
                backdropFilter: 'blur(10px)'
              }}
            >
              <RocketOutlined style={{ fontSize: 40, color: '#fff' }} />
            </div>

            <Title level={2} style={{ color: '#fff', marginBottom: 16, fontSize: 36 }}>
              {siteName}
            </Title>
            <Paragraph style={{ color: 'rgba(255,255,255,0.9)', fontSize: 16, marginBottom: 48 }}>
              {t('loginPage.brandDescription')}
            </Paragraph>

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: 24,
                textAlign: 'left'
              }}
            >
              {features.map((item, index) => (
                <div key={index} style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
                  <div
                    style={{
                      width: 40,
                      height: 40,
                      background: 'rgba(255,255,255,0.15)',
                      borderRadius: 10,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 18,
                      flexShrink: 0
                    }}
                  >
                    {item.icon}
                  </div>
                  <div>
                    <div style={{ fontSize: 15, fontWeight: 600, marginBottom: 4 }}>{item.title}</div>
                    <div style={{ fontSize: 13, color: 'rgba(255,255,255,0.7)' }}>{item.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div style={{ position: 'absolute', bottom: 40, color: 'rgba(255,255,255,0.6)', fontSize: 13 }}>
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

          <div style={{ textAlign: 'center', marginBottom: isMobile ? 28 : 40 }}>
            <Title level={3} style={{ marginBottom: 8 }}>{t('loginPage.title')}</Title>
            <Text type="secondary">{t('loginPage.subtitle')}</Text>
          </div>

          <Form layout="vertical" onFinish={onLogin} size={isMobile ? 'middle' : 'large'}>
            <Form.Item
              name="email"
              rules={[
                { required: true, message: t('loginPage.emailRequired') },
                { type: 'email', message: t('loginPage.emailInvalid') }
              ]}
            >
              <Input
                prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                placeholder={t('loginPage.emailPlaceholder')}
              />
            </Form.Item>
            <Form.Item
              name="password"
              rules={[{ required: true, message: t('loginPage.passwordRequired') }]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                placeholder={t('loginPage.passwordPlaceholder')}
              />
            </Form.Item>
            <Form.Item>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Checkbox>{t('loginPage.rememberMe')}</Checkbox>
                <a style={{ color: '#1890ff', fontSize: 13 }}>{t('loginPage.forgotPassword')}</a>
              </div>
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size={isMobile ? 'middle' : 'large'}
              style={{ height: isMobile ? 42 : 44 }}
            >
              {t('loginPage.submit')}
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

          <div style={{ marginTop: isMobile ? 20 : 24, textAlign: 'center' }}>
            <RouterLink to="/register" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('loginPage.registerLink')}
            </RouterLink>
          </div>
          <div style={{ marginTop: 16, textAlign: 'center' }}>
            <RouterLink to="/admin-login" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('loginPage.adminLoginLink')}
            </RouterLink>
          </div>
          <div style={{ marginTop: 12, textAlign: 'center' }}>
            <RouterLink to="/enterprise-register" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('loginPage.enterpriseRegisterLink')}
            </RouterLink>
          </div>

          <div style={{ marginTop: 24, textAlign: 'center' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('loginPage.supportPrefix')}
              <a>{t('loginPage.supportLink')}</a>
            </Text>
          </div>
        </div>
      </div>
    </div>
  )
}

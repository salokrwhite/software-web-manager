import { Alert, Button, Form, Input, message, Spin, Typography, Checkbox, Grid, Space } from 'antd'
import { useNavigate, Link as RouterLink } from 'react-router-dom'
import { useEffect, useState } from 'react'
import {
  RocketOutlined,
  SafetyOutlined,
  GlobalOutlined,
  TeamOutlined,
  CheckCircleOutlined,
  LockOutlined,
  MailOutlined
} from '@ant-design/icons'
import i18next from 'i18next'
import { initReactI18next, useTranslation } from 'react-i18next'
import api, { getErrorMessage } from '../api/client'
import { useSiteName } from '../utils/siteName'

const { Title, Text, Paragraph } = Typography
type RegisterLanguage = 'zh' | 'en'

const REGISTER_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialRegisterLanguage = (): RegisterLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(REGISTER_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const registerResources = {
  zh: {
    translation: {
      registerPage: {
        title: '注册账号',
        subtitle: '创建普通用户账号',
        brandDescription: '企业级应用版本管理解决方案，助力您的软件交付流程',
        switchToEnglish: 'English',
        switchToChinese: '中文',
        registerClosedWarning: '当前系统已关闭新用户注册',
        registerSuccess: '注册成功，请联系管理员加入组织',
        registerFailed: '注册失败',
        emailFirstWarning: '请先输入邮箱',
        emailInvalidWarning: '请输入有效的邮箱地址',
        codeSentSuccess: '验证码已发送，请查收邮箱',
        sendCodeFailed: '发送验证码失败',
        emailPlaceholder: '邮箱',
        emailRequired: '请输入邮箱',
        emailFormatInvalid: '请输入有效的邮箱地址',
        emailCodePlaceholder: '邮箱验证码',
        emailCodeRequired: '请输入邮箱验证码',
        emailCodeFormatInvalid: '请输入6位数字验证码',
        sendCode: '发送验证码',
        sendCodeCooldown: '{{seconds}}s后重发',
        passwordPlaceholder: '设置密码（至少8位）',
        passwordRequired: '请输入密码',
        passwordLengthInvalid: '密码至少8位',
        agreementPrefix: '我已阅读并同意 ',
        agreementAnd: ' 和 ',
        terms: '服务条款',
        privacy: '隐私政策',
        agreementRequired: '请阅读并同意服务条款',
        submitButton: '注册账号',
        disabledMessage: '当前系统已关闭新用户注册',
        loginLink: '已有账号？去登录',
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
      registerPage: {
        title: 'Create Account',
        subtitle: 'Create a standard user account',
        brandDescription: 'Enterprise-grade application version management to streamline your software delivery process.',
        switchToEnglish: 'English',
        switchToChinese: '中文',
        registerClosedWarning: 'User registration is currently disabled',
        registerSuccess: 'Registration successful. Please contact your administrator to join an organization.',
        registerFailed: 'Registration failed',
        emailFirstWarning: 'Please enter your email first',
        emailInvalidWarning: 'Please enter a valid email address',
        codeSentSuccess: 'Verification code sent. Please check your inbox.',
        sendCodeFailed: 'Failed to send verification code',
        emailPlaceholder: 'Email',
        emailRequired: 'Please enter your email',
        emailFormatInvalid: 'Please enter a valid email address',
        emailCodePlaceholder: 'Email verification code',
        emailCodeRequired: 'Please enter the email verification code',
        emailCodeFormatInvalid: 'Please enter a 6-digit verification code',
        sendCode: 'Send Code',
        sendCodeCooldown: 'Resend in {{seconds}}s',
        passwordPlaceholder: 'Set password (at least 8 characters)',
        passwordRequired: 'Please enter your password',
        passwordLengthInvalid: 'Password must be at least 8 characters',
        agreementPrefix: 'I have read and agree to ',
        agreementAnd: ' and ',
        terms: 'Terms of Service',
        privacy: 'Privacy Policy',
        agreementRequired: 'Please read and accept the terms',
        submitButton: 'Create Account',
        disabledMessage: 'User registration is currently disabled',
        loginLink: 'Already have an account? Sign in',
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
    resources: registerResources,
    lng: getInitialRegisterLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', registerResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', registerResources.en.translation, true, true)
}

export default function Register() {
  const navigate = useNavigate()
  const { t, i18n } = useTranslation()
  const siteName = useSiteName()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [form] = Form.useForm()
  const [registerEnabled, setRegisterEnabled] = useState(true)
  const [loading, setLoading] = useState(true)
  const [sendingCode, setSendingCode] = useState(false)
  const [codeCooldown, setCodeCooldown] = useState(0)
  const currentLanguage: RegisterLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const languageSwitchLabel = currentLanguage === 'zh'
    ? t('registerPage.switchToEnglish')
    : t('registerPage.switchToChinese')

  const handleSwitchLanguage = () => {
    const nextLanguage: RegisterLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(REGISTER_LANGUAGE_STORAGE_KEY, nextLanguage)
    void i18n.changeLanguage(nextLanguage).finally(() => {
      window.location.reload()
    })
  }

  const loadPublicSettings = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/public/settings')
      setRegisterEnabled(res?.data?.allow_user_register !== false)
    } catch {
      setRegisterEnabled(true)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPublicSettings()
  }, [])

  useEffect(() => {
    if (codeCooldown <= 0) return
    const timer = window.setInterval(() => {
      setCodeCooldown((prev) => (prev > 1 ? prev - 1 : 0))
    }, 1000)
    return () => window.clearInterval(timer)
  }, [codeCooldown])

  const onRegister = async (values: any) => {
    if (!registerEnabled) {
      message.warning(t('registerPage.registerClosedWarning'))
      return
    }
    try {
      await api.post('/api/auth/register', {
        email: values.email,
        password: values.password,
        email_code: values.email_code
      })
      message.success(t('registerPage.registerSuccess'))
      navigate('/login')
    } catch (err: any) {
      message.error(getErrorMessage(err, t('registerPage.registerFailed')))
    }
  }

  const sendEmailCode = async () => {
    if (!registerEnabled) {
      message.warning(t('registerPage.registerClosedWarning'))
      return
    }
    if (codeCooldown > 0) return
    const email = String(form.getFieldValue('email') || '').trim()
    if (!email) {
      message.warning(t('registerPage.emailFirstWarning'))
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      message.warning(t('registerPage.emailInvalidWarning'))
      return
    }
    setSendingCode(true)
    try {
      await api.post('/api/auth/register/send-code', { email })
      message.success(t('registerPage.codeSentSuccess'))
      setCodeCooldown(60)
    } catch (err: any) {
      const retryAfter = Number(err?.response?.data?.retry_after_seconds || 0)
      if (Number.isFinite(retryAfter) && retryAfter > 0) {
        setCodeCooldown(Math.max(1, Math.floor(retryAfter)))
      }
      message.error(getErrorMessage(err, t('registerPage.sendCodeFailed')))
    } finally {
      setSendingCode(false)
    }
  }

  const featureItems = t('registerPage.features', { returnObjects: true }) as unknown as Array<{
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
      {/* 左侧品牌区域（仅桌面端显示） */}
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
          {/* 背景装饰 */}
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
              {t('registerPage.brandDescription')}
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

      {/* 右侧表单区域 */}
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
            <Title level={3} style={{ marginBottom: 8 }}>{t('registerPage.title')}</Title>
            <Text type="secondary">{t('registerPage.subtitle')}</Text>
          </div>

          {loading ? (
            <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
              <Spin />
            </div>
          ) : registerEnabled ? (
            <Form form={form} layout="vertical" onFinish={onRegister} size={isMobile ? 'middle' : 'large'}>
              <Form.Item
                name="email"
                rules={[
                  { required: true, message: t('registerPage.emailRequired') },
                  { type: 'email', message: t('registerPage.emailFormatInvalid') }
                ]}
              >
                <Input
                  prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder={t('registerPage.emailPlaceholder')}
                />
              </Form.Item>
              <Form.Item
                name="email_code"
                rules={[
                  { required: true, message: t('registerPage.emailCodeRequired') },
                  { pattern: /^\d{6}$/, message: t('registerPage.emailCodeFormatInvalid') }
                ]}
              >
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                    placeholder={t('registerPage.emailCodePlaceholder')}
                    maxLength={6}
                  />
                  <Button
                    onClick={sendEmailCode}
                    loading={sendingCode}
                    disabled={codeCooldown > 0}
                    style={{ minWidth: isMobile ? 116 : 128 }}
                  >
                    {codeCooldown > 0
                      ? t('registerPage.sendCodeCooldown', { seconds: codeCooldown })
                      : t('registerPage.sendCode')}
                  </Button>
                </Space.Compact>
              </Form.Item>
              <Form.Item
                name="password"
                rules={[
                  { required: true, message: t('registerPage.passwordRequired') },
                  { min: 8, message: t('registerPage.passwordLengthInvalid') }
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder={t('registerPage.passwordPlaceholder')}
                />
              </Form.Item>
              <Form.Item
                name="agreement"
                valuePropName="checked"
                rules={[
                  {
                    validator: (_, value) => value
                      ? Promise.resolve()
                      : Promise.reject(new Error(t('registerPage.agreementRequired')))
                  }
                ]}
              >
                <Checkbox>
                  {t('registerPage.agreementPrefix')}
                  <RouterLink to="/terms">{t('registerPage.terms')}</RouterLink>
                  {t('registerPage.agreementAnd')}
                  <RouterLink to="/privacy">{t('registerPage.privacy')}</RouterLink>
                </Checkbox>
              </Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                block
                size={isMobile ? 'middle' : 'large'}
                style={{ height: isMobile ? 42 : 44 }}
              >
                {t('registerPage.submitButton')}
              </Button>
            </Form>
          ) : (
            <Alert
              type="warning"
              showIcon
              message={t('registerPage.disabledMessage')}
            />
          )}

          <div style={{ marginTop: isMobile ? 20 : 24, textAlign: 'center' }}>
            <RouterLink to="/login" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('registerPage.loginLink')}
            </RouterLink>
          </div>
          <div style={{ marginTop: 12, textAlign: 'center' }}>
            <RouterLink to="/admin-login" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('registerPage.adminLoginLink')}
            </RouterLink>
          </div>
          <div style={{ marginTop: 12, textAlign: 'center' }}>
            <RouterLink to="/enterprise-register" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('registerPage.enterpriseRegisterLink')}
            </RouterLink>
          </div>

          <div style={{ marginTop: 24, textAlign: 'center' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('registerPage.supportPrefix')}
              <a>{t('registerPage.supportLink')}</a>
            </Text>
          </div>
        </div>
      </div>
    </div>
  )
}

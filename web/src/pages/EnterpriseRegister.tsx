import { Alert, Button, Card, Checkbox, Form, Grid, Input, Spin, Steps, Upload, message, Typography, Space } from 'antd'
import { useNavigate, Link as RouterLink } from 'react-router-dom'
import { useTranslation, initReactI18next } from 'react-i18next'
import {
  RocketOutlined,
  BankOutlined,
  MailOutlined,
  LockOutlined,
  UploadOutlined,
  GlobalOutlined
} from '@ant-design/icons'
import i18next from 'i18next'
import { useEffect, useRef, useState } from 'react'
import api from '../api/client'

const { Title, Text, Paragraph } = Typography

const maxMaterialSize = 20 * 1024 * 1024
const draftStorageKey = 'enterprise_register_draft'

type EnterpriseRegisterLanguage = 'zh' | 'en'

const ENTERPRISE_REGISTER_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const getInitialEnterpriseRegisterLanguage = (): EnterpriseRegisterLanguage => {
  if (typeof window === 'undefined') {
    return 'zh'
  }

  const savedLanguage = window.localStorage.getItem(ENTERPRISE_REGISTER_LANGUAGE_STORAGE_KEY)
  if (savedLanguage === 'zh' || savedLanguage === 'en') {
    return savedLanguage
  }

  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const enterpriseRegisterResources = {
  zh: {
    translation: {
      enterpriseRegisterPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        registerDisabledWarning: '当前系统已关闭企业用户注册',
        orgNameEmpty: '请填写企业名称',
        adminEmailEmpty: '请填写管理员邮箱',
        passwordInvalid: '请输入有效的管理员密码',
        confirmPasswordMismatch: '两次输入的密码不一致',
        materialsRequired: '请上传企业材料',
        materialsSizeLimit: '材料文件大小不能超过 20MB',
        submitSuccess: '企业注册已提交审核',
        submitFailed: '提交失败',
        backToLogin: '返回登录',
        brandTitle: '企业注册申请',
        brandDescription: '提交企业信息与材料，审核通过后即可使用',
        footerCopyright: '© 2024 SWM Software Web Manager. All rights reserved.',
        stepOrgInfo: '企业信息',
        stepAdminAccount: '管理员账号',
        stepMaterialsAgreement: '材料与协议',
        orgNameLabel: '企业名称',
        orgNameRequired: '请输入企业名称',
        orgNamePlaceholder: '企业/组织名称',
        adminEmailLabel: '企业管理员邮箱',
        adminEmailRequired: '请输入管理员邮箱',
        emailInvalid: '请输入有效的邮箱地址',
        adminEmailPlaceholder: '管理员邮箱',
        passwordLabel: '管理员密码',
        passwordRequired: '请输入密码',
        passwordMin: '密码至少6位',
        passwordPlaceholder: '设置密码（至少6位）',
        confirmPasswordLabel: '确认密码',
        confirmPasswordRequired: '请确认密码',
        confirmPasswordPlaceholder: '再次输入密码',
        materialsLabel: '企业材料',
        selectMaterials: '选择材料',
        materialsHint: '必传材料，单个文件不超过 20MB',
        agreementPrefix: '我已阅读并同意 ',
        agreementAnd: ' 和 ',
        terms: '服务条款',
        privacy: '隐私政策',
        agreementRequired: '请阅读并同意服务条款与隐私政策',
        prevStep: '上一步',
        nextStep: '下一步',
        submitApplication: '提交注册申请'
      }
    }
  },
  en: {
    translation: {
      enterpriseRegisterPage: {
        switchToEnglish: 'English',
        switchToChinese: '中文',
        registerDisabledWarning: 'Enterprise registration is currently disabled',
        orgNameEmpty: 'Please enter the enterprise name',
        adminEmailEmpty: 'Please enter the admin email',
        passwordInvalid: 'Please enter a valid admin password',
        confirmPasswordMismatch: 'The two passwords do not match',
        materialsRequired: 'Please upload enterprise documents',
        materialsSizeLimit: 'Each document must be smaller than 20MB',
        submitSuccess: 'Enterprise registration submitted for review',
        submitFailed: 'Submission failed',
        backToLogin: 'Back to Login',
        brandTitle: 'Enterprise Registration',
        brandDescription: 'Submit enterprise information and documents. You can start using the platform after approval.',
        footerCopyright: '© 2024 SWM Software Web Manager. All rights reserved.',
        stepOrgInfo: 'Enterprise Info',
        stepAdminAccount: 'Admin Account',
        stepMaterialsAgreement: 'Documents & Agreement',
        orgNameLabel: 'Enterprise Name',
        orgNameRequired: 'Please enter enterprise name',
        orgNamePlaceholder: 'Enterprise/Organization name',
        adminEmailLabel: 'Enterprise Admin Email',
        adminEmailRequired: 'Please enter admin email',
        emailInvalid: 'Please enter a valid email address',
        adminEmailPlaceholder: 'Admin email',
        passwordLabel: 'Admin Password',
        passwordRequired: 'Please enter password',
        passwordMin: 'Password must be at least 6 characters',
        passwordPlaceholder: 'Set password (at least 6 characters)',
        confirmPasswordLabel: 'Confirm Password',
        confirmPasswordRequired: 'Please confirm your password',
        confirmPasswordPlaceholder: 'Enter password again',
        materialsLabel: 'Enterprise Documents',
        selectMaterials: 'Select Documents',
        materialsHint: 'Required documents, each file must be under 20MB',
        agreementPrefix: 'I have read and agree to ',
        agreementAnd: ' and ',
        terms: 'Terms of Service',
        privacy: 'Privacy Policy',
        agreementRequired: 'Please read and accept the Terms and Privacy Policy',
        prevStep: 'Previous',
        nextStep: 'Next',
        submitApplication: 'Submit Registration'
      }
    }
  }
} as const

if (!i18next.isInitialized) {
  void i18next.use(initReactI18next).init({
    resources: enterpriseRegisterResources,
    lng: getInitialEnterpriseRegisterLanguage(),
    fallbackLng: 'zh',
    interpolation: { escapeValue: false }
  })
} else {
  i18next.addResourceBundle('zh', 'translation', enterpriseRegisterResources.zh.translation, true, true)
  i18next.addResourceBundle('en', 'translation', enterpriseRegisterResources.en.translation, true, true)
}

export default function EnterpriseRegister() {
  const { t, i18n } = useTranslation()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [registerEnabled, setRegisterEnabled] = useState(true)
  const [settingsLoading, setSettingsLoading] = useState(true)
  const currentLanguage: EnterpriseRegisterLanguage = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
  const isEnglish = currentLanguage === 'en'
  const languageSwitchLabel = currentLanguage === 'zh'
    ? t('enterpriseRegisterPage.switchToEnglish')
    : t('enterpriseRegisterPage.switchToChinese')

  const handleSwitchLanguage = () => {
    const nextLanguage: EnterpriseRegisterLanguage = currentLanguage === 'zh' ? 'en' : 'zh'
    window.localStorage.setItem(ENTERPRISE_REGISTER_LANGUAGE_STORAGE_KEY, nextLanguage)
    void i18n.changeLanguage(nextLanguage).finally(() => {
      window.location.reload()
    })
  }

  const loadPublicSettings = async () => {
    setSettingsLoading(true)
    try {
      const res = await api.get('/api/public/settings')
      setRegisterEnabled(res?.data?.allow_enterprise_register !== false)
    } catch {
      setRegisterEnabled(true)
    } finally {
      setSettingsLoading(false)
    }
  }

  const readDraft = () => {
    try {
      const raw = sessionStorage.getItem(draftStorageKey)
      if (!raw) return null
      const parsed = JSON.parse(raw)
      const step = typeof parsed?.step === 'number' ? Math.min(2, Math.max(0, parsed.step)) : 0
      const values = parsed?.values && typeof parsed.values === 'object' ? parsed.values : undefined
      return { step, values }
    } catch {
      return null
    }
  }

  const initialDraft = readDraft()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [fileList, setFileList] = useState<any[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(() => initialDraft?.step ?? 0)
  const stepRef = useRef(currentStep)
  const draftRef = useRef<Record<string, any>>(initialDraft?.values ?? {})

  const stepFields: string[][] = [
    ['org_name'],
    ['admin_email', 'password', 'confirmPassword'],
    ['agreement']
  ]

  const handleSubmit = async () => {
    if (!registerEnabled) {
      message.warning(t('enterpriseRegisterPage.registerDisabledWarning'))
      return
    }
    try {
      const values = { ...draftRef.current, ...form.getFieldsValue(true) }
      const orgName = (values.org_name || '').trim()
      if (!orgName || orgName === 'undefined' || orgName === 'null') {
        message.error(t('enterpriseRegisterPage.orgNameEmpty'))
        setCurrentStep(0)
        return
      }
      const adminEmail = (values.admin_email || '').trim()
      if (!adminEmail || adminEmail === 'undefined' || adminEmail === 'null') {
        message.error(t('enterpriseRegisterPage.adminEmailEmpty'))
        setCurrentStep(1)
        return
      }
      const password = values.password || ''
      const confirmPassword = values.confirmPassword || ''
      if (!password || password.length < 6) {
        message.error(t('enterpriseRegisterPage.passwordInvalid'))
        setCurrentStep(1)
        return
      }
      if (confirmPassword !== password) {
        message.error(t('enterpriseRegisterPage.confirmPasswordMismatch'))
        setCurrentStep(1)
        return
      }
      await form.validateFields()
      if (fileList.length === 0) {
        message.error(t('enterpriseRegisterPage.materialsRequired'))
        return
      }
      const oversized = fileList.find((file) => (file.size || file.originFileObj?.size || 0) > maxMaterialSize)
      if (oversized) {
        message.error(t('enterpriseRegisterPage.materialsSizeLimit'))
        return
      }
      const formData = new FormData()
      formData.append('org_name', orgName)
      formData.append('admin_email', adminEmail)
      formData.append('password', password)
      fileList.forEach((file) => {
        const raw = file.originFileObj || file
        formData.append('materials', raw)
      })
      setSubmitting(true)
      const res = await api.post('/api/auth/enterprise-register', formData)
      draftRef.current = {}
      sessionStorage.removeItem(draftStorageKey)
      message.success(t('enterpriseRegisterPage.submitSuccess'))
      const orgId = res?.data?.org?.id
      if (orgId) {
        navigate(`/pending?id=${orgId}`)
      } else {
        navigate('/pending')
      }
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || t('enterpriseRegisterPage.submitFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  const goNext = async () => {
    try {
      await form.validateFields(stepFields[currentStep])
      setCurrentStep((prev) => prev + 1)
    } catch {
      return
    }
  }

  const goPrev = () => {
    setCurrentStep((prev) => Math.max(0, prev - 1))
  }

  useEffect(() => {
    loadPublicSettings()
  }, [])

  useEffect(() => {
    stepRef.current = currentStep
    sessionStorage.setItem(draftStorageKey, JSON.stringify({ step: currentStep, values: draftRef.current }))
  }, [currentStep])

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
              {t('enterpriseRegisterPage.brandTitle')}
            </Title>
            <Paragraph style={{ color: 'rgba(255,255,255,0.9)', fontSize: 16, marginBottom: 0 }}>
              {t('enterpriseRegisterPage.brandDescription')}
            </Paragraph>
          </div>

          <div style={{ position: 'absolute', bottom: 40, color: 'rgba(255,255,255,0.6)', fontSize: 13 }}>
            {t('enterpriseRegisterPage.footerCopyright')}
          </div>
        </div>
      )}

      <div
        style={{
          width: isMobile ? '100%' : (isEnglish ? 720 : 640),
          background: '#fff',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: isMobile ? 'flex-start' : 'center',
          padding: isMobile ? '28px 16px 36px' : '60px',
          boxShadow: isMobile ? 'none' : '-4px 0 20px rgba(0,0,0,0.05)'
        }}
      >
        <div style={{ maxWidth: isEnglish ? 600 : 520, margin: '0 auto', width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: isMobile ? 12 : 16 }}>
            <Button icon={<GlobalOutlined />} onClick={handleSwitchLanguage}>
              {languageSwitchLabel}
            </Button>
          </div>

          <div style={{ marginBottom: isMobile ? 18 : 24 }}>
            <RouterLink to="/login" style={{ color: '#1890ff', fontSize: 14, textDecoration: 'none' }}>
              {t('enterpriseRegisterPage.backToLogin')}
            </RouterLink>
          </div>

          <Card>
            {settingsLoading ? (
              <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
                <Spin />
              </div>
            ) : registerEnabled ? (
              <>
                <Steps
                  current={currentStep}
                  items={[
                    {
                      title: (
                        <span style={{ whiteSpace: 'normal', textAlign: 'center', lineHeight: 1.2 }}>
                          {t('enterpriseRegisterPage.stepOrgInfo')}
                        </span>
                      )
                    },
                    {
                      title: (
                        <span style={{ whiteSpace: 'normal', textAlign: 'center', lineHeight: 1.2 }}>
                          {t('enterpriseRegisterPage.stepAdminAccount')}
                        </span>
                      )
                    },
                    {
                      title: (
                        <span style={{ whiteSpace: 'normal', textAlign: 'center', lineHeight: 1.2 }}>
                          {t('enterpriseRegisterPage.stepMaterialsAgreement')}
                        </span>
                      )
                    }
                  ]}
                  style={{ marginBottom: isMobile ? 20 : 24 }}
                  direction={isMobile ? 'vertical' : 'horizontal'}
                  labelPlacement={isMobile ? 'horizontal' : 'vertical'}
                  size={isMobile ? 'small' : 'default'}
                />
                <Form
                  form={form}
                  layout="vertical"
                  onFinish={handleSubmit}
                  size={isMobile ? 'middle' : 'large'}
                  initialValues={initialDraft?.values}
                  onValuesChange={(_, allValues) => {
                    const values = { ...draftRef.current, ...form.getFieldsValue(true) }
                    draftRef.current = values
                    sessionStorage.setItem(draftStorageKey, JSON.stringify({ step: stepRef.current, values }))
                  }}
                >
                  {currentStep === 0 && (
                    <Form.Item
                      name="org_name"
                      label={t('enterpriseRegisterPage.orgNameLabel')}
                      rules={[{ required: true, message: t('enterpriseRegisterPage.orgNameRequired') }]}
                    >
                      <Input
                        prefix={<BankOutlined style={{ color: '#bfbfbf' }} />}
                        placeholder={t('enterpriseRegisterPage.orgNamePlaceholder')}
                      />
                    </Form.Item>
                  )}

                  {currentStep === 1 && (
                    <>
                      <Form.Item
                        name="admin_email"
                        label={t('enterpriseRegisterPage.adminEmailLabel')}
                        rules={[
                          { required: true, message: t('enterpriseRegisterPage.adminEmailRequired') },
                          { type: 'email', message: t('enterpriseRegisterPage.emailInvalid') }
                        ]}
                      >
                        <Input
                          prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                          placeholder={t('enterpriseRegisterPage.adminEmailPlaceholder')}
                        />
                      </Form.Item>
                      <Form.Item
                        name="password"
                        label={t('enterpriseRegisterPage.passwordLabel')}
                        rules={[
                          { required: true, message: t('enterpriseRegisterPage.passwordRequired') },
                          { min: 6, message: t('enterpriseRegisterPage.passwordMin') }
                        ]}
                      >
                        <Input.Password
                          prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                          placeholder={t('enterpriseRegisterPage.passwordPlaceholder')}
                        />
                      </Form.Item>
                      <Form.Item
                        name="confirmPassword"
                        label={t('enterpriseRegisterPage.confirmPasswordLabel')}
                        dependencies={['password']}
                        rules={[
                          { required: true, message: t('enterpriseRegisterPage.confirmPasswordRequired') },
                          ({ getFieldValue }) => ({
                            validator(_, value) {
                              if (!value || getFieldValue('password') === value) {
                                return Promise.resolve()
                              }
                              return Promise.reject(new Error(t('enterpriseRegisterPage.confirmPasswordMismatch')))
                            }
                          })
                        ]}
                      >
                        <Input.Password
                          prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                          placeholder={t('enterpriseRegisterPage.confirmPasswordPlaceholder')}
                        />
                      </Form.Item>
                    </>
                  )}

                  {currentStep === 2 && (
                    <>
                      <Form.Item label={t('enterpriseRegisterPage.materialsLabel')}>
                        <Upload
                          multiple
                          beforeUpload={() => false}
                          fileList={fileList}
                          onChange={({ fileList: nextList }) => setFileList(nextList)}
                        >
                          <Button icon={<UploadOutlined />} style={isMobile ? { width: '100%' } : undefined}>
                            {t('enterpriseRegisterPage.selectMaterials')}
                          </Button>
                        </Upload>
                        <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                          {t('enterpriseRegisterPage.materialsHint')}
                        </Text>
                      </Form.Item>
                      <Form.Item
                        name="agreement"
                        valuePropName="checked"
                        rules={[
                          {
                            validator: (_, value) => value
                              ? Promise.resolve()
                              : Promise.reject(new Error(t('enterpriseRegisterPage.agreementRequired')))
                          }
                        ]}
                      >
                        <Checkbox>
                          {t('enterpriseRegisterPage.agreementPrefix')}
                          <RouterLink to="/terms">{t('enterpriseRegisterPage.terms')}</RouterLink>
                          {t('enterpriseRegisterPage.agreementAnd')}
                          <RouterLink to="/privacy">{t('enterpriseRegisterPage.privacy')}</RouterLink>
                        </Checkbox>
                      </Form.Item>
                    </>
                  )}

                  <Space
                    direction={isMobile ? 'vertical' : 'horizontal'}
                    style={
                      isMobile
                        ? { display: 'flex', marginTop: 8, width: '100%' }
                        : { display: 'flex', justifyContent: 'space-between', marginTop: 8, width: '100%' }
                    }
                  >
                    <Button onClick={goPrev} disabled={currentStep === 0} size={isMobile ? 'middle' : 'large'} style={isMobile ? { width: '100%' } : undefined}>
                      {t('enterpriseRegisterPage.prevStep')}
                    </Button>
                    {currentStep < 2 ? (
                      <Button type="primary" onClick={goNext} size={isMobile ? 'middle' : 'large'} style={isMobile ? { width: '100%' } : undefined}>
                        {t('enterpriseRegisterPage.nextStep')}
                      </Button>
                    ) : (
                      <Button type="primary" htmlType="submit" loading={submitting} size={isMobile ? 'middle' : 'large'} style={isMobile ? { width: '100%' } : undefined}>
                        {t('enterpriseRegisterPage.submitApplication')}
                      </Button>
                    )}
                  </Space>
                </Form>
              </>
            ) : (
              <Alert
                type="warning"
                showIcon
                message={t('enterpriseRegisterPage.registerDisabledWarning')}
              />
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}

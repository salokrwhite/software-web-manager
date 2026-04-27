import { Button, Card, Empty, Form, Grid, Input, InputNumber, Select, Space, Switch, Tabs, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import api, { getErrorMessage } from '../api/client'
import { emitSiteNameUpdated } from '../utils/siteName'

const { Title, Text } = Typography

type SystemSettingsResponse = {
  site_name?: string
  allow_user_register?: boolean
  allow_enterprise_register?: boolean
  mail_activation_enabled?: boolean
  org_plan_types?: string[]
  home_page_announcement_enabled?: boolean
  home_page_announcement_content?: string
  apps_page_announcement_enabled?: boolean
  apps_page_announcement_content?: string
  page_announcement_enabled?: boolean
  page_announcement_content?: string
  smtp_sender_name?: string
  smtp_sender_email?: string
  smtp_host?: string
  smtp_port?: number
  smtp_username?: string
  smtp_conn_ttl_seconds?: number
  smtp_force_ssl?: boolean
  smtp_password_configured?: boolean
  register_email_code_template?: string
  service_status_overall_status?: string
  service_status_overall_message?: string
  service_status_announcement?: string
  service_status_components?: ServiceStatusComponent[]
  service_status_incidents?: ServiceStatusIncident[]
  service_status_updated_at?: string
}

type ServiceStatusComponent = {
  name: string
  status: 'operational' | 'degraded' | 'outage' | 'maintenance'
  description?: string
}

type ServiceStatusIncident = {
  title: string
  status: 'investigating' | 'identified' | 'monitoring' | 'resolved'
  started_at?: string
  updated_at?: string
  description?: string
}

const PLAN_OPTIONS = [
  { label: 'Free', value: 'free' },
  { label: 'Team', value: 'team' },
  { label: 'Enterprise', value: 'enterprise' }
]
const SERVICE_STATUS_OPTIONS = [
  { label: '运行正常', value: 'operational' },
  { label: '部分降级', value: 'degraded' },
  { label: '服务中断', value: 'outage' },
  { label: '维护中', value: 'maintenance' }
]
const INCIDENT_STATUS_OPTIONS = [
  { label: '调查中', value: 'investigating' },
  { label: '已识别', value: 'identified' },
  { label: '监控中', value: 'monitoring' },
  { label: '已恢复', value: 'resolved' }
]
const DEFAULT_SERVICE_COMPONENTS: ServiceStatusComponent[] = [
  { name: 'Web 控制台', status: 'operational', description: '管理后台与控制台访问' },
  { name: 'API 服务', status: 'operational', description: '开放 API 与管理 API' },
  { name: '文件分发', status: 'operational', description: '安装包与制品下载' }
]
const SYSTEM_SETTING_TAB_KEYS = ['base', 'security', 'mail', 'mail-template', 'announcement', 'service-status', 'integration']
const SERVICE_STATUS_TIME_EXAMPLE = '2026-03-04T18:03:32+08:00'

const formatServiceStatusTime = (value: string) => {
  const text = String(value || '').trim()
  if (!text) return ''
  const rfc3339Match = text.match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})(?::(\d{2}))?(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/)
  if (rfc3339Match) {
    const datePart = rfc3339Match[1]
    const secondPart = rfc3339Match[3] || '00'
    const tzPart = rfc3339Match[4] === 'Z' ? '+00:00' : rfc3339Match[4]
    return `${datePart}T${rfc3339Match[2]}:${secondPart}${tzPart}`
  }
  return text
}

const buildCurrentServiceStatusTime = () => {
  const now = new Date()
  const pad = (num: number) => String(num).padStart(2, '0')
  const offsetMinutes = -now.getTimezoneOffset()
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const absMinutes = Math.abs(offsetMinutes)
  const offsetHour = pad(Math.floor(absMinutes / 60))
  const offsetMinute = pad(absMinutes % 60)
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}${sign}${offsetHour}:${offsetMinute}`
}

export default function SystemSettings() {
  const navigate = useNavigate()
  const { tab } = useParams<{ tab?: string }>()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [baseForm] = Form.useForm()
  const [mailForm] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [savingBase, setSavingBase] = useState(false)
  const [savingUserSession, setSavingUserSession] = useState(false)
  const [savingMail, setSavingMail] = useState(false)
  const [savingMailTemplate, setSavingMailTemplate] = useState(false)
  const [savingAnnouncement, setSavingAnnouncement] = useState(false)
  const [savingServiceStatus, setSavingServiceStatus] = useState(false)
  const [testingMail, setTestingMail] = useState(false)
  const [allowUserRegister, setAllowUserRegister] = useState(true)
  const [allowEnterpriseRegister, setAllowEnterpriseRegister] = useState(true)
  const [mailActivationEnabled, setMailActivationEnabled] = useState(false)
  const [orgPlanTypes, setOrgPlanTypes] = useState<string[]>(['free', 'team', 'enterprise'])
  const [homePageAnnouncementEnabled, setHomePageAnnouncementEnabled] = useState(false)
  const [homePageAnnouncementContent, setHomePageAnnouncementContent] = useState('')
  const [appsPageAnnouncementEnabled, setAppsPageAnnouncementEnabled] = useState(false)
  const [appsPageAnnouncementContent, setAppsPageAnnouncementContent] = useState('')
  const [smtpPasswordConfigured, setSMTPPasswordConfigured] = useState(false)
  const [registerEmailCodeTemplate, setRegisterEmailCodeTemplate] = useState('')
  const [serviceStatusOverall, setServiceStatusOverall] = useState<'operational' | 'degraded' | 'outage' | 'maintenance'>('operational')
  const [serviceStatusMessage, setServiceStatusMessage] = useState('所有系统正常运行')
  const [serviceStatusAnnouncement, setServiceStatusAnnouncement] = useState('')
  const [serviceStatusComponents, setServiceStatusComponents] = useState<ServiceStatusComponent[]>(DEFAULT_SERVICE_COMPONENTS)
  const [serviceStatusIncidents, setServiceStatusIncidents] = useState<ServiceStatusIncident[]>([])
  const [serviceStatusUpdatedAt, setServiceStatusUpdatedAt] = useState('')
  const [testMailTo, setTestMailTo] = useState('')
  const activeTab = SYSTEM_SETTING_TAB_KEYS.includes(tab || '') ? (tab as string) : 'base'

  const loadSettings = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/system/settings')
      const data = (res.data || {}) as SystemSettingsResponse
      emitSiteNameUpdated(data.site_name || '')
      baseForm.setFieldsValue({ site_name: data.site_name || '' })
      setOrgPlanTypes(Array.isArray(data.org_plan_types) && data.org_plan_types.length > 0 ? data.org_plan_types : ['free', 'team', 'enterprise'])
      setAllowUserRegister(data.allow_user_register ?? true)
      setAllowEnterpriseRegister(data.allow_enterprise_register ?? true)
      setMailActivationEnabled(data.mail_activation_enabled ?? false)
      const legacyAnnouncementEnabled = data.page_announcement_enabled ?? false
      const legacyAnnouncementContent = data.page_announcement_content || ''
      setHomePageAnnouncementEnabled(data.home_page_announcement_enabled ?? legacyAnnouncementEnabled)
      setHomePageAnnouncementContent(data.home_page_announcement_content ?? legacyAnnouncementContent)
      setAppsPageAnnouncementEnabled(data.apps_page_announcement_enabled ?? legacyAnnouncementEnabled)
      setAppsPageAnnouncementContent(data.apps_page_announcement_content ?? legacyAnnouncementContent)
      setRegisterEmailCodeTemplate(
        data.register_email_code_template ||
          '您好，\n\n您的注册验证码是 {{code}}，有效期 {{minutes}} 分钟。\n\n如果这不是您的操作，请忽略此邮件。\n\n{{site_name}}'
      )
      setServiceStatusOverall(
        (data.service_status_overall_status as 'operational' | 'degraded' | 'outage' | 'maintenance') || 'operational'
      )
      setServiceStatusMessage(data.service_status_overall_message || '所有系统正常运行')
      setServiceStatusAnnouncement(String(data.service_status_announcement || ''))
      setServiceStatusComponents(
        Array.isArray(data.service_status_components) && data.service_status_components.length > 0
          ? data.service_status_components
          : DEFAULT_SERVICE_COMPONENTS
      )
      setServiceStatusIncidents(Array.isArray(data.service_status_incidents) ? data.service_status_incidents : [])
      setServiceStatusUpdatedAt(String(data.service_status_updated_at || ''))

      const passwordConfigured = data.smtp_password_configured === true
      setSMTPPasswordConfigured(passwordConfigured)
      mailForm.setFieldsValue({
        smtp_sender_name: data.smtp_sender_name || '',
        smtp_sender_email: data.smtp_sender_email || '',
        smtp_host: data.smtp_host || '',
        smtp_port: data.smtp_port ?? 465,
        smtp_username: data.smtp_username || '',
        smtp_password: '',
        smtp_conn_ttl_seconds: data.smtp_conn_ttl_seconds ?? 15,
        smtp_force_ssl: data.smtp_force_ssl ?? true
      })
    } catch (err: any) {
      message.error(getErrorMessage(err, '加载系统设置失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSettings()
  }, [])

  useEffect(() => {
    if (!tab) {
      navigate('/system/settings/base', { replace: true })
      return
    }
    if (!SYSTEM_SETTING_TAB_KEYS.includes(tab)) {
      navigate('/system/settings/base', { replace: true })
    }
  }, [tab, navigate])

  const handleSaveBase = async () => {
    const values = await baseForm.validateFields()
    setSavingBase(true)
    try {
      const res = await api.patch('/api/system/settings', {
        site_name: values.site_name,
        org_plan_types: orgPlanTypes
      })
      emitSiteNameUpdated(res?.data?.site_name || values.site_name)
      message.success('系统设置已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存系统设置失败'))
    } finally {
      setSavingBase(false)
    }
  }

  const handleSaveUserSession = async () => {
    setSavingUserSession(true)
    try {
      await api.patch('/api/system/settings', {
        allow_user_register: allowUserRegister,
        allow_enterprise_register: allowEnterpriseRegister,
        mail_activation_enabled: mailActivationEnabled
      })
      message.success('用户会话设置已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存系统设置失败'))
    } finally {
      setSavingUserSession(false)
    }
  }

  const handleSaveAnnouncement = async () => {
    setSavingAnnouncement(true)
    try {
      await api.patch('/api/system/settings', {
        home_page_announcement_enabled: homePageAnnouncementEnabled,
        home_page_announcement_content: homePageAnnouncementContent,
        apps_page_announcement_enabled: appsPageAnnouncementEnabled,
        apps_page_announcement_content: appsPageAnnouncementContent
      })
      message.success('页面公告已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存页面公告失败'))
    } finally {
      setSavingAnnouncement(false)
    }
  }

  const buildSMTPPayload = (values: any) => {
    const payload: Record<string, any> = {
      smtp_sender_name: (values.smtp_sender_name || '').trim(),
      smtp_sender_email: (values.smtp_sender_email || '').trim(),
      smtp_host: (values.smtp_host || '').trim(),
      smtp_port: Number(values.smtp_port),
      smtp_username: (values.smtp_username || '').trim(),
      smtp_conn_ttl_seconds: Number(values.smtp_conn_ttl_seconds),
      smtp_force_ssl: values.smtp_force_ssl === true
    }
    const password = String(values.smtp_password || '').trim()
    if (password !== '') {
      payload.smtp_password = password
    }
    return payload
  }

  const ensureSMTPPassword = (values: any) => {
    const password = String(values.smtp_password || '').trim()
    if (smtpPasswordConfigured) return true
    if (password === '') {
      message.warning('请先配置 SMTP 密码')
      return false
    }
    return true
  }

  const handleSaveMail = async () => {
    const values = await mailForm.validateFields()
    if (!ensureSMTPPassword(values)) return
    const payload = buildSMTPPayload(values)
    setSavingMail(true)
    try {
      await api.patch('/api/system/settings', payload)
      message.success('邮件配置已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存邮件配置失败'))
    } finally {
      setSavingMail(false)
    }
  }

  const handleSaveMailTemplate = async () => {
    const content = registerEmailCodeTemplate.trim()
    if (!content) {
      message.warning('请输入注册邮箱验证码模板')
      return
    }
    setSavingMailTemplate(true)
    try {
      await api.patch('/api/system/settings', {
        register_email_code_template: content
      })
      message.success('邮件模板已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存邮件模板失败'))
    } finally {
      setSavingMailTemplate(false)
    }
  }

  const updateServiceComponent = (index: number, patch: Partial<ServiceStatusComponent>) => {
    setServiceStatusComponents((prev) => prev.map((item, idx) => (idx === index ? { ...item, ...patch } : item)))
  }

  const addServiceComponent = () => {
    setServiceStatusComponents((prev) => [...prev, { name: '', status: 'operational', description: '' }])
  }

  const removeServiceComponent = (index: number) => {
    setServiceStatusComponents((prev) => prev.filter((_, idx) => idx !== index))
  }

  const updateServiceIncident = (index: number, patch: Partial<ServiceStatusIncident>) => {
    setServiceStatusIncidents((prev) => prev.map((item, idx) => (idx === index ? { ...item, ...patch } : item)))
  }

  const addServiceIncident = () => {
    const nowText = buildCurrentServiceStatusTime()
    setServiceStatusIncidents((prev) => [
      ...prev,
      { title: '', status: 'investigating', started_at: nowText, updated_at: nowText, description: '' }
    ])
  }

  const removeServiceIncident = (index: number) => {
    setServiceStatusIncidents((prev) => prev.filter((_, idx) => idx !== index))
  }

  const handleSaveServiceStatus = async () => {
    const components = serviceStatusComponents
      .map((item) => ({
        name: String(item.name || '').trim(),
        status: item.status || 'operational',
        description: String(item.description || '').trim()
      }))
      .filter((item) => item.name !== '')
    const incidents = serviceStatusIncidents
      .map((item) => ({
        title: String(item.title || '').trim(),
        status: item.status || 'investigating',
        started_at: String(item.started_at || '').trim(),
        updated_at: String(item.updated_at || '').trim(),
        description: String(item.description || '').trim()
      }))
      .filter((item) => item.title !== '')

    setSavingServiceStatus(true)
    try {
      await api.patch('/api/system/settings', {
        service_status_overall_status: serviceStatusOverall,
        service_status_overall_message: serviceStatusMessage,
        service_status_announcement: serviceStatusAnnouncement,
        service_status_components: components,
        service_status_incidents: incidents
      })
      message.success('服务状态看板已保存')
      await loadSettings()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存服务状态失败'))
    } finally {
      setSavingServiceStatus(false)
    }
  }

  const handleTestMail = async () => {
    const values = await mailForm.validateFields()
    if (!ensureSMTPPassword(values)) return
    const toEmail = testMailTo.trim()
    if (!toEmail) {
      message.warning('请输入测试收件邮箱')
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(toEmail)) {
      message.warning('请输入有效的测试收件邮箱')
      return
    }
    const payload = buildSMTPPayload(values)
    payload.to_email = toEmail
    setTestingMail(true)
    try {
      await api.post('/api/system/settings/mail/test', payload)
      message.success('测试邮件发送成功')
    } catch (err: any) {
      message.error(getErrorMessage(err, '测试邮件发送失败'))
    } finally {
      setTestingMail(false)
    }
  }

  const baseTab = (
    <Card loading={loading} style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
      <Form form={baseForm} layout="vertical" onFinish={handleSaveBase}>
        <Form.Item
          name="site_name"
          label="站点名称"
          extra="用于配置站点名称，后续可扩展为登录页、导航栏和邮件通知等系统级展示名称。"
          rules={[
            { required: true, message: '请输入站点名称' },
            { max: 100, message: '站点名称最多 100 个字符' }
          ]}
        >
          <Input placeholder="请输入站点名称" maxLength={100} showCount />
        </Form.Item>
        <Form.Item
          label="企业套餐类型"
          extra="由系统管理员配置企业可选套餐，企业管理页面会使用该配置。当前支持 Free、Team、Enterprise。"
        >
          <Select
            mode="multiple"
            value={orgPlanTypes}
            options={PLAN_OPTIONS}
            onChange={(values) => setOrgPlanTypes(values)}
            placeholder="请选择企业套餐类型"
          />
        </Form.Item>
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
          <Button type="primary" htmlType="submit" loading={savingBase} style={isMobile ? { width: '100%' } : undefined}>
            保存
          </Button>
          <Button onClick={loadSettings} disabled={savingBase} style={isMobile ? { width: '100%' } : undefined}>
            重置
          </Button>
        </Space>
      </Form>
    </Card>
  )

  const userSessionTab = (
    <Card style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        <Title level={5} style={{ margin: 0 }}>
          注册与登录
        </Title>
        <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', justifyContent: 'space-between', gap: 16 }}>
          <div>
            <Text style={{ fontSize: 15 }}>允许新用户注册</Text>
            <div>
              <Text type="secondary">关闭后，无法再通过前台注册新的用户。</Text>
            </div>
          </div>
          <Switch checked={allowUserRegister} onChange={setAllowUserRegister} />
        </div>
        <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', justifyContent: 'space-between', gap: 16 }}>
          <div>
            <Text style={{ fontSize: 15 }}>允许企业用户注册</Text>
            <div>
              <Text type="secondary">关闭后，无法再通过前台注册新的企业用户。</Text>
            </div>
          </div>
          <Switch checked={allowEnterpriseRegister} onChange={setAllowEnterpriseRegister} />
        </div>
        <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', justifyContent: 'space-between', gap: 16 }}>
          <div>
            <Text style={{ fontSize: 15 }}>邮件激活</Text>
            <div>
              <Text type="secondary">
                开启后，新用户注册需要点击邮件中的激活链接才能完成。请确认 邮件发信设置 是否正确，否则激活邮件无法送达。
              </Text>
            </div>
          </div>
          <Switch checked={mailActivationEnabled} onChange={setMailActivationEnabled} />
        </div>
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
          <Button type="primary" onClick={handleSaveUserSession} loading={savingUserSession} style={isMobile ? { width: '100%' } : undefined}>
            保存
          </Button>
          <Button onClick={loadSettings} disabled={savingUserSession} style={isMobile ? { width: '100%' } : undefined}>
            重置
          </Button>
        </Space>
      </Space>
    </Card>
  )

  const mailTab = (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }} loading={loading}>
        <Form form={mailForm} layout="vertical">
          <Form.Item name="smtp_sender_name" label="发件人名">
            <Input placeholder="例如：SWM系统通知" />
          </Form.Item>
          <Form.Item
            name="smtp_sender_email"
            label="发件人邮箱"
            rules={[
              { required: true, message: '请输入发件人邮箱' },
              { type: 'email', message: '请输入有效的发件人邮箱' }
            ]}
          >
            <Input placeholder="no-reply@example.com" />
          </Form.Item>
          <Form.Item name="smtp_host" label="SMTP 服务器" rules={[{ required: true, message: '请输入SMTP服务器' }]}>
            <Input placeholder="smtp.example.com" />
          </Form.Item>
          <Form.Item name="smtp_port" label="SMTP 端口" rules={[{ required: true, message: '请输入SMTP端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="smtp_username" label="SMTP 用户名" rules={[{ required: true, message: '请输入SMTP用户名' }]}>
            <Input placeholder="SMTP账号" />
          </Form.Item>
          <Form.Item
            name="smtp_password"
            label="SMTP 密码"
            extra="留空表示不修改当前密码。"
          >
            <Input.Password placeholder="请输入SMTP密码" />
          </Form.Item>
          <Form.Item name="smtp_conn_ttl_seconds" label="SMTP 连接有效期（秒）" rules={[{ required: true, message: '请输入连接有效期' }]}>
            <InputNumber min={1} max={86400} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="smtp_force_ssl"
            label="强制使用 SSL"
            valuePropName="checked"
            extra="是否强制使用 SSL 加密连接。如果无法发送邮件，可关闭此项，会尝试使用 STARTTLS 并决定是否使用加密连接。"
          >
            <Switch />
          </Form.Item>
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Button type="primary" onClick={handleSaveMail} loading={savingMail} style={isMobile ? { width: '100%' } : undefined}>
              保存
            </Button>
            <Button onClick={loadSettings} disabled={savingMail} style={isMobile ? { width: '100%' } : undefined}>
              重置
            </Button>
          </Space>
        </Form>
      </Card>

      <Card title="测试发送" style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Input
            value={testMailTo}
            onChange={(e) => setTestMailTo(e.target.value)}
            placeholder="测试收件邮箱"
          />
          <Button type="primary" onClick={handleTestMail} loading={testingMail} style={isMobile ? { width: '100%' } : undefined}>
            发送测试邮件
          </Button>
        </Space>
      </Card>
    </Space>
  )

  const announcementTab = (
    <Card style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', justifyContent: 'space-between', gap: 16 }}>
          <div>
            <Text style={{ fontSize: 15 }}>首页公告</Text>
            <div>
              <Text type="secondary">开启后，会在官网首页标题上方展示公告栏。</Text>
            </div>
          </div>
          <Switch checked={homePageAnnouncementEnabled} onChange={setHomePageAnnouncementEnabled} />
        </div>
        <div>
          <Text style={{ fontSize: 15 }}>首页公告内容</Text>
          <Input.TextArea
            value={homePageAnnouncementContent}
            onChange={(e) => setHomePageAnnouncementContent(e.target.value)}
            placeholder="请输入首页公告内容"
            rows={4}
            maxLength={500}
            showCount
            disabled={!homePageAnnouncementEnabled}
            style={{ marginTop: 8 }}
          />
        </div>
        <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', justifyContent: 'space-between', gap: 16 }}>
          <div>
            <Text style={{ fontSize: 15 }}>应用管理页面公告</Text>
            <div>
              <Text type="secondary">开启后，会在应用管理页面顶部展示公告栏。</Text>
            </div>
          </div>
          <Switch checked={appsPageAnnouncementEnabled} onChange={setAppsPageAnnouncementEnabled} />
        </div>
        <div>
          <Text style={{ fontSize: 15 }}>应用管理页面公告内容</Text>
          <Input.TextArea
            value={appsPageAnnouncementContent}
            onChange={(e) => setAppsPageAnnouncementContent(e.target.value)}
            placeholder="请输入应用管理页面公告内容"
            rows={4}
            maxLength={500}
            showCount
            disabled={!appsPageAnnouncementEnabled}
            style={{ marginTop: 8 }}
          />
        </div>
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
          <Button type="primary" onClick={handleSaveAnnouncement} loading={savingAnnouncement} style={isMobile ? { width: '100%' } : undefined}>
            保存
          </Button>
          <Button onClick={loadSettings} disabled={savingAnnouncement} style={isMobile ? { width: '100%' } : undefined}>
            重置
          </Button>
        </Space>
      </Space>
    </Card>
  )

  const mailTemplateTab = (
    <Card style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          <Text style={{ fontSize: 15 }}>注册邮箱验证码模板</Text>
          <div>
            <Text type="secondary">{'支持变量：{{code}}、{{minutes}}、{{site_name}}。'}</Text>
          </div>
        </div>
        <Input.TextArea
          value={registerEmailCodeTemplate}
          onChange={(e) => setRegisterEmailCodeTemplate(e.target.value)}
          rows={10}
          maxLength={2000}
          showCount
          placeholder="请输入注册邮箱验证码模板"
        />
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
          <Button
            type="primary"
            onClick={handleSaveMailTemplate}
            loading={savingMailTemplate}
            style={isMobile ? { width: '100%' } : undefined}
          >
            保存
          </Button>
          <Button onClick={loadSettings} disabled={savingMailTemplate} style={isMobile ? { width: '100%' } : undefined}>
            重置
          </Button>
        </Space>
      </Space>
    </Card>
  )

  const serviceStatusTab = (
    <Card style={{ boxShadow: 'none', borderRadius: isMobile ? 8 : 10 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          <Text style={{ fontSize: 15 }}>顶部公告</Text>
          <Input.TextArea
            value={serviceStatusAnnouncement}
            onChange={(e) => setServiceStatusAnnouncement(e.target.value)}
            rows={3}
            maxLength={1000}
            showCount
            placeholder="可填写维护通知、风险提示、计划升级时间等"
            style={{ marginTop: 8 }}
          />
        </div>
        <div>
          <Text style={{ fontSize: 15 }}>总览状态</Text>
          <Select
            style={{ width: '100%', marginTop: 8 }}
            value={serviceStatusOverall}
            options={SERVICE_STATUS_OPTIONS}
            onChange={(value) => setServiceStatusOverall(value as 'operational' | 'degraded' | 'outage' | 'maintenance')}
          />
        </div>
        <div>
          <Text style={{ fontSize: 15 }}>总览说明</Text>
          <Input.TextArea
            value={serviceStatusMessage}
            onChange={(e) => setServiceStatusMessage(e.target.value)}
            rows={3}
            maxLength={300}
            showCount
            placeholder="例如：所有系统正常运行"
            style={{ marginTop: 8 }}
          />
        </div>
        <div>
          <Space style={{ marginBottom: 8, width: '100%', justifyContent: 'space-between' }}>
            <Text style={{ fontSize: 15 }}>服务组件</Text>
            <Button onClick={addServiceComponent}>新增组件</Button>
          </Space>
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            {serviceStatusComponents.map((item, index) => (
              <Card key={`component-${index}`} size="small" style={{ borderRadius: 8 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Input
                    placeholder="组件名称（如 API 服务）"
                    value={item.name}
                    onChange={(e) => updateServiceComponent(index, { name: e.target.value })}
                  />
                  <Select
                    value={item.status}
                    options={SERVICE_STATUS_OPTIONS}
                    onChange={(value) =>
                      updateServiceComponent(index, {
                        status: value as ServiceStatusComponent['status']
                      })
                    }
                  />
                  <Input
                    placeholder="组件说明（可选）"
                    value={item.description}
                    onChange={(e) => updateServiceComponent(index, { description: e.target.value })}
                  />
                  <Button danger onClick={() => removeServiceComponent(index)}>
                    删除组件
                  </Button>
                </Space>
              </Card>
            ))}
          </Space>
        </div>
        <div>
          <Space style={{ marginBottom: 8, width: '100%', justifyContent: 'space-between' }}>
            <Text style={{ fontSize: 15 }}>事件历史</Text>
            <Button onClick={addServiceIncident}>新增事件</Button>
          </Space>
          <Text type="secondary">时间格式：{SERVICE_STATUS_TIME_EXAMPLE}（附带时区，精确到秒）</Text>
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            {serviceStatusIncidents.map((item, index) => (
              <Card key={`incident-${index}`} size="small" style={{ borderRadius: 8 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Input
                    placeholder="事件标题"
                    value={item.title}
                    onChange={(e) => updateServiceIncident(index, { title: e.target.value })}
                  />
                  <Select
                    value={item.status}
                    options={INCIDENT_STATUS_OPTIONS}
                    onChange={(value) =>
                      updateServiceIncident(index, {
                        status: value as ServiceStatusIncident['status']
                      })
                    }
                  />
                  <Input
                    placeholder={`开始时间（可选，如 ${SERVICE_STATUS_TIME_EXAMPLE}）`}
                    value={item.started_at}
                    onChange={(e) => updateServiceIncident(index, { started_at: e.target.value })}
                  />
                  <Input
                    placeholder={`更新时间（可选，如 ${SERVICE_STATUS_TIME_EXAMPLE}）`}
                    value={item.updated_at}
                    onChange={(e) => updateServiceIncident(index, { updated_at: e.target.value })}
                  />
                  <Input.TextArea
                    placeholder="事件说明（可选）"
                    rows={3}
                    value={item.description}
                    onChange={(e) => updateServiceIncident(index, { description: e.target.value })}
                  />
                  <Button danger onClick={() => removeServiceIncident(index)}>
                    删除事件
                  </Button>
                </Space>
              </Card>
            ))}
          </Space>
        </div>
        {serviceStatusUpdatedAt ? (
          <Text type="secondary">最近更新时间：{formatServiceStatusTime(serviceStatusUpdatedAt)}</Text>
        ) : null}
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
          <Button
            type="primary"
            onClick={handleSaveServiceStatus}
            loading={savingServiceStatus}
            style={isMobile ? { width: '100%' } : undefined}
          >
            保存
          </Button>
          <Button onClick={loadSettings} disabled={savingServiceStatus} style={isMobile ? { width: '100%' } : undefined}>
            重置
          </Button>
        </Space>
      </Space>
    </Card>
  )

  const items = [
    { key: 'base', label: '基础设置', children: baseTab, forceRender: true },
    { key: 'security', label: '用户会话', children: userSessionTab, forceRender: true },
    { key: 'mail', label: '邮件', children: mailTab, forceRender: true },
    { key: 'mail-template', label: '邮件模板', children: mailTemplateTab, forceRender: true },
    { key: 'announcement', label: '页面公告', children: announcementTab, forceRender: true },
    { key: 'service-status', label: '服务状态', children: serviceStatusTab, forceRender: true },
    {
      key: 'integration',
      label: '集成配置',
      children: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="集成配置功能建设中" />,
      forceRender: true
    }
  ]

  return (
    <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
      <Title level={isMobile ? 5 : 4} style={{ marginTop: 0, marginBottom: 8 }}>
        系统设置
      </Title>
      <Tabs
        style={{ marginTop: 16 }}
        size={isMobile ? 'small' : 'middle'}
        tabBarGutter={isMobile ? 12 : 24}
        tabBarStyle={{ overflowX: 'auto' }}
        activeKey={activeTab}
        onChange={(key) => navigate(`/system/settings/${key}`)}
        items={items}
      />
    </Card>
  )
}

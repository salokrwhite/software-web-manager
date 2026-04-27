import { Alert, Badge, Button, Card, Col, Empty, Grid, Row, Space, Spin, Tag, Typography } from 'antd'
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, SettingOutlined } from '@ant-design/icons'
import { ReactNode, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Paragraph, Text } = Typography

type ServiceStatusLanguage = 'zh' | 'en'
type ServiceOverallStatus = 'operational' | 'degraded' | 'outage' | 'maintenance'
type ServiceIncidentStatus = 'investigating' | 'identified' | 'monitoring' | 'resolved'

type ServiceStatusComponent = {
  name: string
  status: ServiceOverallStatus
  description?: string
}

type ServiceStatusIncident = {
  title: string
  status: ServiceIncidentStatus
  started_at?: string
  updated_at?: string
  description?: string
}

type PublicSettingsResponse = {
  site_name?: string
  service_status_overall_status?: ServiceOverallStatus
  service_status_overall_message?: string
  service_status_announcement?: string
  service_status_components?: ServiceStatusComponent[]
  service_status_incidents?: ServiceStatusIncident[]
  service_status_updated_at?: string
}

const SERVICE_STATUS_LANGUAGE_STORAGE_KEY = 'swm_service_status_lang'
const DEFAULT_SITE_NAME_BY_LANG: Record<ServiceStatusLanguage, string> = {
  zh: 'SWM 软件版本管理平台',
  en: 'SWM Software Release Manager'
}
const DEFAULT_OVERALL_MESSAGE_BY_LANG: Record<ServiceStatusLanguage, string> = {
  zh: '所有系统正常运行',
  en: 'All systems are operating normally.'
}
const DEFAULT_COMPONENTS_BY_LANG: Record<ServiceStatusLanguage, ServiceStatusComponent[]> = {
  zh: [
    { name: 'Web 控制台', status: 'operational', description: '管理后台与控制台访问' },
    { name: 'API 服务', status: 'operational', description: '开放 API 与管理 API' },
    { name: '文件分发', status: 'operational', description: '安装包与制品下载' }
  ],
  en: [
    { name: 'Web Console', status: 'operational', description: 'Management console and dashboard access' },
    { name: 'API Service', status: 'operational', description: 'Open APIs and management APIs' },
    { name: 'Artifact Delivery', status: 'operational', description: 'Package and artifact download services' }
  ]
}

const OVERALL_META_BY_LANG: Record<ServiceStatusLanguage, Record<ServiceOverallStatus, { color: string; text: string; icon: ReactNode }>> = {
  zh: {
    operational: { color: 'success', text: '全部服务正常', icon: <CheckCircleOutlined /> },
    degraded: { color: 'warning', text: '部分服务降级', icon: <ClockCircleOutlined /> },
    outage: { color: 'error', text: '服务不可用', icon: <CloseCircleOutlined /> },
    maintenance: { color: 'processing', text: '维护中', icon: <SettingOutlined /> }
  },
  en: {
    operational: { color: 'success', text: 'Operational', icon: <CheckCircleOutlined /> },
    degraded: { color: 'warning', text: 'Degraded', icon: <ClockCircleOutlined /> },
    outage: { color: 'error', text: 'Outage', icon: <CloseCircleOutlined /> },
    maintenance: { color: 'processing', text: 'Maintenance', icon: <SettingOutlined /> }
  }
}

const INCIDENT_META_BY_LANG: Record<ServiceStatusLanguage, Record<ServiceIncidentStatus, { color: string; text: string }>> = {
  zh: {
    investigating: { color: 'orange', text: '调查中' },
    identified: { color: 'gold', text: '已识别' },
    monitoring: { color: 'blue', text: '监控中' },
    resolved: { color: 'green', text: '已恢复' }
  },
  en: {
    investigating: { color: 'orange', text: 'Investigating' },
    identified: { color: 'gold', text: 'Identified' },
    monitoring: { color: 'blue', text: 'Monitoring' },
    resolved: { color: 'green', text: 'Resolved' }
  }
}

const PAGE_TEXT_BY_LANG: Record<
  ServiceStatusLanguage,
  {
    titleSuffix: string
    backHome: string
    login: string
    lastUpdated: string
    serviceComponentStatus: string
    incidentHistory: string
    noIncidents: string
    startAt: string
    updatedAt: string
    noDescription: string
  }
> = {
  zh: {
    titleSuffix: '服务状态',
    backHome: '返回首页',
    login: '控制台登录',
    lastUpdated: '最后更新',
    serviceComponentStatus: '服务组件状态',
    incidentHistory: '事件历史',
    noIncidents: '暂无事件',
    startAt: '开始',
    updatedAt: '更新',
    noDescription: '暂无说明'
  },
  en: {
    titleSuffix: 'Service Status',
    backHome: 'Back to Home',
    login: 'Console Login',
    lastUpdated: 'Last Updated',
    serviceComponentStatus: 'Service Components',
    incidentHistory: 'Incident History',
    noIncidents: 'No incidents',
    startAt: 'Started',
    updatedAt: 'Updated',
    noDescription: 'No description'
  }
}

const STATUS_REFRESH_INTERVAL_MS = 5 * 60 * 1000

const getInitialLanguage = (): ServiceStatusLanguage => {
  if (typeof window === 'undefined') return 'zh'
  const saved = window.localStorage.getItem(SERVICE_STATUS_LANGUAGE_STORAGE_KEY)
  if (saved === 'zh' || saved === 'en') return saved
  return window.navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const formatStatusTime = (value: string) => {
  const text = String(value || '').trim()
  if (!text) return ''
  const rfc3339Match = text.match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})(?::(\d{2}))?(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/)
  if (rfc3339Match) {
    const datePart = rfc3339Match[1]
    const secondPart = rfc3339Match[3] || '00'
    const tzPart = rfc3339Match[4] === 'Z' ? '+00:00' : rfc3339Match[4]
    return `${datePart} ${rfc3339Match[2]}:${secondPart} UTC${tzPart}`
  }
  const date = new Date(text)
  if (Number.isNaN(date.getTime())) return text
  const pad = (num: number) => String(num).padStart(2, '0')
  const offsetMinutes = -date.getTimezoneOffset()
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const absMinutes = Math.abs(offsetMinutes)
  const offsetHour = pad(Math.floor(absMinutes / 60))
  const offsetMinute = pad(absMinutes % 60)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())} UTC${sign}${offsetHour}:${offsetMinute}`
}

export default function ServiceStatusPage() {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [language, setLanguage] = useState<ServiceStatusLanguage>(getInitialLanguage)
  const [loading, setLoading] = useState(true)
  const [siteName, setSiteName] = useState('')
  const [overall, setOverall] = useState<ServiceOverallStatus>('operational')
  const [overallMessage, setOverallMessage] = useState('')
  const [announcement, setAnnouncement] = useState('')
  const [components, setComponents] = useState<ServiceStatusComponent[]>([])
  const [incidents, setIncidents] = useState<ServiceStatusIncident[]>([])
  const [updatedAt, setUpdatedAt] = useState('')

  useEffect(() => {
    window.localStorage.setItem(SERVICE_STATUS_LANGUAGE_STORAGE_KEY, language)
  }, [language])

  useEffect(() => {
    let active = true
    const loadStatus = async (silent = false) => {
      if (!silent) setLoading(true)
      try {
        const res = await api.get('/api/public/settings')
        if (!active) return
        const data = (res.data || {}) as PublicSettingsResponse
        setSiteName(String(data.site_name || ''))
        setOverall(data.service_status_overall_status || 'operational')
        setOverallMessage(String(data.service_status_overall_message || ''))
        setAnnouncement(String(data.service_status_announcement || ''))
        setComponents(Array.isArray(data.service_status_components) ? data.service_status_components : [])
        setIncidents(Array.isArray(data.service_status_incidents) ? data.service_status_incidents : [])
        setUpdatedAt(String(data.service_status_updated_at || ''))
      } finally {
        if (!silent && active) setLoading(false)
      }
    }
    loadStatus(false)
    const timer = window.setInterval(() => {
      loadStatus(true)
    }, STATUS_REFRESH_INTERVAL_MS)
    return () => {
      active = false
      window.clearInterval(timer)
    }
  }, [])

  const pageText = useMemo(() => PAGE_TEXT_BY_LANG[language], [language])
  const overallMetaMap = useMemo(() => OVERALL_META_BY_LANG[language], [language])
  const incidentMetaMap = useMemo(() => INCIDENT_META_BY_LANG[language], [language])
  const displaySiteName = useMemo(() => siteName || DEFAULT_SITE_NAME_BY_LANG[language], [language, siteName])
  const displayOverallMessage = useMemo(
    () => overallMessage || DEFAULT_OVERALL_MESSAGE_BY_LANG[language],
    [language, overallMessage]
  )
  const displayComponents = useMemo(
    () => (components.length > 0 ? components : DEFAULT_COMPONENTS_BY_LANG[language]),
    [components, language]
  )
  const overallMeta = useMemo(
    () => overallMetaMap[overall] || overallMetaMap.operational,
    [overall, overallMetaMap]
  )

  return (
    <div style={{ minHeight: '100vh', background: '#f5f8fb' }}>
      <header style={{ background: '#001529', color: '#fff', padding: isMobile ? '12px 16px' : '16px 24px' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: isMobile ? 'wrap' : 'nowrap', rowGap: 10 }}>
          <Space>
            <Badge status="processing" />
            <Text style={{ color: '#fff', fontSize: isMobile ? 16 : 18, fontWeight: 600 }}>
              {displaySiteName} {pageText.titleSuffix}
            </Text>
          </Space>
          <Space wrap>
            <Button size={isMobile ? 'middle' : 'large'} type={language === 'zh' ? 'primary' : 'default'} onClick={() => setLanguage('zh')}>
              中文
            </Button>
            <Button size={isMobile ? 'middle' : 'large'} type={language === 'en' ? 'primary' : 'default'} onClick={() => setLanguage('en')}>
              EN
            </Button>
            <Button onClick={() => navigate('/')} size={isMobile ? 'middle' : 'large'}>
              {pageText.backHome}
            </Button>
            <Button type="primary" onClick={() => navigate('/login')} size={isMobile ? 'middle' : 'large'}>
              {pageText.login}
            </Button>
          </Space>
        </div>
      </header>

      <main style={{ maxWidth: 1200, margin: '0 auto', padding: isMobile ? '16px' : '24px' }}>
        {loading ? (
          <div style={{ padding: '64px 0', textAlign: 'center' }}>
            <Spin />
          </div>
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {announcement ? (
              <Alert
                type="warning"
                showIcon
                message={<div style={{ whiteSpace: 'pre-wrap' }}>{announcement}</div>}
              />
            ) : null}
            <Card style={{ borderRadius: 12 }}>
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Space align="center">
                  <Tag color={overallMeta.color} icon={overallMeta.icon as any}>
                    {overallMeta.text}
                  </Tag>
                  {updatedAt ? <Text type="secondary">{pageText.lastUpdated}: {formatStatusTime(updatedAt)}</Text> : null}
                </Space>
                <Alert
                  type={overallMeta.color === 'error' ? 'error' : overallMeta.color === 'warning' ? 'warning' : 'success'}
                  showIcon
                  message={displayOverallMessage}
                />
              </Space>
            </Card>

            <Card title={pageText.serviceComponentStatus} style={{ borderRadius: 12 }}>
              <Row gutter={isMobile ? [12, 12] : [16, 16]}>
                {displayComponents.map((item, index) => (
                  <Col xs={24} md={12} lg={8} key={`${item.name}-${index}`}>
                    <Card size="small" style={{ borderRadius: 10 }}>
                      <Space direction="vertical" size={8} style={{ width: '100%' }}>
                        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                          <Text strong>{item.name}</Text>
                          <Tag color={(overallMetaMap[item.status] || overallMetaMap.operational).color}>
                            {(overallMetaMap[item.status] || overallMetaMap.operational).text}
                          </Tag>
                        </Space>
                        <Text type="secondary">{item.description || pageText.noDescription}</Text>
                      </Space>
                    </Card>
                  </Col>
                ))}
              </Row>
            </Card>

            <Card title={pageText.incidentHistory} style={{ borderRadius: 12 }}>
              {incidents.length === 0 ? (
                <Empty description={pageText.noIncidents} />
              ) : (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  {incidents.map((item, index) => {
                    const meta = incidentMetaMap[item.status] || incidentMetaMap.investigating
                    return (
                      <Card key={`${item.title}-${index}`} size="small" style={{ borderRadius: 10 }}>
                        <Space direction="vertical" size={6} style={{ width: '100%' }}>
                          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                            <Text strong>{item.title}</Text>
                            <Tag color={meta.color}>{meta.text}</Tag>
                          </Space>
                          {item.description ? <Paragraph style={{ marginBottom: 0 }}>{item.description}</Paragraph> : null}
                          <Space wrap>
                            {item.started_at ? <Text type="secondary">{pageText.startAt}: {formatStatusTime(item.started_at)}</Text> : null}
                            {item.updated_at ? <Text type="secondary">{pageText.updatedAt}: {formatStatusTime(item.updated_at)}</Text> : null}
                          </Space>
                        </Space>
                      </Card>
                    )
                  })}
                </Space>
              )}
            </Card>
          </Space>
        )}
      </main>
    </div>
  )
}

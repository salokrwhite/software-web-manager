import {
  Card,
  DatePicker,
  Form,
  Table,
  Button,
  message,
  Typography,
  Row,
  Col,
  Select,
  Statistic,
  Progress,
  theme,
  Space,
  Tag,
  Avatar,
  Empty,
  Spin,
  Grid
} from 'antd'
import {
  BarChartOutlined,
  SearchOutlined,
  DownloadOutlined,
  ReloadOutlined,
  RiseOutlined,
  FallOutlined,
  UserOutlined,
  MobileOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  EyeOutlined
} from '@ant-design/icons'
import { useState, useEffect } from 'react'
import api from '../api/client'
import dayjs from 'dayjs'

const { Title, Text } = Typography
const { RangePicker } = DatePicker
const { Option } = Select

interface App {
  ID: string
  Name: string
  Slug: string
}

interface OverviewItem {
  EventName: string
  Count: number
}

interface VersionItem {
  Version: string
  Count: number
}

interface AnalyticsRefreshResponse {
  ok: boolean
  rows_affected: number
  from: string
  to: string
  refreshed_at: string
}

export default function Analytics() {
  const [apps, setApps] = useState<App[]>([])
  const [selectedApp, setSelectedApp] = useState<string>('')
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs()
  ])
  const [overview, setOverview] = useState<OverviewItem[]>([])
  const [versions, setVersions] = useState<VersionItem[]>([])
  const [failures, setFailures] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const { token } = theme.useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg

  // 加载应用列表
  useEffect(() => {
    loadApps()
  }, [])

  const loadApps = async () => {
    try {
      const res = await api.get('/api/apps')
      const items = res.data.items || []
      setApps(items)
      if (items.length > 0 && !selectedApp) {
        setSelectedApp(items[0].ID)
        loadAnalytics(items[0].ID)
      }
    } catch (err) {
      message.error('加载应用列表失败')
    }
  }

  const loadAnalytics = async (
    appId: string = selectedApp,
    range: [dayjs.Dayjs, dayjs.Dayjs] = dateRange,
    withRefresh: boolean = false
  ) => {
    if (!appId) return
    
    setLoading(true)
    try {
      const params = {
        app_id: appId,
        from: range[0].format('YYYY-MM-DD'),
        to: range[1].format('YYYY-MM-DD')
      }
      if (withRefresh) {
        const refreshRes = await api.post<AnalyticsRefreshResponse>('/api/analytics/refresh', params)
        if (refreshRes?.data?.ok) {
          message.success('数据已刷新', 1)
        }
      }
      
      const [overviewRes, versionsRes, failuresRes] = await Promise.all([
        api.get('/api/analytics/overview', { params }),
        api.get('/api/analytics/versions', { params }),
        api.get('/api/analytics/failures', { params })
      ])
      
      setOverview(overviewRes.data.items || [])
      setVersions(versionsRes.data.items || [])
      setFailures(failuresRes.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载数据失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = () => {
    loadAnalytics(selectedApp, dateRange, true)
  }

  const handleAppChange = (appId: string) => {
    setSelectedApp(appId)
    loadAnalytics(appId, dateRange, false)
  }

  const handleDateChange = (dates: any) => {
    if (dates) {
      setDateRange(dates)
      loadAnalytics(selectedApp, dates, true)
    }
  }

  // 计算统计数据
  const getEventCount = (eventName: string) => {
    const item = overview.find(i => i.EventName === eventName)
    return item?.Count || 0
  }

  const checkUpdateCount = getEventCount('check_update')
  const downloadCompleted = getEventCount('download_completed')
  const installCompleted = getEventCount('install_completed')
  const appStarted = getEventCount('app_started')
  const updateAvailable = getEventCount('update_available')
  const downloadStarted = getEventCount('download_started')
  const updateFailed = getEventCount('update_failed')
  const updateSuccessRate = updateAvailable > 0 ? Math.round((installCompleted / updateAvailable) * 100) : 0

  const stats = [
    {
      title: '更新检查',
      value: checkUpdateCount,
      icon: <SearchOutlined />,
      color: token.colorPrimary
    },
    {
      title: '下载完成',
      value: downloadCompleted,
      icon: <CheckCircleOutlined />,
      color: token.colorSuccess
    },
    {
      title: '发现更新',
      value: updateAvailable,
      icon: <EyeOutlined />,
      color: token.colorPrimary
    },
    {
      title: '开始下载',
      value: downloadStarted,
      icon: <DownloadOutlined />,
      color: token.colorInfo
    },
    {
      title: '安装完成',
      value: installCompleted,
      icon: <MobileOutlined />,
      color: token.colorWarning
    },
    {
      title: '应用启动',
      value: appStarted,
      icon: <UserOutlined />,
      color: token.colorInfo
    },
    {
      title: '更新失败',
      value: updateFailed,
      icon: <CloseCircleOutlined />,
      color: token.colorError
    }
  ]

  // 计算漏斗数据
  const funnelData = [
    { stage: '检查更新', count: checkUpdateCount, key: 'check_update' },
    { stage: '发现更新', count: getEventCount('update_available'), key: 'update_available' },
    { stage: '开始下载', count: getEventCount('download_started'), key: 'download_started' },
    { stage: '下载完成', count: downloadCompleted, key: 'download_completed' },
    { stage: '安装完成', count: installCompleted, key: 'install_completed' }
  ].filter(item => item.count > 0)

  const maxFunnelCount = Math.max(...funnelData.map(f => f.count), 1)

  // 计算版本分布
  const totalVersionUsers = versions.reduce((sum, v) => sum + v.Count, 0)
  const versionData = versions.map((v, index) => ({
    version: v.Version || 'unknown',
    users: v.Count,
    percent: totalVersionUsers > 0 ? Math.round((v.Count / totalVersionUsers) * 100) : 0,
    color: index === 0 ? token.colorPrimary : index === 1 ? token.colorSuccess : index === 2 ? token.colorWarning : token.colorError
  }))

  // 事件名称映射
  const eventNameMap: Record<string, string> = {
    'check_update': '检查更新',
    'update_available': '发现更新',
    'download_started': '开始下载',
    'download_completed': '下载完成',
    'install_started': '开始安装',
    'install_completed': '安装完成',
    'app_started': '应用启动',
    'update_failed': '更新失败',
    'update_cancelled': '更新取消'
  }

  return (
    <div>
      {/* 页面标题 */}
      <Row
        justify="space-between"
        align={isMobile ? 'top' : 'middle'}
        style={{ marginBottom: isMobile ? 16 : 24 }}
        gutter={isMobile ? [12, 12] : undefined}
      >
        <Col xs={24} lg={12}>
          <Space direction="vertical" size={4}>
            <Title level={4} style={{ margin: 0 }}>数据分析</Title>
            <Text type="secondary">查看应用版本更新数据和分析报告</Text>
          </Space>
        </Col>
        <Col xs={24} lg={12}>
          <Space style={isMobile ? { width: '100%' } : { width: '100%', justifyContent: 'flex-end' }}>
            <Button
              icon={<ReloadOutlined />}
              onClick={handleRefresh}
              loading={loading}
              style={isMobile ? { width: '100%' } : undefined}
            >
              刷新数据
            </Button>
          </Space>
        </Col>
      </Row>

      {/* 筛选条件 */}
      <Card
        style={{
          marginBottom: isMobile ? 16 : 24,
          borderRadius: isMobile ? 10 : 12,
          border: 'none',
          boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
        }}
        styles={{ body: { padding: isMobile ? 16 : 24 } }}
      >
        <Space
          size={isMobile ? 12 : 16}
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%' }}
        >
          <div style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>选择应用</Text>
            <Select
              value={selectedApp}
              onChange={handleAppChange}
              style={{ width: isMobile ? '100%' : 200 }}
              placeholder="选择应用"
            >
              {apps.map(app => (
                <Option key={app.ID} value={app.ID}>{app.Name}</Option>
              ))}
            </Select>
          </div>
          <div style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>时间范围</Text>
            <RangePicker 
              value={dateRange}
              onChange={handleDateChange}
              style={{ width: isMobile ? '100%' : 280 }}
            />
          </div>
          <div style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>&nbsp;</Text>
            <Button
              type="primary"
              icon={<BarChartOutlined />}
              onClick={handleRefresh}
              loading={loading}
              style={isMobile ? { width: '100%' } : undefined}
            >
              分析
            </Button>
          </div>
        </Space>
      </Card>

      {/* 统计卡片 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]} style={{ marginBottom: isMobile ? 16 : 24 }}>
        {stats.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card
              loading={loading}
              style={{
                borderRadius: isMobile ? 10 : 12,
                border: 'none',
                boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
              }}
              styles={{ body: { padding: isMobile ? 16 : 24 } }}
            >
              <Space align="start" size={isMobile ? 12 : 16}>
                <div
                  style={{
                    width: isMobile ? 42 : 48,
                    height: isMobile ? 42 : 48,
                    borderRadius: isMobile ? 10 : 12,
                    background: `${stat.color}15`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: isMobile ? 20 : 24,
                    color: stat.color
                  }}
                >
                  {stat.icon}
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: isMobile ? 13 : 14 }}>{stat.title}</Text>
                  <div style={{ marginTop: 4 }}>
                    <Text strong style={{ fontSize: isMobile ? 22 : 24 }}>{stat.value.toLocaleString()}</Text>
                  </div>
                </div>
              </Space>
            </Card>
          </Col>
        ))}
        <Col xs={24} sm={12} lg={6}>
          <Card
            loading={loading}
            style={{
              borderRadius: isMobile ? 10 : 12,
              border: 'none',
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
            }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            <Space align="start" size={isMobile ? 12 : 16}>
              <div
                style={{
                  width: isMobile ? 42 : 48,
                  height: isMobile ? 42 : 48,
                  borderRadius: isMobile ? 10 : 12,
                  background: `${token.colorSuccess}15`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: isMobile ? 20 : 24,
                  color: token.colorSuccess
                }}
              >
                <CheckCircleOutlined />
              </div>
              <div>
                <Text type="secondary" style={{ fontSize: isMobile ? 13 : 14 }}>更新成功率</Text>
                <div style={{ marginTop: 4 }}>
                  <Text strong style={{ fontSize: isMobile ? 22 : 24 }}>{updateSuccessRate}%</Text>
                </div>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>

      {/* 主要内容区 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]}>
        {/* 更新漏斗 */}
        <Col xs={24} lg={12}>
          <Card
            title="更新漏斗"
            extra={<Text type="secondary">{dateRange[0].format('MM-DD')} 至 {dateRange[1].format('MM-DD')}</Text>}
            style={{
              borderRadius: isMobile ? 10 : 12,
              border: 'none',
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
            }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {funnelData.length === 0 ? (
              <Empty description="暂无漏斗数据" />
            ) : (
              <Space direction="vertical" size={isMobile ? 14 : 24} style={{ width: '100%' }}>
                {funnelData.map((item, index) => {
                  const percent = Math.round((item.count / maxFunnelCount) * 100)
                  return (
                    <div key={item.key}>
                      <Row justify="space-between" style={{ marginBottom: 8 }}>
                        <Space size={isMobile ? 6 : 8}>
                          <Avatar
                            size="small"
                            style={{
                              background: index === 0 ? token.colorPrimary :
                                         index === 1 ? token.colorSuccess :
                                         index === 2 ? token.colorWarning :
                                         index === 3 ? token.colorError :
                                         token.colorInfo
                            }}
                          >
                            {index + 1}
                          </Avatar>
                          <Text strong style={{ fontSize: isMobile ? 13 : 14 }}>{item.stage}</Text>
                        </Space>
                        <Space size={isMobile ? 4 : 8} wrap>
                          <Text style={{ fontSize: isMobile ? 13 : 14 }}>{item.count.toLocaleString()}</Text>
                          <Text type="secondary" style={{ fontSize: isMobile ? 12 : 14 }}>({percent}%)</Text>
                        </Space>
                      </Row>
                      <Progress
                        percent={percent}
                        showInfo={false}
                        strokeColor={
                          index === 0 ? token.colorPrimary :
                          index === 1 ? token.colorSuccess :
                          index === 2 ? token.colorWarning :
                          index === 3 ? token.colorError :
                          token.colorInfo
                        }
                        strokeWidth={10}
                        style={{ margin: 0 }}
                      />
                    </div>
                  )
                })}
              </Space>
            )}
          </Card>
        </Col>

        {/* 版本分布 */}
        <Col xs={24} lg={12}>
          <Card
            title="版本分布"
            extra={<Text type="secondary">活跃用户数</Text>}
            style={{
              borderRadius: isMobile ? 10 : 12,
              border: 'none',
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
            }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {versionData.length === 0 ? (
              <Empty description="暂无版本数据" />
            ) : (
              <Space direction="vertical" size={isMobile ? 14 : 20} style={{ width: '100%' }}>
                {versionData.map((item, index) => (
                  <div key={index}>
                    <Row justify="space-between" style={{ marginBottom: 8 }}>
                      <Space size={isMobile ? 4 : 8} wrap>
                        <Tag color={index === 0 ? 'blue' : index === 1 ? 'green' : 'default'}>
                          {item.version}
                        </Tag>
                        {index === 0 && <Tag color="success">最新</Tag>}
                      </Space>
                      <Space size={isMobile ? 4 : 8} wrap>
                        <Text type="secondary" style={{ fontSize: isMobile ? 12 : 14 }}>{item.users.toLocaleString()} 用户</Text>
                        <Text strong>{item.percent}%</Text>
                      </Space>
                    </Row>
                    <Progress
                      percent={item.percent}
                      showInfo={false}
                      strokeColor={item.color}
                      strokeWidth={8}
                      style={{ margin: 0 }}
                    />
                  </div>
                ))}
              </Space>
            )}
          </Card>
        </Col>
      </Row>

      {/* 事件总览 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]} style={{ marginTop: isMobile ? 12 : 24 }}>
        <Col xs={24}>
          <Card
            title="事件总览"
            style={{
              borderRadius: isMobile ? 10 : 12,
              border: 'none',
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
            }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {overview.length === 0 ? (
              <Empty description="暂无事件数据" />
            ) : (
              <Table
                rowKey="EventName"
                dataSource={overview}
                loading={loading}
                size={isMobile ? 'small' : 'middle'}
                scroll={isMobile ? { x: 620 } : undefined}
                pagination={{ pageSize: isMobile ? 6 : 10, showSizeChanger: !isMobile }}
                columns={[
                  {
                    title: '事件',
                    dataIndex: 'EventName',
                    render: (name: string) => (
                      <Space size={isMobile ? 6 : 8}>
                        {name.includes('completed') ? <CheckCircleOutlined style={{ color: token.colorSuccess }} /> :
                         name.includes('cancelled') || name.includes('failed') ? <CloseCircleOutlined style={{ color: token.colorError }} /> :
                         <ClockCircleOutlined style={{ color: token.colorPrimary }} />}
                        <Text>{eventNameMap[name] || name}</Text>
                      </Space>
                    )
                  },
                  {
                    title: '数量',
                    dataIndex: 'Count',
                    align: 'right',
                    render: (count: number) => <Text strong>{count.toLocaleString()}</Text>
                  },
                  {
                    title: '占比',
                    align: 'right',
                    render: (_: any, record: OverviewItem) => {
                      const total = overview.reduce((sum, i) => sum + i.Count, 0)
                      const percent = total > 0 ? Math.round((record.Count / total) * 100) : 0
                      return (
                        <Space size={isMobile ? 4 : 8}>
                          <Progress percent={percent} size="small" style={{ width: isMobile ? 72 : 100 }} showInfo={false} />
                          <Text type="secondary">{percent}%</Text>
                        </Space>
                      )
                    }
                  }
                ]}
              />
            )}
          </Card>
        </Col>
        <Col xs={24}>
          <Card
            title="失败原因 Top"
            style={{
              borderRadius: isMobile ? 10 : 12,
              border: 'none',
              boxShadow: '0 2px 8px rgba(0,0,0,0.04)'
            }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {failures.length === 0 ? (
              <Empty description="暂无失败数据" />
            ) : (
              <Table
                rowKey="Reason"
                dataSource={failures}
                size={isMobile ? 'small' : 'middle'}
                scroll={isMobile ? { x: 480 } : undefined}
                pagination={{ pageSize: isMobile ? 6 : 10, showSizeChanger: !isMobile }}
                columns={[
                  { title: '原因', dataIndex: 'Reason' },
                  { title: '次数', dataIndex: 'Count', render: (c: number) => c.toLocaleString() }
                ]}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}

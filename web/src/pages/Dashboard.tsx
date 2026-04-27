import {
  Card,
  Col,
  Row,
  Statistic,
  Typography,
  Space,
  Tag,
  Progress,
  List,
  Avatar,
  theme,
  Button,
  Select,
  Empty,
  Spin,
  Grid
} from 'antd'
import {
  AppstoreOutlined,
  DownloadOutlined,
  EyeOutlined,
  RocketOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  BarChartOutlined
} from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api/client'

const { Title, Text } = Typography
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

export default function Dashboard() {
  const [apps, setApps] = useState<App[]>([])
  const [selectedApp, setSelectedApp] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [overviewData, setOverviewData] = useState<OverviewItem[]>([])
  const [versionData, setVersionData] = useState<VersionItem[]>([])
  const [refreshingAnalytics, setRefreshingAnalytics] = useState(false)
  const { token } = theme.useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const userEmail = sessionStorage.getItem('user_email') || '管理员'

  // 加载应用列表
  useEffect(() => {
    loadApps()
  }, [])

  // 当选择应用时加载数据
  useEffect(() => {
    if (selectedApp) {
      loadAnalytics(selectedApp)
    }
  }, [selectedApp])

  const loadApps = async () => {
    try {
      const res = await api.get('/api/apps')
      const items = res.data.items || []
      setApps(items)
      if (items.length > 0) {
        setSelectedApp(items[0].ID)
      }
      setLoading(false)
    } catch (err) {
      setLoading(false)
    }
  }

  const loadAnalytics = async (appId: string) => {
    setLoading(true)
    try {
      const [overviewRes, versionsRes] = await Promise.all([
        api.get('/api/analytics/overview', { params: { app_id: appId } }),
        api.get('/api/analytics/versions', { params: { app_id: appId } })
      ])
      setOverviewData(overviewRes.data.items || [])
      setVersionData(versionsRes.data.items || [])
    } catch (err) {
      console.error('加载分析数据失败:', err)
    } finally {
      setLoading(false)
    }
  }

  const refreshAnalytics = async () => {
    if (!selectedApp) return
    setRefreshingAnalytics(true)
    try {
      await api.post<AnalyticsRefreshResponse>('/api/analytics/refresh', { app_id: selectedApp })
      await loadAnalytics(selectedApp)
    } finally {
      setRefreshingAnalytics(false)
    }
  }

  // 计算统计数据
  const getEventCount = (eventName: string) => {
    const item = overviewData.find(i => i.EventName === eventName)
    return item?.Count || 0
  }

  const checkUpdateCount = getEventCount('check_update')
  const downloadCount = getEventCount('download_completed')
  const activeReleases = versionData.length

  const stats = [
    {
      title: '应用数量',
      value: apps.length,
      icon: <AppstoreOutlined style={{ color: token.colorPrimary }} />,
      color: token.colorPrimary
    },
    {
      title: '更新检查次数',
      value: checkUpdateCount,
      icon: <EyeOutlined style={{ color: token.colorSuccess }} />,
      color: token.colorSuccess
    },
    {
      title: '下载完成次数',
      value: downloadCount,
      icon: <DownloadOutlined style={{ color: token.colorWarning }} />,
      color: token.colorWarning
    },
    {
      title: '活跃版本数',
      value: activeReleases,
      icon: <RocketOutlined style={{ color: token.colorInfo }} />,
      color: token.colorInfo
    }
  ]

  // 计算版本分布百分比
  const totalVersionUsers = versionData.reduce((sum, v) => sum + v.Count, 0)
  const versionDistribution = versionData.map((v, index) => ({
    version: v.Version || 'unknown',
    users: v.Count,
    percent: totalVersionUsers > 0 ? Math.round((v.Count / totalVersionUsers) * 100) : 0,
    color: index === 0 ? token.colorPrimary : index === 1 ? token.colorSuccess : index === 2 ? token.colorWarning : token.colorError
  }))

  // 生成最近活动数据（从事件数据转换）
  const recentActivities = overviewData
    .filter(item => item.Count > 0)
    .slice(0, 5)
    .map((item, index) => ({
      id: index,
      action: getEventDisplayName(item.EventName),
      count: item.Count,
      time: '最近30天',
      status: item.EventName.includes('completed') ? 'success' : item.EventName.includes('failed') ? 'error' : 'warning'
    }))

  function getEventDisplayName(eventName: string): string {
    const map: Record<string, string> = {
      'check_update': '检查更新',
      'update_available': '发现更新',
      'download_started': '开始下载',
      'download_completed': '下载完成',
      'install_completed': '安装完成',
      'app_started': '应用启动'
    }
    return map[eventName] || eventName
  }

  return (
    <div>
      {/* 欢迎区域 */}
      <Card
        style={{
          marginBottom: isMobile ? 16 : 24,
          background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
          border: 'none'
        }}
        styles={{ body: { padding: isMobile ? 16 : 24 } }}
      >
        <Row align={isMobile ? 'top' : 'middle'} justify="space-between" gutter={isMobile ? [12, 12] : undefined}>
          <Col xs={24} lg={12}>
            <Title level={isMobile ? 4 : 3} style={{ color: '#fff', margin: 0 }}>
              欢迎回来，{userEmail}
            </Title>
            <Text style={{ color: 'rgba(255,255,255,0.85)', fontSize: isMobile ? 14 : 16 }}>
              今日系统运行正常，共有 {checkUpdateCount.toLocaleString()} 次更新检查
            </Text>
          </Col>
          <Col xs={24} lg={12}>
            <Space
              direction={isMobile ? 'vertical' : 'horizontal'}
              size={isMobile ? 10 : 8}
              style={isMobile ? { width: '100%' } : { width: '100%', justifyContent: 'flex-end' }}
              wrap={!isMobile}
            >
              <Select
                value={selectedApp}
                onChange={setSelectedApp}
                style={{ width: isMobile ? '100%' : 220 }}
                placeholder="选择应用"
              >
                {apps.map(app => (
                  <Option key={app.ID} value={app.ID}>{app.Name}</Option>
                ))}
              </Select>
              <Button
                icon={<BarChartOutlined />}
                onClick={refreshAnalytics}
                loading={refreshingAnalytics}
                disabled={!selectedApp}
                style={isMobile ? { width: '100%' } : undefined}
              >
                刷新分析
              </Button>
              <Button
                type="primary"
                style={isMobile ? { width: '100%', background: '#fff', color: token.colorPrimary } : { background: '#fff', color: token.colorPrimary }}
                disabled={!selectedApp}
              >
                <Link to={selectedApp ? `/apps/${selectedApp}/releases` : '/apps'} style={{ color: 'inherit' }}>
                  管理应用
                </Link>
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 统计卡片 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]} style={{ marginBottom: isMobile ? 16 : 24 }}>
        {stats.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card
              loading={loading}
              style={{
                borderRadius: isMobile ? 10 : 12,
                boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
                border: 'none'
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
                    fontSize: isMobile ? 20 : 24
                  }}
                >
                  {stat.icon}
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: isMobile ? 13 : 14 }}>{stat.title}</Text>
                  <div style={{ marginTop: 4 }}>
                    <Text strong style={{ fontSize: isMobile ? 24 : 28 }}>{stat.value.toLocaleString()}</Text>
                  </div>
                </div>
              </Space>
            </Card>
          </Col>
        ))}
      </Row>

      {/* 下方内容区 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]}>
        {/* 事件总览 */}
        <Col xs={24} lg={14}>
          <Card
            title="事件总览"
            extra={<Link to="/analytics">查看全部</Link>}
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {overviewData.length === 0 ? (
              <Empty description="暂无数据" />
            ) : (
              <List
                dataSource={recentActivities}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta
                      avatar={
                        <Avatar
                          icon={
                            item.status === 'success' ? <CheckCircleOutlined /> :
                            item.status === 'warning' ? <ClockCircleOutlined /> :
                            <ExclamationCircleOutlined />
                          }
                          style={{
                            background:
                              item.status === 'success' ? token.colorSuccess + '20' :
                              item.status === 'warning' ? token.colorWarning + '20' :
                              token.colorError + '20',
                            color:
                              item.status === 'success' ? token.colorSuccess :
                              item.status === 'warning' ? token.colorWarning :
                              token.colorError
                          }}
                        />
                      }
                      title={
                        <Space wrap size={isMobile ? 6 : 8}>
                          <Text strong>{item.action}</Text>
                          <Tag>{item.count.toLocaleString()} 次</Tag>
                        </Space>
                      }
                      description={item.time}
                    />
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>

        {/* 版本分布 */}
        <Col xs={24} lg={10}>
          <Card
            title="版本分布"
            extra={<Link to="/analytics">详情</Link>}
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            {versionDistribution.length === 0 ? (
              <Empty description="暂无版本数据" />
            ) : (
              <Space direction="vertical" size={isMobile ? 14 : 20} style={{ width: '100%' }}>
                {versionDistribution.map((item, index) => (
                  <div key={index}>
                    <Row justify="space-between" style={{ marginBottom: 8 }}>
                      <Text style={{ maxWidth: isMobile ? '48%' : '56%' }} ellipsis={{ tooltip: item.version }}>
                        {item.version}
                      </Text>
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
    </div>
  )
}

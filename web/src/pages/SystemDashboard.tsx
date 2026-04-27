import { useEffect, useState } from 'react'
import { Card, Col, DatePicker, Grid, Row, Select, Space, Statistic, Table, Tag, Typography, message } from 'antd'
import { AppstoreOutlined, BarChartOutlined, DeploymentUnitOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons'
import dayjs, { Dayjs } from 'dayjs'
import api from '../api/client'

const { Title, Text } = Typography
const { RangePicker } = DatePicker
const { Option } = Select

type OrgItem = {
  id: string
  name: string
}

type OverviewResponse = {
  orgs: { total: number; pending: number; active: number; disabled: number }
  users: { total: number; pending: number; active: number; disabled: number }
  apps: { total: number }
  devices: { total: number }
  events: { total: number }
  daily: { date: string; event_count: number }[]
}

export default function SystemDashboard() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [orgId, setOrgId] = useState<string>('')
  const [range, setRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(30, 'day'), dayjs()])
  const [overview, setOverview] = useState<OverviewResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const loadOrgs = async () => {
    try {
      const res = await api.get('/api/system/orgs')
      const items = (res.data.items || []).map((o: any) => ({
        id: o.id || o.ID,
        name: o.name || o.Name
      }))
      setOrgs(items)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载企业列表失败')
    }
  }

  const loadOverview = async () => {
    setLoading(true)
    try {
      const params: any = {
        from: range[0].format('YYYY-MM-DD'),
        to: range[1].format('YYYY-MM-DD')
      }
      if (orgId) {
        params.org_id = orgId
      }
      const res = await api.get('/api/system/overview', { params })
      setOverview(res.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载仪表盘失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
  }, [])

  useEffect(() => {
    loadOverview()
  }, [orgId, range[0].valueOf(), range[1].valueOf()])

  const orgStats = overview?.orgs || { total: 0, pending: 0, active: 0, disabled: 0 }
  const userStats = overview?.users || { total: 0, pending: 0, active: 0, disabled: 0 }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>系统仪表盘</Title>
        <Text type="secondary">全局运营与数据概览</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between' }}
          size={isMobile ? 10 : 8}
        >
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary">企业筛选：</Text>
            <Select
              value={orgId || 'all'}
              style={{ width: isMobile ? '100%' : 240 }}
              onChange={(value) => setOrgId(value === 'all' ? '' : value)}
            >
              <Option value="all">全部企业</Option>
              {orgs.map((org) => (
                <Option key={org.id} value={org.id}>{org.name}</Option>
              ))}
            </Select>
          </Space>
          <RangePicker
            style={isMobile ? { width: '100%' } : undefined}
            value={range}
            onChange={(values) => {
              if (values && values.length === 2) {
                setRange([values[0] as Dayjs, values[1] as Dayjs])
              }
            }}
          />
        </Space>
      </Card>

      <Row gutter={isMobile ? [12, 12] : [16, 16]}>
        <Col xs={24} sm={12} lg={4}>
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Statistic title="企业总数" value={orgStats.total} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Statistic title="用户总数" value={userStats.total} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Statistic title="应用总数" value={overview?.apps?.total || 0} prefix={<AppstoreOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Statistic title="设备总数" value={overview?.devices?.total || 0} prefix={<DeploymentUnitOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Statistic title="事件总数" value={overview?.events?.total || 0} prefix={<BarChartOutlined />} />
          </Card>
        </Col>
      </Row>

      <Row gutter={isMobile ? [12, 12] : [16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="企业状态" style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Space wrap>
              <Tag color="orange">待审核: {orgStats.pending}</Tag>
              <Tag color="green">已通过: {orgStats.active}</Tag>
              <Tag color="red">已禁用: {orgStats.disabled}</Tag>
            </Space>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="用户状态" style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Space wrap>
              <Tag color="orange">待激活: {userStats.pending}</Tag>
              <Tag color="green">启用: {userStats.active}</Tag>
              <Tag color="red">禁用: {userStats.disabled}</Tag>
            </Space>
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16, borderRadius: isMobile ? 10 : 12 }} title="事件趋势">
        <Table
          rowKey={(row) => row.date}
          dataSource={overview?.daily || []}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 420 } : undefined}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            { title: '日期', dataIndex: 'date' },
            { title: '事件数', dataIndex: 'event_count' }
          ]}
        />
      </Card>
    </div>
  )
}

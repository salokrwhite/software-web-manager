import {
  Alert,
  Button,
  Form,
  Input,
  Modal,
  Table,
  message,
  Card,
  Typography,
  Space,
  Tag,
  Avatar,
  Row,
  Col,
  Statistic,
  theme,
  Tooltip,
  Dropdown,
  Grid
} from 'antd'
import {
  PlusOutlined,
  AppstoreOutlined,
  MoreOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  RocketOutlined,
  PauseCircleOutlined,
  DatabaseOutlined
} from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import api, { getErrorMessage } from '../api/client'

const { Title, Text } = Typography

interface AppItem {
  id: string
  name: string
  slug: string
  description?: string
  version?: string
  created_at?: string
  status?: string
}

export default function Apps() {
  const [apps, setApps] = useState<AppItem[]>([])
  const [loading, setLoading] = useState(true)
  const [pageAnnouncementEnabled, setPageAnnouncementEnabled] = useState(false)
  const [pageAnnouncementContent, setPageAnnouncementContent] = useState('')
  const [open, setOpen] = useState(false)
  const [appCount, setAppCount] = useState(0)
  const [appLimit, setAppLimit] = useState(0)
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const { token } = theme.useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()
  const isPersonal = orgType === 'personal'

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/apps')
      const rawItems = res.data.items || []
      // 添加模拟数据用于展示
      const items = rawItems.map((app: any, index: number) => ({
        id: app.ID || app.id,
        name: app.Name || app.name,
        slug: app.Slug || app.slug,
        description: app.Description || app.description,
        status: (app.Status || app.status || 'active').toLowerCase(),
        version: app.Version || app.version,
        created_at: app.CreatedAt || app.created_at
      }))
      setApps(items)
      setAppCount(res.data.count ?? rawItems.length)
      setAppLimit(res.data.limit ?? 0)
    } finally {
      setLoading(false)
    }
  }

  const loadPageAnnouncement = async () => {
    try {
      const res = await api.get('/api/public/settings')
      const enabled =
        res?.data?.apps_page_announcement_enabled === true ||
        (res?.data?.apps_page_announcement_enabled == null && res?.data?.page_announcement_enabled === true)
      const content = String(
        res?.data?.apps_page_announcement_content ??
        res?.data?.page_announcement_content ??
        ''
      )
      setPageAnnouncementEnabled(enabled)
      setPageAnnouncementContent(content)
    } catch {
      setPageAnnouncementEnabled(false)
      setPageAnnouncementContent('')
    }
  }

  useEffect(() => {
    load()
    loadPageAnnouncement()
  }, [])

  const onCreate = async () => {
    try {
      const values = await form.validateFields()
      const res = await api.post('/api/apps', values)
      const status = (res.data?.app?.Status || res.data?.app?.status || '').toLowerCase()
      if (status === 'pending') {
        message.success('已提交审核')
      } else {
        message.success('创建成功')
      }
      setOpen(false)
      form.resetFields()
      load()
    } catch (err: any) {
      const code = err?.response?.data?.error
      if (code === 'personal_app_limit_reached') {
        const limit = err?.response?.data?.limit
        message.error(limit ? `个人用户最多可创建 ${limit} 个应用` : '已达到个人应用创建上限')
        return
      }
      message.error(getErrorMessage(err, '创建失败'))
    }
  }

  const handleDelete = async (record: AppItem) => {
    if (isPersonal && record.status && record.status !== 'active' && record.status !== 'rejected') {
      return
    }
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除应用 "${record.name}" 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await api.delete(`/api/apps/${record.id}`)
          message.success('删除成功')
          load() // 刷新列表
        } catch (err: any) {
          message.error(getErrorMessage(err, '删除失败'))
        }
      }
    })
  }

  const columns = [
    {
      title: '应用',
      dataIndex: 'name',
      key: 'name',
      render: (_: any, record: AppItem) => (
        <Space size={isMobile ? 10 : 16}>
          <Avatar
            size={isMobile ? 40 : 48}
            icon={<AppstoreOutlined />}
            style={{
              background: `${token.colorPrimary}15`,
              color: token.colorPrimary,
              fontSize: isMobile ? 20 : 24
            }}
          />
          <div>
            <Link to={`/apps/${record.id}/releases`}>
              <Text strong style={{ fontSize: isMobile ? 14 : 16 }}>{record.name}</Text>
            </Link>
            <div>
              <Text type="secondary" style={{ fontSize: isMobile ? 12 : 13 }}>{record.slug}</Text>
            </div>
          </div>
        </Space>
      )
    },
    {
      title: '当前版本',
      dataIndex: 'version',
      key: 'version',
      width: 120,
      render: (version?: string) => (
        version ? (
          <Tag icon={<RocketOutlined />} color="blue">
            {version}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        )
      )
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : status === 'pending' ? 'orange' : status === 'rejected' ? 'red' : 'default'}>
          {status === 'active' ? '已通过' : status === 'pending' ? '待审核' : status === 'rejected' ? '已驳回' : status}
        </Tag>
      )
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      responsive: ['md' as const],
      render: (time?: string) => (
        time ? <Text type="secondary">{new Date(time).toLocaleString()}</Text> : <Text type="secondary">-</Text>
      )
    },
    {
      title: '操作',
      key: 'action',
      width: isMobile ? 96 : 120,
      render: (_: any, record: AppItem) => {
        const locked = !!(isPersonal && record.status && record.status !== 'active' && record.status !== 'rejected')
        return (
          <Space size={isMobile ? 4 : 8} wrap={isMobile}>
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => navigate(`/apps/${record.id}/releases`)}
              title="查看"
            >
              {!isMobile && '查看'}
            </Button>
            <Tooltip title={locked ? '待审核，无法操作' : ''}>
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleDelete(record)}
                disabled={locked}
                title="删除"
              >
                {!isMobile && '删除'}
              </Button>
            </Tooltip>
          </Space>
        )
      }
    }
  ]

  return (
    <div>
      {pageAnnouncementEnabled && pageAnnouncementContent && (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message={<div style={{ whiteSpace: 'pre-wrap' }}>{pageAnnouncementContent}</div>}
        />
      )}
      {/* 页面标题区 */}
      <Row
        justify="space-between"
        align={isMobile ? 'top' : 'middle'}
        style={{ marginBottom: isMobile ? 16 : 24 }}
        gutter={isMobile ? [12, 12] : undefined}
      >
        <Col xs={24} lg={12}>
          <Space direction="vertical" size={4}>
            <Title level={4} style={{ margin: 0 }}>应用管理</Title>
            <Text type="secondary">管理您的所有应用和版本</Text>
          </Space>
        </Col>
        <Col xs={24} lg={12}>
          <Space
            size={isMobile ? 8 : 12}
            align={isMobile ? 'start' : 'center'}
            direction={isMobile ? 'vertical' : 'horizontal'}
            style={isMobile ? { width: '100%' } : { width: '100%', justifyContent: 'flex-end' }}
            wrap={!isMobile}
          >
            {appLimit > 0 && (
              <Text type="secondary">已创建 {appCount}/{appLimit}</Text>
            )}
            <Button
              type="primary"
              size={isMobile ? 'middle' : 'large'}
              icon={<PlusOutlined />}
              onClick={() => setOpen(true)}
              disabled={appLimit > 0 && appCount >= appLimit}
              style={isMobile ? { width: '100%' } : undefined}
            >
              新建应用
            </Button>
          </Space>
        </Col>
      </Row>

      {/* 统计卡片 */}
      <Row gutter={isMobile ? [12, 12] : [24, 24]} style={{ marginBottom: isMobile ? 16 : 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            <Statistic
              title="总应用数"
              value={apps.length}
              prefix={<AppstoreOutlined style={{ color: token.colorPrimary }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            <Statistic
              title="已通过"
              value={apps.filter(a => a.status === 'active').length}
              prefix={<RocketOutlined style={{ color: token.colorSuccess }} />}
              valueStyle={{ color: token.colorSuccess }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            <Statistic
              title="待审核"
              value={apps.filter(a => a.status === 'pending').length}
              prefix={<PauseCircleOutlined style={{ color: token.colorWarning }} />}
              valueStyle={{ color: token.colorWarning }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            styles={{ body: { padding: isMobile ? 16 : 24 } }}
          >
            <Statistic
              title="剩余可用"
              value={appLimit > 0 ? Math.max(0, appLimit - appCount) : '不限'}
              prefix={<DatabaseOutlined style={{ color: token.colorPrimary }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* 应用列表 */}
      <Card
        style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
        styles={{ body: { padding: 0 } }}
      >
        <Table
          rowKey="id"
          dataSource={apps}
          columns={columns}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 780 } : undefined}
          pagination={{
            pageSize: isMobile ? 6 : 10,
            showSizeChanger: !isMobile,
            showTotal: (total) => `共 ${total} 个应用`
          }}
        />
      </Card>

      {/* 新建应用弹窗 */}
      <Modal
        title="新建应用"
        open={open}
        onOk={onCreate}
        onCancel={() => {
          setOpen(false)
          form.resetFields()
        }}
        width={isMobile ? 'calc(100vw - 32px)' : 480}
        okText="创建"
        cancelText="取消"
      >
        <Form layout="vertical" form={form} style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="应用名称"
            rules={[{ required: true, message: '请输入应用名称' }]}
          >
            <Input placeholder="例如：MyApp" size={isMobile ? 'middle' : 'large'} />
          </Form.Item>
          <Form.Item
            name="slug"
            label="应用标识"
            rules={[
              { required: true, message: '请输入应用标识' },
              { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符' }
            ]}
          >
            <Input placeholder="例如：my-app" size={isMobile ? 'middle' : 'large'} />
          </Form.Item>
          <Form.Item name="description" label="应用描述">
            <Input.TextArea rows={3} placeholder="简要描述您的应用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

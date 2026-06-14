import { useEffect, useState } from 'react'
import { Button, Card, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select
const DEFAULT_PLAN_TYPES = ['free', 'team', 'enterprise']

const formatPlanLabel = (plan: string) => {
  const value = (plan || '').toLowerCase()
  if (value === 'team') return 'Team'
  if (value === 'enterprise') return 'Enterprise'
  return 'Free'
}

type SystemOrgItem = {
  id: string
  name: string
  plan: string
  status: string
  owner_email: string
  member_count: number
  app_count: number
  created_at: string
}

export default function SystemOrgs() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [items, setItems] = useState<SystemOrgItem[]>([])
  const [loading, setLoading] = useState(false)
  const [statusFilter, setStatusFilter] = useState<string>('active')
  const [idSearch, setIdSearch] = useState<string>('')
  const [emailSearch, setEmailSearch] = useState<string>('')
  const [createOpen, setCreateOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [planTypes, setPlanTypes] = useState<string[]>(DEFAULT_PLAN_TYPES)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    try {
      const params = statusFilter === 'all' ? {} : { status: statusFilter }
      const res = await api.get('/api/system/orgs', { params })
      setItems(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载组织失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [statusFilter])

  useEffect(() => {
    loadPlanTypes()
  }, [])

  const loadPlanTypes = async () => {
    try {
      const res = await api.get('/api/system/settings')
      const items = Array.isArray(res?.data?.org_plan_types) ? res.data.org_plan_types : []
      const normalized = items
        .map((item: string) => (item || '').toLowerCase().trim())
        .filter((item: string) => item === 'free' || item === 'team' || item === 'enterprise')
      setPlanTypes(normalized.length > 0 ? normalized : DEFAULT_PLAN_TYPES)
    } catch {
      setPlanTypes(DEFAULT_PLAN_TYPES)
    }
  }

  const approveOrg = async (orgId: string) => {
    try {
      await api.post(`/api/system/orgs/${orgId}/approve`)
      message.success('已启用组织')
      load()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '操作失败')
    }
  }

  const disableOrg = async (orgId: string) => {
    try {
      await api.post(`/api/system/orgs/${orgId}/disable`)
      message.success('已禁用组织')
      load()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '操作失败')
    }
  }

  const onCreate = async () => {
    try {
      const values = await form.validateFields()
      setCreating(true)
      await api.post('/api/system/orgs', values)
      message.success('企业已创建')
      setCreateOpen(false)
      form.resetFields()
      load()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    } finally {
      setCreating(false)
    }
  }

  const statusTag = (status: string) => {
    const s = (status || '').toLowerCase()
    if (s === 'active') return <Tag color="green">已通过</Tag>
    if (s === 'pending') return <Tag color="orange">待审核</Tag>
    if (s === 'rejected') return <Tag color="red">已驳回</Tag>
    if (s === 'disabled') return <Tag color="red">已禁用</Tag>
    return <Tag>{status}</Tag>
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中组织？',
      content: '删除后将硬删除该组织下所有应用、安装包、日志、工单及附件，且不可恢复；仅删除企业管理员账号，子用户账号数据会保留。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/orgs/batch-delete', { ids: selectedRowKeys })
          message.success('删除成功')
          setSelectedRowKeys([])
          load()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>系统组织管理</Title>
        <Text type="secondary">审核与禁用企业组织</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between' }}
        >
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary">状态筛选：</Text>
            <Select value={statusFilter} style={{ width: isMobile ? '100%' : 160 }} onChange={setStatusFilter}>
              <Option value="active">已通过</Option>
              <Option value="disabled">已禁用</Option>
            </Select>
            <Input.Search
              placeholder="按企业 ID 搜索"
              allowClear
              style={{ width: isMobile ? '100%' : 220 }}
              value={idSearch}
              onChange={e => setIdSearch(e.target.value)}
              onSearch={v => setIdSearch(v)}
            />
            <Input.Search
              placeholder="按管理员邮箱搜索"
              allowClear
              style={{ width: isMobile ? '100%' : 220 }}
              value={emailSearch}
              onChange={e => setEmailSearch(e.target.value)}
              onSearch={v => setEmailSearch(v)}
            />
          </Space>
          <Button type="primary" onClick={() => setCreateOpen(true)} style={isMobile ? { width: '100%' } : undefined}>创建企业</Button>
        </Space>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <div
          style={{
            display: 'flex',
            flexDirection: isMobile ? 'column' : 'row',
            alignItems: isMobile ? 'stretch' : 'center',
            justifyContent: 'space-between',
            gap: 10,
            marginBottom: 12
          }}
        >
          <Text type="secondary">已选 {selectedRowKeys.length} 条</Text>
          <Button danger disabled={selectedRowKeys.length === 0 || loading} onClick={handleBatchDelete} style={isMobile ? { width: '100%' } : undefined}>
            批量删除
          </Button>
        </div>
        <Table
          rowKey={(row) => row.id}
          dataSource={items.filter(it => {
            const idOk = !idSearch.trim() || it.id.toLowerCase().includes(idSearch.trim().toLowerCase())
            const emailOk = !emailSearch.trim() || (it.owner_email || '').toLowerCase().includes(emailSearch.trim().toLowerCase())
            return idOk && emailOk
          })}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1260 } : { x: 1320 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as string[])
          }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            { title: '名称', dataIndex: 'name', width: 180 },
            { title: '状态', dataIndex: 'status', width: 110, render: (v: string) => statusTag(v) },
            { title: '计划', dataIndex: 'plan', width: 100 },
            { title: '管理员邮箱', dataIndex: 'owner_email', width: 220 },
            { title: '成员数', dataIndex: 'member_count', width: 90 },
            { title: '应用数', dataIndex: 'app_count', width: 90 },
            { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
            {
              title: '操作',
              width: 220,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: SystemOrgItem) => (
                <Space size={[6, 6]} wrap>
                  {row.status === 'pending' && (
                    <Button size="small" type="primary" onClick={() => approveOrg(row.id)}>{isMobile ? '通过' : '审批'}</Button>
                  )}
                  {row.status === 'disabled' && (
                    <Button size="small" onClick={() => approveOrg(row.id)}>启用</Button>
                  )}
                  {row.status === 'active' && (
                    <Button size="small" danger onClick={() => disableOrg(row.id)}>禁用</Button>
                  )}

                </Space>
              )
            }
          ]}
        />
      </Card>

      <Modal
        open={createOpen}
        title="创建企业组织"
        onOk={onCreate}
        confirmLoading={creating}
        onCancel={() => { setCreateOpen(false); form.resetFields() }}
        okText="创建"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={form}>
          <Form.Item name="org_name" label="企业名称" rules={[{ required: true, message: '请输入企业名称' }]}>
            <Input placeholder="企业名称" />
          </Form.Item>
          <Form.Item name="owner_email" label="企业管理员邮箱" rules={[{ required: true, message: '请输入管理员邮箱' }, { type: 'email', message: '邮箱格式错误' }]}>
            <Input placeholder="admin@company.com" />
          </Form.Item>
          <Form.Item name="password" label="管理员初始密码" rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '至少 6 位' }]}>
            <Input.Password placeholder="初始密码" />
          </Form.Item>
          <Form.Item name="plan" label="计划(可选)">
            <Select allowClear placeholder="不选择则默认 Free">
              {planTypes.map((plan) => (
                <Option key={plan} value={plan}>
                  {formatPlanLabel(plan)}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

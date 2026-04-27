import { Button, Card, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'
import { formatTicketStatus } from '../utils/ticket'

const { Title, Text } = Typography
const { Option } = Select

type OrgItem = {
  id: string
  name: string
}

type TicketItem = {
  id: string
  org_id: string
  org_name: string
  title: string
  status: string
  assignee_type: string
  assignee_user_id?: string | null
  assignee_email?: string
  created_by_email?: string
  created_at: string
  attachment_count: number
}

const formatDateTime = (value?: string | null) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const formatAssignee = (ticket: { assignee_type?: string; assignee_email?: string; assignee_user_id?: string | null }) => {
  if ((ticket.assignee_type || '').toLowerCase() === 'system') return '系统管理员'
  return ticket.assignee_email || ticket.assignee_user_id || '-'
}

export default function SystemTickets() {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [items, setItems] = useState<TicketItem[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [pageSize, setPageSize] = useState(10)
  const [current, setCurrent] = useState(1)
  const [queryParams, setQueryParams] = useState<any>({})
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [form] = Form.useForm()

  const loadOrgs = async () => {
    try {
      const res = await api.get('/api/system/orgs')
      const list = (res.data.items || []).map((o: any) => ({
        id: o.id || o.ID,
        name: o.name || o.Name
      }))
      setOrgs(list)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载企业列表失败')
    }
  }

  const loadTickets = async (params?: any, page = current, size = pageSize) => {
    setLoading(true)
    try {
      const limit = size
      const offset = (page - 1) * size
      const res = await api.get('/api/system/tickets', { params: { ...params, limit, offset } })
      setItems(res.data.items || [])
      setTotal(res.data.total || 0)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载工单失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
    loadTickets()
  }, [])

  const onSearch = async () => {
    const values = await form.validateFields()
    const params: any = {
      org_id: values.org_id === 'all' ? undefined : values.org_id,
      status: values.status === 'all' ? undefined : values.status,
      q: values.q
    }
    setQueryParams(params)
    setCurrent(1)
    loadTickets(params, 1, pageSize)
  }

  const handleTableChange = (page: number, size?: number) => {
    const nextSize = size || pageSize
    setCurrent(page)
    setPageSize(nextSize)
    loadTickets(queryParams, page, nextSize)
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中工单？',
      content: '删除后将清理工单、附件及聊天记录，且不可恢复。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        await api.post('/api/system/tickets/batch-delete', { ids: selectedRowKeys })
        message.success('删除成功')
        setSelectedRowKeys([])
        loadTickets(queryParams, current, pageSize)
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>工单详情</Title>
        <Text type="secondary">查看全量工单并更新处理状态</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <div
          style={{
            display: 'flex',
            flexDirection: isMobile ? 'column' : 'row',
            alignItems: isMobile ? 'stretch' : 'center',
            justifyContent: 'space-between',
            gap: 12
          }}
        >
          <Form layout={isMobile ? 'vertical' : 'inline'} form={form} initialValues={{ status: 'all', org_id: 'all' }} style={isMobile ? { width: '100%' } : undefined}>
            <Form.Item name="org_id">
              <Select style={{ width: isMobile ? '100%' : 220 }}>
                <Option value="all">全部企业</Option>
                {orgs.map((org) => (
                  <Option key={org.id} value={org.id}>{org.name}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status">
              <Select style={{ width: isMobile ? '100%' : 160 }}>
                <Option value="all">全部状态</Option>
                <Option value="submitted">已提交</Option>
                <Option value="in_progress">处理中</Option>
                <Option value="resolved">已完成</Option>
              </Select>
            </Form.Item>
            <Form.Item name="q">
              <Input placeholder="关键词" style={{ width: isMobile ? '100%' : 200 }} allowClear />
            </Form.Item>
            <Form.Item>
              <Button type="primary" onClick={onSearch} style={isMobile ? { width: '100%' } : undefined}>查询</Button>
            </Form.Item>
          </Form>
          <Button danger disabled={selectedRowKeys.length === 0} onClick={handleBatchDelete} style={isMobile ? { width: '100%' } : undefined}>
            批量删除
          </Button>
        </div>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Table
          rowKey="id"
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as string[])
          }}
          dataSource={items}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1130 } : { x: 1240 }}
          pagination={{
            current,
            pageSize,
            total,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50'],
            onChange: handleTableChange,
            onShowSizeChange: handleTableChange
          }}
          columns={[
            { title: '标题', dataIndex: 'title', width: 260 },
            { title: '企业', dataIndex: 'org_name', width: 180, render: (v: string) => v || '-' },
            {
              title: '状态',
              dataIndex: 'status',
              width: 120,
              render: (v: string) => <Tag>{formatTicketStatus(v)}</Tag>
            },
            {
              title: '派发对象',
              width: 150,
              render: (_: any, row: TicketItem) => formatAssignee(row)
            },
            {
              title: '创建人',
              dataIndex: 'created_by_email',
              width: 200,
              render: (v: string) => v || '-'
            },
            {
              title: '创建时间',
              dataIndex: 'created_at',
              width: 180,
              render: (v: string) => formatDateTime(v)
            },
            {
              title: '附件数',
              dataIndex: 'attachment_count',
              width: 90
            },
            {
              title: '操作',
              width: 90,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: TicketItem) => (
                <Button size="small" onClick={() => navigate(`/system/tickets/${row.id}`)}>{isMobile ? '详情' : '查看'}</Button>
              )
            }
          ]}
        />
      </Card>
    </div>
  )
}

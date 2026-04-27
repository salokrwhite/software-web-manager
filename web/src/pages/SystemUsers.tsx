import { Button, Card, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

type OrgItem = {
  id: string
  name: string
}

type SystemUserItem = {
  id: string
  email: string
  status: string
  system_role: string
  org_name: string
  org_role: string
  org_count: number
  created_at: string
}

export default function SystemUsers() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [users, setUsers] = useState<SystemUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [pageSize, setPageSize] = useState(10)
  const [current, setCurrent] = useState(1)
  const [queryParams, setQueryParams] = useState<any>({})
  const [currentUserId, setCurrentUserId] = useState<string>('')
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [editOpen, setEditOpen] = useState(false)
  const [editUser, setEditUser] = useState<SystemUserItem | null>(null)
  const [editLoading, setEditLoading] = useState(false)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()

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

  const loadProfile = async () => {
    try {
      const res = await api.get('/api/system/profile')
      setCurrentUserId(res.data?.id || '')
    } catch {
      setCurrentUserId('')
    }
  }

  const loadUsers = async (params?: any, page = current, size = pageSize) => {
    setLoading(true)
    try {
      const limit = size
      const offset = (page - 1) * size
      const res = await api.get('/api/system/users', { params: { ...params, limit, offset } })
      setUsers(res.data.items || [])
      setTotal(res.data.total || 0)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载用户失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
    loadProfile()
    loadUsers()
  }, [])

  const onSearch = async () => {
    const values = await form.validateFields()
    const params: any = {
      q: values.q,
      status: values.status === 'all' ? undefined : values.status,
      org_id: values.org_id === 'all' ? undefined : values.org_id,
      role: values.role === 'all' ? undefined : values.role,
      system_role: values.system_role === 'all' ? undefined : values.system_role
    }
    setQueryParams(params)
    setCurrent(1)
    loadUsers(params, 1, pageSize)
  }

  const handleTableChange = (page: number, size?: number) => {
    const nextSize = size || pageSize
    setCurrent(page)
    setPageSize(nextSize)
    loadUsers(queryParams, page, nextSize)
  }

  const toggleUserStatus = async (user: SystemUserItem) => {
    const isDisabled = user.status === 'disabled'
    const action = isDisabled ? '启用' : '禁用'
    Modal.confirm({
      title: `确认${action}该账号？`,
      content: `${user.email}`,
      okType: isDisabled ? 'primary' : 'danger',
      onOk: async () => {
        try {
          const url = isDisabled ? `/api/system/users/${user.id}/enable` : `/api/system/users/${user.id}/disable`
          await api.post(url)
          message.success(`${action}成功`)
          loadUsers(queryParams, current, pageSize)
        } catch (err: any) {
          message.error(err?.response?.data?.error || `${action}失败`)
        }
      }
    })
  }

  const openEditUser = (user: SystemUserItem) => {
    setEditUser(user)
    setEditOpen(true)
    editForm.setFieldsValue({
      email: user.email,
      status: user.status || 'active',
      system_role: (user.system_role || 'none').toLowerCase(),
      password: '',
      confirm_password: ''
    })
  }

  const handleEditUser = async () => {
    if (!editUser) return
    const values = await editForm.validateFields()
    if (values.password && values.password !== values.confirm_password) {
      message.error('两次输入的密码不一致')
      return
    }
    const payload: any = {
      email: values.email,
      status: values.status,
      system_role: values.system_role
    }
    if (values.password) {
      payload.password = values.password
    }
    setEditLoading(true)
    try {
      await api.patch(`/api/system/users/${editUser.id}`, payload)
      message.success('更新成功')
      setEditOpen(false)
      setEditUser(null)
      editForm.resetFields()
      loadUsers(queryParams, current, pageSize)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
    } finally {
      setEditLoading(false)
    }
  }

  const renderStatus = (status: string) => {
    const key = (status || '').toLowerCase()
    if (key === 'active') return <Tag color="green">启用</Tag>
    if (key === 'pending') return <Tag color="orange">待激活</Tag>
    if (key === 'disabled') return <Tag color="red">禁用</Tag>
    return <Tag>{status}</Tag>
  }

  const renderSystemRole = (role: string) => {
    const key = (role || '').toLowerCase()
    if (key === 'system_admin') return <Tag color="blue">系统管理员</Tag>
    if (key === 'org_admin') return <Tag color="geekblue">企业管理员</Tag>
    if (!key) return '-'
    if (key === 'none') return <Tag>普通用户</Tag>
    return <Tag>{key}</Tag>
  }

  const editIsSystemAdmin = (editUser?.system_role || '').toLowerCase() === 'system_admin'
  const editIsCurrentUser = !!currentUserId && editUser?.id === currentUserId

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中用户？',
      content: '删除后将清理用户关联数据，且不可恢复。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/users/batch-delete', { ids: selectedRowKeys })
          message.success('删除成功')
          setSelectedRowKeys([])
          loadUsers(queryParams, current, pageSize)
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>用户管理</Title>
        <Text type="secondary">管理系统内全部账号</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Form
          layout={isMobile ? 'vertical' : 'inline'}
          form={form}
          initialValues={{ status: 'all', org_id: 'all', system_role: 'all', role: 'all' }}
          style={isMobile ? { width: '100%' } : undefined}
        >
          <Form.Item name="q">
            <Input placeholder="邮箱关键字" style={{ width: isMobile ? '100%' : 220 }} allowClear />
          </Form.Item>
          <Form.Item name="status">
            <Select style={{ width: isMobile ? '100%' : 140 }}>
              <Option value="all">全部状态</Option>
              <Option value="active">启用</Option>
              <Option value="pending">待激活</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="org_id">
            <Select style={{ width: isMobile ? '100%' : 200 }}>
              <Option value="all">全部企业</Option>
              {orgs.map((org) => (
                <Option key={org.id} value={org.id}>{org.name}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="role">
            <Select style={{ width: isMobile ? '100%' : 140 }}>
              <Option value="all">全部角色</Option>
              <Option value="owner">owner</Option>
              <Option value="admin">admin</Option>
              <Option value="dev">dev</Option>
              <Option value="viewer">viewer</Option>
            </Select>
          </Form.Item>
          <Form.Item name="system_role">
            <Select style={{ width: isMobile ? '100%' : 160 }}>
              <Option value="all">系统角色</Option>
              <Option value="system_admin">系统管理员</Option>
              <Option value="org_admin">企业管理员</Option>
              <Option value="none">普通用户</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={onSearch} style={isMobile ? { width: '100%' } : undefined}>查询</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <div
          style={{
            display: 'flex',
            flexDirection: isMobile ? 'column' : 'row',
            alignItems: isMobile ? 'stretch' : 'center',
            justifyContent: 'space-between',
            marginBottom: 12,
            gap: 10
          }}
        >
          <Text type="secondary">已选 {selectedRowKeys.length} 条</Text>
          <Button danger disabled={selectedRowKeys.length === 0 || loading} onClick={handleBatchDelete} style={isMobile ? { width: '100%' } : undefined}>
            批量删除
          </Button>
        </div>
        <Table
          rowKey="id"
          dataSource={users}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1380 } : { x: 1460 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as string[]),
            getCheckboxProps: (record: SystemUserItem) => {
              const isSystemAdmin = (record.system_role || '').toLowerCase() === 'system_admin'
              const isCurrentUser = !!currentUserId && record.id === currentUserId
              return { disabled: isSystemAdmin || isCurrentUser }
            }
          }}
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
            { title: '邮箱', dataIndex: 'email', width: 220 },
            { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => renderStatus(v) },
            { title: '系统角色', dataIndex: 'system_role', width: 130, render: (v: string) => renderSystemRole(v) },
            { title: '企业', dataIndex: 'org_name', width: 200, render: (v: string) => v || '未加入' },
            { title: '企业角色', dataIndex: 'org_role', width: 120, render: (v: string) => v || '-' },
            { title: '企业数量', dataIndex: 'org_count', width: 90 },
            { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
            {
              title: '操作',
              width: 160,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: SystemUserItem) => {
                const isSystemAdmin = (row.system_role || '').toLowerCase() === 'system_admin'
                const isCurrentUser = !!currentUserId && row.id === currentUserId
                const disableDisabled = isSystemAdmin || isCurrentUser
                const actionLabel = row.status === 'disabled' ? '启用' : '禁用'
                return (
                  <Space size={[6, 6]} wrap>
                    <Button
                      size="small"
                      danger={row.status !== 'disabled'}
                      disabled={row.status !== 'disabled' && disableDisabled}
                      onClick={() => toggleUserStatus(row)}
                    >
                      {actionLabel}
                    </Button>
                    <Button size="small" onClick={() => openEditUser(row)}>编辑</Button>
                  </Space>
                )
              }
            }
          ]}
        />
      </Card>

      <Modal
        open={editOpen}
        title="编辑用户"
        onCancel={() => { setEditOpen(false); setEditUser(null); editForm.resetFields() }}
        onOk={handleEditUser}
        confirmLoading={editLoading}
        okText="保存"
        cancelText="取消"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={editForm}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式错误' }]}
          >
            <Input placeholder="邮箱" />
          </Form.Item>
          <Form.Item
            name="password"
            label="新密码(可选)"
            rules={[
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  if (value.length < 6) return Promise.reject(new Error('密码至少 6 位'))
                  return Promise.resolve()
                }
              }
            ]}
          >
            <Input.Password placeholder="不修改请留空" />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label="确认新密码"
            dependencies={['password']}
            rules={[
              ({ getFieldValue }) => ({
                validator: (_, value) => {
                  const password = getFieldValue('password')
                  if (!password && !value) return Promise.resolve()
                  if (password && value !== password) return Promise.reject(new Error('两次输入的密码不一致'))
                  return Promise.resolve()
                }
              })
            ]}
          >
            <Input.Password placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
            <Select disabled={editIsSystemAdmin}>
              <Option value="active">启用</Option>
              <Option value="pending">待激活</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="system_role" label="系统角色" rules={[{ required: true, message: '请选择系统角色' }]}>
            <Select disabled={editIsSystemAdmin || editIsCurrentUser}>
              <Option value="system_admin">系统管理员</Option>
              <Option value="org_admin">企业管理员</Option>
              <Option value="none">普通用户</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

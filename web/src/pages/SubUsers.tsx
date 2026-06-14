import { Button, Card, Form, Grid, Input, Modal, Select, Space, Switch, Table, Typography, message, Tag } from 'antd'
import { useEffect, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select
const defaultRoles = ['admin', 'dev', 'viewer']

export default function SubUsers() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const orgId = sessionStorage.getItem('org_id') || ''
  const [orgName, setOrgName] = useState('')
  const [members, setMembers] = useState<any[]>([])
  const [loadingMembers, setLoadingMembers] = useState(false)
  const [creatingUser, setCreatingUser] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editingMember, setEditingMember] = useState<any | null>(null)
  const [savingEdit, setSavingEdit] = useState(false)
  const [roles, setRoles] = useState<string[]>(defaultRoles)
  const [memberForm] = Form.useForm()
  const [editForm] = Form.useForm()

  const loadOrgs = async () => {
    try {
      const res = await api.get('/api/orgs')
      const items = res.data.items || []
      const current = items.find((o: any) => (o.id || o.ID) === orgId)
      setOrgName(current?.name || current?.Name || orgId)
    } catch {
      setOrgName(orgId)
    }
  }

  const loadMembers = async () => {
    if (!orgId) return
    setLoadingMembers(true)
    try {
      const res = await api.get(`/api/orgs/${orgId}/members`)
      const items = res.data.items || []
      setMembers(items)
      const usedRoles = items
        .map((it: any) => String(it.Role || it.role || '').toLowerCase().trim())
        .filter((v: string) => v !== '' && v !== 'owner')
      setRoles((prev) => Array.from(new Set([...prev, ...usedRoles])))
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载成员失败')
    } finally {
      setLoadingMembers(false)
    }
  }

  const loadRoles = async () => {
    if (!orgId) return
    try {
      const res = await api.get(`/api/orgs/${orgId}/roles`)
      const custom = (res.data?.items || [])
        .filter((it: any) => String(it.status || it.Status || 'active').toLowerCase() === 'active')
        .map((it: any) => String(it.role_name || it.RoleName || '').toLowerCase().trim())
        .filter((v: string) => v !== '' && v !== 'owner')
      setRoles(Array.from(new Set([...defaultRoles, ...custom])))
    } catch {
      setRoles(defaultRoles)
    }
  }

  useEffect(() => {
    loadOrgs()
    loadRoles()
    loadMembers()
  }, [orgId])

  const createUser = async () => {
    if (!orgId) return
    try {
      const values = await memberForm.validateFields()
      setCreatingUser(true)
      await api.post(`/api/orgs/${orgId}/users`, values)
      message.success('创建成功')
      memberForm.resetFields()
      setCreateOpen(false)
      loadMembers()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    } finally {
      setCreatingUser(false)
    }
  }

  const updateMember = async (userId: string, role: string, enabled: boolean) => {
    if (!orgId) return
    try {
      await api.patch(`/api/orgs/${orgId}/members/${userId}`, { role, status: enabled ? 'active' : 'disabled' })
      message.success('成员已更新')
      loadMembers()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
    }
  }

  const openEdit = (row: any) => {
    const role = String(row.Role || row.role || 'viewer').toLowerCase()
    const status = String(row.Status || row.status || 'active').toLowerCase()
    setEditingMember(row)
    editForm.setFieldsValue({
      role,
      enabled: status !== 'disabled'
    })
  }

  const submitEdit = async () => {
    if (!editingMember) return
    try {
      const values = await editForm.validateFields()
      setSavingEdit(true)
      await updateMember(
        String(editingMember.UserID || editingMember.user_id || ''),
        String(values.role || 'viewer').toLowerCase(),
        values.enabled === true
      )
      setEditingMember(null)
      editForm.resetFields()
    } finally {
      setSavingEdit(false)
    }
  }

  const removeMember = async (userId: string) => {
    if (!orgId) return
    try {
      await api.delete(`/api/orgs/${orgId}/members/${userId}`)
      message.success('成员已移除')
      loadMembers()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '移除失败')
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>子用户管理</Title>
        <Text type="secondary">支持后台创建账号与成员管理</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Text type="secondary">当前组织：</Text>
        <Text style={{ marginLeft: 8 }}>{orgName || orgId || '-'}</Text>
      </Card>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}
        >
          <Title level={5} style={{ margin: 0 }}>成员列表</Title>
          <Button type="primary" onClick={() => setCreateOpen(true)} style={isMobile ? { width: '100%' } : undefined}>添加成员</Button>
        </Space>
        <Table
          rowKey={(row) => row.UserID || row.user_id}
          dataSource={members}
          loading={loadingMembers}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 980 } : { x: 1080 }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            { title: '邮箱', dataIndex: 'Email', width: 220, render: (_: any, row: any) => row.Email || row.email || '-' },
            { title: '用户ID', dataIndex: 'UserID', width: 220, render: (_: any, row: any) => row.UserID || row.user_id },
            {
              title: '角色',
              dataIndex: 'Role',
              width: 120,
              render: (_: any, row: any) => {
                const currentRole = (row.Role || row.role || '').toLowerCase()
                if (currentRole === 'owner') return <Tag color="gold">所有者</Tag>
                if (currentRole === 'admin') return <Tag color="blue">管理员</Tag>
                if (currentRole === 'dev') return <Tag color="purple">开发者</Tag>
                if (currentRole === 'viewer') return <Tag>观察者</Tag>
                return <Tag color="cyan">{currentRole || '-'}</Tag>
              }
            },
            {
              title: '状态',
              dataIndex: 'Status',
              width: 100,
              render: (_: any, row: any) => {
                const status = String(row.Status || row.status || 'active').toLowerCase()
                return status === 'disabled' ? <Tag color="red">禁用</Tag> : <Tag color="green">启用</Tag>
              }
            },
            {
              title: '加入时间',
              dataIndex: 'CreatedAt',
              width: 180,
              render: (_: any, row: any) => {
                const v = row.CreatedAt || row.created_at
                return v ? new Date(v).toLocaleString() : '-'
              }
            },
            {
              title: '操作',
              width: 150,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: any) => {
                const currentRole = (row.Role || row.role || '').toLowerCase()
                return (
                  <Space size={[6, 6]} wrap>
                    <Button
                      size="small"
                      onClick={() => openEdit(row)}
                      disabled={currentRole === 'owner'}
                    >
                      编辑
                    </Button>
                    <Button
                      danger
                      size="small"
                      disabled={currentRole === 'owner'}
                      onClick={() => removeMember(row.UserID || row.user_id)}
                    >
                      移除
                    </Button>
                  </Space>
                )
              }
            }
          ]}
        />
      </Card>

      <Modal
        open={createOpen}
        title="添加子用户"
        onCancel={() => setCreateOpen(false)}
        onOk={createUser}
        okText="确认添加"
        cancelText="取消"
        confirmLoading={creatingUser}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={memberForm}>
          <Form.Item
            name="email"
            label="成员邮箱"
            rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式错误' }]}
          >
            <Input placeholder="成员邮箱" />
          </Form.Item>
          <Form.Item
            name="password"
            label="初始密码"
            rules={[{ required: true, message: '请输入初始密码' }, { min: 6, message: '至少 6 位' }]}
          >
            <Input.Password placeholder="初始密码" />
          </Form.Item>
          <Form.Item
            name="role"
            label="角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Select placeholder="角色">
              {roles.map((r) => (
                <Option key={r} value={r}>{r}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!editingMember}
        title="编辑子用户"
        onCancel={() => {
          setEditingMember(null)
          editForm.resetFields()
        }}
        onOk={submitEdit}
        okText="保存"
        cancelText="取消"
        confirmLoading={savingEdit}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={editForm}>
          <Form.Item
            name="role"
            label="角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Select placeholder="角色">
              {roles.map((r) => (
                <Option key={r} value={r}>{r}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="enabled"
            label="是否启用"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

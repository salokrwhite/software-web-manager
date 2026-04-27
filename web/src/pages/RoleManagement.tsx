import { Button, Card, Checkbox, Drawer, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import api, { getErrorMessage } from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

type OrgRole = {
  id: string
  role_name: string
  description?: string
  status: 'active' | 'disabled'
  is_builtin?: boolean
  created_at?: string
}

type PermissionItem = {
  permission_code: string
  module: string
  name: string
  description?: string
}

export default function RoleManagement() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const orgId = sessionStorage.getItem('org_id') || ''
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<OrgRole[]>([])
  const [open, setOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState<OrgRole | null>(null)
  const [form] = Form.useForm()

  const [permOpen, setPermOpen] = useState(false)
  const [permSaving, setPermSaving] = useState(false)
  const [permItems, setPermItems] = useState<PermissionItem[]>([])
  const [permChecked, setPermChecked] = useState<string[]>([])
  const [permRoleName, setPermRoleName] = useState('')

  const permByModule = useMemo(() => {
    const map: Record<string, PermissionItem[]> = {}
    permItems.forEach((it) => {
      const key = it.module || 'default'
      if (!map[key]) map[key] = []
      map[key].push(it)
    })
    return map
  }, [permItems])

  const loadData = async () => {
    if (!orgId) return
    setLoading(true)
    try {
      const res = await api.get(`/api/orgs/${orgId}/roles`)
      const list = (res.data?.items || []) as any[]
      setItems(
        list.map((it) => ({
          id: it.id || it.ID,
          role_name: it.role_name || it.RoleName,
          description: it.description || it.Description || '',
          status: (it.status || it.Status || 'active').toLowerCase(),
          is_builtin: Boolean(it.is_builtin ?? it.IsBuiltin),
          created_at: it.created_at || it.CreatedAt
        }))
      )
    } catch (err: any) {
      message.error(getErrorMessage(err, '加载角色失败'))
    } finally {
      setLoading(false)
    }
  }

  const loadPermissions = async () => {
    if (!orgId) return
    try {
      const res = await api.get(`/api/orgs/${orgId}/permissions`)
      const list = (res.data?.items || []) as PermissionItem[]
      setPermItems(list)
    } catch (err: any) {
      message.error(getErrorMessage(err, '加载权限点失败'))
    }
  }

  useEffect(() => {
    loadData()
    loadPermissions()
  }, [orgId])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ status: 'active' })
    setOpen(true)
  }

  const openEdit = (row: OrgRole) => {
    setEditing(row)
    form.setFieldsValue({
      role_name: row.role_name,
      description: row.description || '',
      status: row.status || 'active'
    })
    setOpen(true)
  }

  const openPermissionConfig = async (row: OrgRole) => {
    if (!orgId) return
    setPermRoleName(row.role_name)
    setPermOpen(true)
    setPermChecked([])
    try {
      const res = await api.get(`/api/orgs/${orgId}/roles/${encodeURIComponent(row.role_name)}/permissions`)
      setPermChecked((res.data?.permission_codes || []) as string[])
    } catch (err: any) {
      message.error(getErrorMessage(err, '加载角色权限失败'))
    }
  }

  const submitPermissionConfig = async () => {
    if (!orgId || !permRoleName) return
    setPermSaving(true)
    try {
      await api.put(`/api/orgs/${orgId}/roles/${encodeURIComponent(permRoleName)}/permissions`, {
        permission_codes: permChecked
      })
      message.success('权限配置已保存')
      setPermOpen(false)
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存权限失败'))
    } finally {
      setPermSaving(false)
    }
  }

  const handleModulePermissionChange = (moduleCodes: string[], nextValues: string[]) => {
    setPermChecked((prev) => {
      const moduleSet = new Set(moduleCodes)
      const preserved = prev.filter((code) => !moduleSet.has(code))
      return [...preserved, ...nextValues]
    })
  }

  const submit = async () => {
    if (!orgId) return
    const values = await form.validateFields()
    setSaving(true)
    try {
      if (editing) {
        await api.patch(`/api/orgs/${orgId}/roles/${encodeURIComponent(editing.role_name)}`, {
          description: values.description,
          status: values.status
        })
        message.success('角色已更新')
      } else {
        await api.post(`/api/orgs/${orgId}/roles`, {
          role_name: values.role_name,
          description: values.description
        })
        message.success('角色已创建')
      }
      setOpen(false)
      form.resetFields()
      await loadData()
    } catch (err: any) {
      message.error(getErrorMessage(err, '保存失败'))
    } finally {
      setSaving(false)
    }
  }

  const remove = (row: OrgRole) => {
    if (!orgId) return
    Modal.confirm({
      title: `删除角色 ${row.role_name}？`,
      content: '仅支持删除未被成员使用的角色。',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await api.delete(`/api/orgs/${orgId}/roles/${encodeURIComponent(row.role_name)}`)
          message.success('角色已删除')
          await loadData()
        } catch (err: any) {
          message.error(getErrorMessage(err, '删除失败'))
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>角色管理</Title>
        <Text type="secondary">权限类型与权限点配置按组织隔离，互不影响。</Text>
      </Space>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}
        >
          <Text type="secondary">当前组织：{orgId || '-'}</Text>
          <Button type="primary" onClick={openCreate} style={isMobile ? { width: '100%' } : undefined}>新建权限类型</Button>
        </Space>
        <Table
          rowKey={(row) => row.id}
          loading={loading}
          dataSource={items}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 920 } : { x: 980 }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            { title: '权限类型', dataIndex: 'role_name', width: 160, render: (v: string) => <Tag color="blue">{v}</Tag> },
            {
              title: '来源',
              dataIndex: 'is_builtin',
              width: 120,
              render: (v: boolean) => (v ? <Tag color="gold">系统内置</Tag> : <Tag>自定义</Tag>)
            },
            { title: '描述', dataIndex: 'description', width: 260, render: (v: string) => v || '-' },
            {
              title: '状态',
              dataIndex: 'status',
              width: 110,
              render: (v: string) => (String(v).toLowerCase() === 'disabled' ? <Tag color="red">禁用</Tag> : <Tag color="green">启用</Tag>)
            },
            {
              title: '操作',
              width: 210,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: OrgRole) => (
                <Space size={[6, 6]} wrap>
                  <Button size="small" onClick={() => openPermissionConfig(row)}>{isMobile ? '配置' : '权限配置'}</Button>
                  <Button size="small" onClick={() => openEdit(row)}>编辑</Button>
                  <Button danger size="small" onClick={() => remove(row)} disabled={row.is_builtin === true}>删除</Button>
                </Space>
              )
            }
          ]}
        />
      </Card>

      <Modal
        open={open}
        title={editing ? '编辑权限类型' : '新建权限类型'}
        onCancel={() => setOpen(false)}
        onOk={submit}
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="role_name"
            label="权限类型"
            rules={[{ required: true, message: '请输入权限类型' }]}
            extra="建议使用英文小写和下划线，例如 qa_reviewer"
          >
            <Input disabled={!!editing} maxLength={64} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input maxLength={255} />
          </Form.Item>
          {editing && (
            <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
              <Select>
                <Option value="active">启用</Option>
                <Option value="disabled">禁用</Option>
              </Select>
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Drawer
        open={permOpen}
        title={`权限配置 - ${permRoleName || '-'}`}
        width={isMobile ? '100%' : 620}
        onClose={() => setPermOpen(false)}
        extra={isMobile ? undefined : <Button type="primary" loading={permSaving} onClick={submitPermissionConfig}>保存</Button>}
      >
        {isMobile && (
          <Button type="primary" loading={permSaving} onClick={submitPermissionConfig} style={{ width: '100%', marginBottom: 12 }}>
            保存
          </Button>
        )}
        <Space direction="vertical" style={{ width: '100%' }} size={20}>
          {Object.entries(permByModule).map(([module, group]) => (
            <Card key={module} title={module} size={isMobile ? 'small' : 'default'}>
              <Checkbox.Group
                value={permChecked.filter((code) => group.some((item) => item.permission_code === code))}
                onChange={(vals) => handleModulePermissionChange(group.map((item) => item.permission_code), vals as string[])}
                style={{ width: '100%' }}
              >
                <Space direction="vertical" style={{ width: '100%' }}>
                  {group.map((item) => (
                    <Checkbox key={item.permission_code} value={item.permission_code}>
                      <Space direction="vertical" size={0}>
                        <Text>{item.name}</Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>{item.permission_code}</Text>
                      </Space>
                    </Checkbox>
                  ))}
                </Space>
              </Checkbox.Group>
            </Card>
          ))}
        </Space>
      </Drawer>
    </div>
  )
}

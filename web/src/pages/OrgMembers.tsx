import { Button, Card, Form, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
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

export default function OrgMembers() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [currentOrgId, setCurrentOrgId] = useState<string>(sessionStorage.getItem('org_id') || '')
  const [modalMode, setModalMode] = useState<'edit' | null>(null)
  const [selectedOrg, setSelectedOrg] = useState<any | null>(null)
  const [transferOpen, setTransferOpen] = useState(false)
  const [members, setMembers] = useState<any[]>([])
  const [loadingMembers, setLoadingMembers] = useState(false)
  const [planTypes, setPlanTypes] = useState<string[]>(DEFAULT_PLAN_TYPES)
  const [form] = Form.useForm()
  const [transferForm] = Form.useForm()

  const loadOrgs = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/orgs')
      const items = res.data.items || []
      setOrgs(items.filter((org: any) => (org.org_type || org.OrgType || '').toLowerCase() !== 'personal'))
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载组织失败')
    } finally {
      setLoading(false)
    }
  }

  const loadMembers = async (orgId: string) => {
    setLoadingMembers(true)
    try {
      const res = await api.get(`/api/orgs/${orgId}/members`)
      setMembers(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载成员失败')
    } finally {
      setLoadingMembers(false)
    }
  }

  useEffect(() => {
    loadOrgs()
    loadPlanTypes()
  }, [])

  const loadPlanTypes = async () => {
    try {
      const res = await api.get('/api/public/settings')
      const items = Array.isArray(res?.data?.org_plan_types) ? res.data.org_plan_types : []
      const normalized = items
        .map((item: string) => (item || '').toLowerCase().trim())
        .filter((item: string) => item === 'free' || item === 'team' || item === 'enterprise')
      setPlanTypes(normalized.length > 0 ? normalized : DEFAULT_PLAN_TYPES)
    } catch {
      setPlanTypes(DEFAULT_PLAN_TYPES)
    }
  }

  const openEdit = (org: any) => {
    setModalMode('edit')
    setSelectedOrg(org)
    form.setFieldsValue({
      name: org.name || org.Name,
      plan: org.plan || org.Plan
    })
  }

  const closeModal = () => {
    setModalMode(null)
    setSelectedOrg(null)
    form.resetFields()
  }

  const saveOrg = async () => {
    try {
      const values = await form.validateFields()
      if (modalMode === 'edit' && selectedOrg) {
        const orgId = selectedOrg.id || selectedOrg.ID
        await api.patch(`/api/orgs/${orgId}`, { name: values.name })
        message.success('组织已更新')
      }
      closeModal()
      loadOrgs()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '操作失败')
    }
  }

  const openTransfer = async (org: any) => {
    const orgId = org.id || org.ID
    if (orgId !== currentOrgId) {
      message.info('请先切换到该组织再进行转移')
      return
    }
    setSelectedOrg(org)
    setTransferOpen(true)
    transferForm.resetFields()
    await loadMembers(orgId)
  }

  const transferOwner = async () => {
    if (!selectedOrg) return
    try {
      const values = await transferForm.validateFields()
      const orgId = selectedOrg.id || selectedOrg.ID
      await api.post(`/api/orgs/${orgId}/transfer-owner`, { new_owner_user_id: values.new_owner_user_id })
      message.success('所有者已转移')
      setTransferOpen(false)
      transferForm.resetFields()
      loadOrgs()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '转移失败')
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>组织管理</Title>
        <Text type="secondary">编辑与转移组织所有者</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Text type="secondary">当前组织：{currentOrgId || '-'}</Text>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Table
          rowKey={(row) => row.id || row.ID}
          dataSource={orgs}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1080 } : { x: 1200 }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            {
              title: '名称',
              width: 180,
              dataIndex: 'name',
              render: (_: any, row: any) => row.name || row.Name
            },
            {
              title: '企业属性',
              width: 260,
              render: (_: any, row: any) => {
                const orgId = row.id || row.ID || '-'
                return (
                  <Text copyable={{ text: orgId }}>ID：{orgId}</Text>
                )
              }
            },
            {
              title: '套餐',
              dataIndex: 'plan',
              width: 120,
              render: (_: any, row: any) => {
                const plan = (row.plan || row.Plan || 'free').toLowerCase()
                return <Tag color="blue">{formatPlanLabel(plan)}</Tag>
              }
            },
            {
              title: '角色',
              width: 120,
              dataIndex: 'role',
              render: (_: any, row: any) => row.role || row.Role || '-'
            },
            {
              title: '成员数',
              width: 100,
              dataIndex: 'member_count',
              render: (_: any, row: any) => row.member_count ?? row.MemberCount ?? '-'
            },
            {
              title: '应用数',
              width: 100,
              dataIndex: 'app_count',
              render: (_: any, row: any) => row.app_count ?? row.AppCount ?? '-'
            },
            {
              title: '操作',
              width: 210,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: any) => {
                const orgId = row.id || row.ID
                const isCurrent = orgId === currentOrgId
                const rowRole = (row.role || row.Role || '').toLowerCase()
                const canEdit = isCurrent && (rowRole === 'owner' || rowRole === 'admin')
                const canTransfer = isCurrent && rowRole === 'owner'
                return (
                  <Space size={[6, 6]} wrap>
                    <Button size="small" onClick={() => openEdit(row)} disabled={!canEdit}>{isMobile ? '编辑' : '编辑组织'}</Button>
                    <Button size="small" onClick={() => openTransfer(row)} disabled={!canTransfer}>{isMobile ? '转移' : '转移所有者'}</Button>
                  </Space>
                )
              }
            }
          ]}
        />
      </Card>

      <Modal
        open={modalMode !== null}
        title="编辑组织"
        onOk={saveOrg}
        onCancel={closeModal}
        okText="保存"
        cancelText="取消"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={form}>
          <Form.Item name="name" label="组织名称" rules={[{ required: true, message: '请输入组织名称' }]}>
            <Input placeholder="组织名称" />
          </Form.Item>
          <Form.Item
            name="plan"
            label="计划"
            extra={<Text type="secondary" style={{ fontSize: 12 }}>企业计划不可自行更改，如需更改请发送工单联系我们</Text>}
          >
            <Select placeholder="请选择套餐类型" disabled>
              {planTypes.map((plan) => (
                <Option key={plan} value={plan}>
                  {formatPlanLabel(plan)}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={transferOpen}
        title="转移组织所有者"
        onOk={transferOwner}
        onCancel={() => setTransferOpen(false)}
        okText="确认转移"
        cancelText="取消"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={transferForm}>
          <Form.Item
            name="new_owner_user_id"
            label="选择新所有者"
            rules={[{ required: true, message: '请选择新所有者' }]}
          >
            <Select loading={loadingMembers} placeholder="选择成员">
              {members.map((m) => (
                <Option key={m.UserID || m.user_id} value={m.UserID || m.user_id}>
                  {(m.Email || m.email) ?? (m.UserID || m.user_id)}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

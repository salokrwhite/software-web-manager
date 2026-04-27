import { Button, Card, Empty, Form, Grid, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import type { Key } from 'react'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select
const defaultRoles = ['admin', 'dev', 'viewer']

export default function MemberInvites() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const orgId = sessionStorage.getItem('org_id') || ''
  const [orgName, setOrgName] = useState('')
  const [invites, setInvites] = useState<any[]>([])
  const [loadingInvites, setLoadingInvites] = useState(false)
  const [creatingInvite, setCreatingInvite] = useState(false)
  const [bulkDeleting, setBulkDeleting] = useState(false)
  const [inviteLinks, setInviteLinks] = useState<Record<string, string>>({})
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [joinRequests, setJoinRequests] = useState<any[]>([])
  const [loadingJoinRequests, setLoadingJoinRequests] = useState(false)
  const [reviewingRequestId, setReviewingRequestId] = useState<string>('')
  const [selectedJoinRequestKeys, setSelectedJoinRequestKeys] = useState<Key[]>([])
  const [deletingJoinRequests, setDeletingJoinRequests] = useState(false)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [rejectTarget, setRejectTarget] = useState<any>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [rejectSubmitting, setRejectSubmitting] = useState(false)
  const [roles, setRoles] = useState<string[]>(defaultRoles)
  const [inviteForm] = Form.useForm()

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

  const loadInvites = async () => {
    if (!orgId) return
    setLoadingInvites(true)
    try {
      const res = await api.get(`/api/orgs/${orgId}/invites`)
      setInvites(res.data.items || [])
      setSelectedRowKeys([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载邀请失败')
    } finally {
      setLoadingInvites(false)
    }
  }

  const loadJoinRequests = async () => {
    if (!orgId) return
    setLoadingJoinRequests(true)
    try {
      const res = await api.get(`/api/orgs/${orgId}/join-requests`)
      setJoinRequests(res.data.items || [])
      setSelectedJoinRequestKeys([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载加入申请失败')
    } finally {
      setLoadingJoinRequests(false)
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
    loadInvites()
    loadJoinRequests()
  }, [orgId])

  const createInvite = async () => {
    if (!orgId) return
    try {
      const values = await inviteForm.validateFields()
      setCreatingInvite(true)
      const res = await api.post(`/api/orgs/${orgId}/invites`, values)
      const inviteId = res.data.invite_id
      const link = res.data.invite_link
      if (inviteId && link) {
        setInviteLinks((prev) => ({ ...prev, [inviteId]: link }))
        try {
          await navigator.clipboard.writeText(link)
          message.success('邀请已生成，链接已复制')
        } catch {
          message.success('邀请已生成')
        }
      } else {
        message.success('邀请已生成')
      }
      inviteForm.resetFields()
      loadInvites()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '邀请失败')
    } finally {
      setCreatingInvite(false)
    }
  }

  const revokeInvite = async (inviteId: string) => {
    if (!orgId) return
    try {
      await api.delete(`/api/orgs/${orgId}/invites/${inviteId}`)
      message.success('邀请已撤销')
      loadInvites()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '撤销失败')
    }
  }

  const handleBulkDelete = async () => {
    if (!orgId) return
    if (selectedRowKeys.length === 0) {
      message.info('请选择要删除的邀请记录')
      return
    }
    Modal.confirm({
      title: '批量删除邀请',
      content: `确定删除选中的 ${selectedRowKeys.length} 条邀请记录吗？`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        setBulkDeleting(true)
        try {
          const res = await api.post(`/api/orgs/${orgId}/invites/batch-delete`, {
            invite_ids: selectedRowKeys.map((key) => String(key))
          })
          const deleted = Number(res?.data?.deleted || 0)
          if (deleted < selectedRowKeys.length) {
            message.warning(`已删除 ${deleted} 条，${selectedRowKeys.length - deleted} 条失败`)
          } else {
            message.success('邀请记录已删除')
          }
          await loadInvites()
        } finally {
          setBulkDeleting(false)
        }
      }
    })
  }

  const copyInviteLink = async (inviteId: string) => {
    const link = inviteLinks[inviteId]
    if (!link) {
      message.info('请重新生成邀请链接')
      return
    }
    try {
      await navigator.clipboard.writeText(link)
      message.success('链接已复制')
    } catch {
      message.error('复制失败')
    }
  }

  const approveJoinRequest = async (row: any) => {
    if (!orgId) return
    const requestId = row.id || row.ID
    if (!requestId) return
    setReviewingRequestId(String(requestId))
    try {
      await api.post(`/api/orgs/${orgId}/join-requests/${requestId}/approve`)
      message.success('已通过加入申请')
      await loadJoinRequests()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '审批失败')
    } finally {
      setReviewingRequestId('')
    }
  }

  const openRejectModal = (row: any) => {
    setRejectTarget(row)
    setRejectReason('')
    setRejectOpen(true)
  }

  const submitReject = async () => {
    if (!orgId || !rejectTarget) return
    const requestId = rejectTarget.id || rejectTarget.ID
    if (!requestId) return
    const reason = rejectReason.trim()
    if (!reason) {
      message.warning('请填写驳回理由')
      return
    }
    setRejectSubmitting(true)
    try {
      await api.post(`/api/orgs/${orgId}/join-requests/${requestId}/reject`, { reason })
      message.success('已驳回加入申请')
      setRejectOpen(false)
      setRejectTarget(null)
      setRejectReason('')
      await loadJoinRequests()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '驳回失败')
    } finally {
      setRejectSubmitting(false)
    }
  }

  const handleBatchDeleteJoinRequests = async () => {
    if (!orgId) return
    if (selectedJoinRequestKeys.length === 0) {
      message.info('请选择要删除的申请记录')
      return
    }
    Modal.confirm({
      title: '批量删除申请记录',
      content: '仅删除申请记录，不影响已产生的实际申请结果。',
      okText: '删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setDeletingJoinRequests(true)
        try {
          const res = await api.post(`/api/orgs/${orgId}/join-requests/batch-delete`, {
            ids: selectedJoinRequestKeys.map((key) => String(key))
          })
          const deleted = Number(res?.data?.deleted || 0)
          message.success(`已删除 ${deleted} 条申请记录`)
          await loadJoinRequests()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        } finally {
          setDeletingJoinRequests(false)
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>邀请成员</Title>
        <Text type="secondary">生成邀请链接以加入当前组织</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Text type="secondary">当前组织：</Text>
        <Text style={{ marginLeft: 8 }}>{orgName || orgId || '-'}</Text>
      </Card>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Title level={5} style={{ marginTop: 0 }}>生成邀请</Title>
        <Form layout={isMobile ? 'vertical' : 'inline'} form={inviteForm}>
          <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式错误' }]}>
            <Input placeholder="成员邮箱" style={{ width: isMobile ? '100%' : 220 }} />
          </Form.Item>
          <Form.Item name="role" rules={[{ required: true, message: '请选择角色' }]}>
            <Select placeholder="角色" style={{ width: isMobile ? '100%' : 140 }}>
              {roles.map((r) => (
                <Option key={r} value={r}>{r}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="expires_in_days" initialValue={7}>
            <InputNumber min={1} max={365} placeholder="有效期(天)" style={{ width: isMobile ? '100%' : 140 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={createInvite} loading={creatingInvite} style={isMobile ? { width: '100%' } : undefined}>生成邀请</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          align="center"
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}
        >
          <Title level={5} style={{ margin: 0 }}>来自用户的加入申请</Title>
          <Button
            danger
            onClick={handleBatchDeleteJoinRequests}
            loading={deletingJoinRequests}
            disabled={selectedJoinRequestKeys.length === 0}
            style={isMobile ? { width: '100%' } : undefined}
          >
            批量删除
          </Button>
        </Space>
        <Table
          rowKey={(row) => row.id || row.ID}
          dataSource={joinRequests}
          loading={loadingJoinRequests}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1180 } : { x: 1280 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无数据" /> }}
          rowSelection={{
            selectedRowKeys: selectedJoinRequestKeys,
            onChange: (keys) => setSelectedJoinRequestKeys(keys)
          }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50'],
            locale: { items_per_page: '条/页' },
            showTotal: (total) => `共 ${total} 条`
          }}
          columns={[
            { title: '申请用户', dataIndex: 'user_email', width: 220 },
            {
              title: '状态',
              dataIndex: 'status',
              width: 110,
              render: (v: string) => {
                const status = (v || '').toLowerCase()
                if (status === 'pending') return <Tag color="orange">待审核</Tag>
                if (status === 'approved') return <Tag color="green">已通过</Tag>
                if (status === 'rejected') return <Tag color="red">已驳回</Tag>
                return <Tag>{v || '未知'}</Tag>
              }
            },
            {
              title: '申请理由',
              dataIndex: 'reason',
              width: 280,
              render: (v: string) => (
                <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {v || '-'}
                </div>
              )
            },
            {
              title: '驳回理由',
              dataIndex: 'review_reason',
              width: 280,
              render: (v: string) => (
                <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {v || '-'}
                </div>
              )
            },
            {
              title: '申请时间',
              dataIndex: 'created_at',
              width: 180,
              render: (v: string) => v ? new Date(v).toLocaleString() : '-'
            },
            {
              title: '操作',
              width: 150,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: any) => {
                const status = String(row.status || '').toLowerCase()
                const pending = status === 'pending'
                const requestId = String(row.id || row.ID || '')
                return (
                  <Space size={[6, 6]} wrap>
                    <Button
                      size="small"
                      type="primary"
                      disabled={!pending}
                      loading={reviewingRequestId === requestId}
                      onClick={() => approveJoinRequest(row)}
                    >
                      通过
                    </Button>
                    <Button
                      size="small"
                      danger
                      disabled={!pending}
                      onClick={() => openRejectModal(row)}
                    >
                      {isMobile ? '拒绝' : '驳回'}
                    </Button>
                  </Space>
                )
              }
            }
          ]}
        />
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Space
          align="center"
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}
        >
          <Title level={5} style={{ margin: 0 }}>邀请列表</Title>
          <Button
            danger
            onClick={handleBulkDelete}
            loading={bulkDeleting}
            disabled={selectedRowKeys.length === 0}
            style={isMobile ? { width: '100%' } : undefined}
          >
            批量删除
          </Button>
        </Space>
        <Table
          rowKey={(row) => row.id || row.ID}
          dataSource={invites}
          loading={loadingInvites}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 980 } : { x: 1060 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无数据" /> }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys)
          }}
          pagination={{
            pageSize: 10,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50', '100'],
            locale: { items_per_page: '条/页' },
            showTotal: (total) => `共 ${total} 条`
          }}
          columns={[
            { title: '邮箱', dataIndex: 'email', width: 220 },
            { title: '角色', dataIndex: 'role', width: 100 },
            {
              title: '状态',
              dataIndex: 'status',
              width: 120,
              render: (v: string) => {
                const status = (v || '').toLowerCase()
                const color = status === 'active' ? 'green' : status === 'expired' ? 'orange' : status === 'used' ? 'blue' : 'red'
                const label = status === 'active'
                  ? '有效'
                  : status === 'used'
                    ? '已使用'
                    : status === 'revoked'
                      ? '已撤销'
                      : status === 'expired'
                        ? '已过期'
                        : v
                return <Tag color={color}>{label}</Tag>
              }
            },
            {
              title: '过期时间',
              dataIndex: 'expires_at',
              width: 180,
              render: (v: string) => v ? new Date(v).toLocaleString() : '-'
            },
            {
              title: '创建时间',
              dataIndex: 'created_at',
              width: 180,
              render: (v: string) => v ? new Date(v).toLocaleString() : '-'
            },
            {
              title: '操作',
              width: 170,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: any) => (
                <Space size={[6, 6]} wrap>
                  <Button size="small" onClick={() => copyInviteLink(row.id || row.ID)}>{isMobile ? '复制' : '复制链接'}</Button>
                  <Button size="small" danger onClick={() => revokeInvite(row.id || row.ID)}>撤销</Button>
                </Space>
              )
            }
          ]}
        />
      </Card>
      <Modal
        open={rejectOpen}
        title="驳回加入申请"
        okText="确认驳回"
        cancelText="取消"
        okButtonProps={{ danger: true }}
        confirmLoading={rejectSubmitting}
        onOk={submitReject}
        onCancel={() => setRejectOpen(false)}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <div style={{ paddingBottom: 12 }}>
          <Input.TextArea
            rows={4}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            maxLength={300}
            showCount
            placeholder="请填写驳回理由（必填）"
          />
        </div>
      </Modal>
    </div>
  )
}

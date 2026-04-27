import { useEffect, useMemo, useState } from 'react'
import { Button, Card, Input, Modal, Space, Table, Tag, Typography, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import api, { storeTokens } from '../api/client'

const { Title, Text } = Typography
const { TextArea } = Input

const statusTag = (status: string) => {
  const value = (status || '').toLowerCase()
  if (value === 'active') return <Tag color="green">正常</Tag>
  if (value === 'pending') return <Tag color="orange">审核中</Tag>
  if (value === 'rejected') return <Tag color="red">已驳回</Tag>
  if (value === 'disabled') return <Tag color="red">已停用</Tag>
  return <Tag>未知</Tag>
}

const inviteStatusTag = (status: string) => {
  const value = (status || '').toLowerCase()
  if (value === 'active') return <Tag color="green">可接受</Tag>
  if (value === 'used') return <Tag>已使用</Tag>
  if (value === 'expired') return <Tag color="orange">已过期</Tag>
  if (value === 'revoked') return <Tag color="red">已撤销</Tag>
  return <Tag>未知</Tag>
}

const joinRequestStatusTag = (status: string) => {
  const value = (status || '').toLowerCase()
  if (value === 'approved') return <Tag color="green">已通过</Tag>
  if (value === 'rejected') return <Tag color="red">已驳回</Tag>
  if (value === 'pending') return <Tag color="orange">待审核</Tag>
  if (value === 'withdrawn') return <Tag>已撤回</Tag>
  return <Tag>{status || '未知'}</Tag>
}

const joinRequestStatusText = (status: string) => {
  const value = (status || '').toLowerCase()
  if (value === 'approved') return '已通过'
  if (value === 'rejected') return '已驳回'
  if (value === 'pending') return '待审核'
  if (value === 'withdrawn') return '已撤回'
  return status || '未知'
}

export default function JoinOrg() {
  const navigate = useNavigate()
  const [searchId, setSearchId] = useState('')
  const [searching, setSearching] = useState(false)
  const [searchResult, setSearchResult] = useState<any>(null)
  const [invites, setInvites] = useState<any[]>([])
  const [loadingInvites, setLoadingInvites] = useState(false)
  const [acceptingId, setAcceptingId] = useState<string>('')
  const [myRequests, setMyRequests] = useState<any[]>([])
  const [loadingMyRequests, setLoadingMyRequests] = useState(false)
  const [applyOpen, setApplyOpen] = useState(false)
  const [applyReason, setApplyReason] = useState('')
  const [submittingApply, setSubmittingApply] = useState(false)
  const [selectedRequestKeys, setSelectedRequestKeys] = useState<string[]>([])
  const [deletingRequests, setDeletingRequests] = useState(false)
  const [withdrawingRequests, setWithdrawingRequests] = useState(false)

  const loadInvites = async () => {
    setLoadingInvites(true)
    try {
      const res = await api.get('/api/org-invites')
      setInvites(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载邀请失败')
    } finally {
      setLoadingInvites(false)
    }
  }

  const loadMyRequests = async () => {
    setLoadingMyRequests(true)
    try {
      const res = await api.get('/api/org-join-requests/mine')
      setMyRequests(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载申请失败')
    } finally {
      setLoadingMyRequests(false)
    }
  }

  useEffect(() => {
    loadInvites()
    loadMyRequests()
  }, [])

  const handleSearch = async () => {
    const id = searchId.trim()
    if (!id) {
      message.warning('请输入企业 ID')
      return
    }
    setSearching(true)
    try {
      const res = await api.get(`/api/orgs/${id}/public`)
      setSearchResult(res.data)
    } catch (err: any) {
      setSearchResult(null)
      message.error(err?.response?.data?.error || '未找到企业')
    } finally {
      setSearching(false)
    }
  }

  const latestRequestStatus = useMemo(() => {
    if (!searchResult?.id) return ''
    const found = myRequests.find((item) => String(item.org_id) === String(searchResult.id))
    return (found?.status || '').toLowerCase()
  }, [myRequests, searchResult])

  const openApplyModal = () => {
    if (!searchResult?.id) return
    setApplyReason('')
    setApplyOpen(true)
  }

  const submitJoinRequest = async () => {
    const orgId = String(searchResult?.id || '')
    if (!orgId) return
    setSubmittingApply(true)
    try {
      await api.post(`/api/orgs/${orgId}/join-requests`, {
        reason: applyReason.trim() || undefined
      })
      message.success('申请已提交，请等待企业管理员审核')
      setApplyOpen(false)
      setApplyReason('')
      await loadMyRequests()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '提交申请失败')
    } finally {
      setSubmittingApply(false)
    }
  }

  const handleBatchDeleteMyRequests = () => {
    if (selectedRequestKeys.length === 0) {
      message.info('请选择要删除的申请记录')
      return
    }
    Modal.confirm({
      title: '确认删除选中的申请记录？',
      content: '仅删除申请记录，不影响已产生的实际申请结果。',
      okText: '删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setDeletingRequests(true)
        try {
          const res = await api.post('/api/org-join-requests/mine/batch-delete', {
            ids: selectedRequestKeys
          })
          const deleted = Number(res?.data?.deleted || 0)
          message.success(`已删除 ${deleted} 条申请记录`)
          setSelectedRequestKeys([])
          await loadMyRequests()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        } finally {
          setDeletingRequests(false)
        }
      }
    })
  }

  const handleBatchWithdrawMyRequests = () => {
    if (selectedRequestKeys.length === 0) {
      message.info('请选择要撤回的申请记录')
      return
    }
    const selectedItems = myRequests.filter((item) => selectedRequestKeys.includes(String(item.id)))
    const hasNonPending = selectedItems.some((item) => String(item.status || '').toLowerCase() !== 'pending')
    if (hasNonPending) {
      message.warning('仅支持撤回待审核申请')
      return
    }
    Modal.confirm({
      title: '确认撤回选中的申请？',
      content: '仅支持撤回待审核申请，撤回后管理员无法继续审批该记录。',
      okText: '撤回',
      cancelText: '取消',
      onOk: async () => {
        setWithdrawingRequests(true)
        try {
          const res = await api.post('/api/org-join-requests/mine/batch-withdraw', {
            ids: selectedRequestKeys
          })
          const withdrawn = Number(res?.data?.withdrawn || 0)
          message.success(`已撤回 ${withdrawn} 条申请`)
          setSelectedRequestKeys([])
          await loadMyRequests()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '撤回失败')
        } finally {
          setWithdrawingRequests(false)
        }
      }
    })
  }

  const acceptInvite = async (record: any) => {
    try {
      setAcceptingId(record.id)
      const res = await api.post(`/api/org-invites/accept-by-id/${record.id}`)
      storeTokens(res.data.tokens)
      if (res.data.org_id) {
        sessionStorage.setItem('org_id', res.data.org_id)
      } else {
        sessionStorage.removeItem('org_id')
      }
      if (res.data.role) {
        sessionStorage.setItem('role', res.data.role)
      } else {
        sessionStorage.removeItem('role')
      }
      if (res.data.user?.email) {
        sessionStorage.setItem('user_email', res.data.user.email)
      } else {
        sessionStorage.removeItem('user_email')
      }
      if (res.data.org_type) {
        sessionStorage.setItem('org_type', res.data.org_type)
      } else {
        sessionStorage.removeItem('org_type')
      }
      sessionStorage.removeItem('impersonating')
      sessionStorage.removeItem('impersonation_org_id')
      sessionStorage.removeItem('system_backup_access_token')
      sessionStorage.removeItem('system_backup_refresh_token')
      sessionStorage.removeItem('system_backup_org_id')
      sessionStorage.removeItem('system_backup_role')
      if (res.data.system_role) {
        sessionStorage.setItem('system_role', res.data.system_role)
      } else {
        sessionStorage.removeItem('system_role')
      }
      message.success('加入企业成功')
      navigate('/dashboard')
    } catch (err: any) {
      message.error(err?.response?.data?.error || '接受邀请失败')
    } finally {
      setAcceptingId('')
    }
  }

  return (
    <div style={{ width: '100%' }}>
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        <div>
          <Title level={3} style={{ marginBottom: 4 }}>加入企业</Title>
          <Text type="secondary">通过企业 ID 搜索或从邀请列表加入企业。</Text>
        </div>

        <Card title="企业搜索">
          <Space wrap>
            <Input
              placeholder="请输入企业 ID"
              value={searchId}
              onChange={(e) => setSearchId(e.target.value)}
              style={{ width: 320 }}
            />
            <Button type="primary" loading={searching} onClick={handleSearch}>搜索</Button>
          </Space>
          {searchResult && (
            <Card size="small" style={{ marginTop: 16, borderRadius: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
                <Space direction="vertical" size={4}>
                  <Text strong>{searchResult.name}</Text>
                  <Space size={8}>
                    {statusTag(searchResult.status)}
                    <Text type="secondary">企业 ID：{searchResult.id}</Text>
                  </Space>
                  {latestRequestStatus && (
                    <Text type="secondary">最近申请状态：{joinRequestStatusText(latestRequestStatus)}</Text>
                  )}
                  {searchResult.status && (searchResult.status || '').toLowerCase() !== 'active' && (
                    <Text type="warning">该企业当前状态不可加入</Text>
                  )}
                  {latestRequestStatus === 'pending' && (
                    <Text type="warning">您已提交申请，请等待审核</Text>
                  )}
                </Space>
                <Button
                  type="primary"
                  onClick={openApplyModal}
                  disabled={
                    (searchResult.status || '').toLowerCase() !== 'active' ||
                    latestRequestStatus === 'pending'
                  }
                >
                  申请加入
                </Button>
              </div>
            </Card>
          )}
        </Card>

        <Card title="我的邀请">
          <Table
            rowKey="id"
            dataSource={invites}
            loading={loadingInvites}
            pagination={{ pageSize: 5 }}
            columns={[
              { title: '企业', dataIndex: 'org_name' },
              { title: '角色', dataIndex: 'role' },
              {
                title: '状态',
                dataIndex: 'status',
                render: (v: string) => inviteStatusTag(v)
              },
              {
                title: '有效期',
                dataIndex: 'expires_at',
                render: (v: string) => v ? new Date(v).toLocaleDateString() : '-'
              },
              {
                title: '创建时间',
                dataIndex: 'created_at',
                render: (v: string) => v ? new Date(v).toLocaleDateString() : '-'
              },
              {
                title: '操作',
                render: (_: any, record: any) => (
                  <Button
                    size="small"
                    type="primary"
                    disabled={(record.status || '').toLowerCase() !== 'active'}
                    loading={acceptingId === record.id}
                    onClick={() => acceptInvite(record)}
                  >
                    接受
                  </Button>
                )
              }
            ]}
          />
        </Card>

        <Card title="我的申请" extra={(
          <Space>
            <Button
              loading={withdrawingRequests}
              disabled={selectedRequestKeys.length === 0}
              onClick={handleBatchWithdrawMyRequests}
            >
              撤回请求
            </Button>
            <Button
              danger
              loading={deletingRequests}
              disabled={selectedRequestKeys.length === 0}
              onClick={handleBatchDeleteMyRequests}
            >
              批量删除
            </Button>
          </Space>
        )}>
          <Table
            rowKey="id"
            dataSource={myRequests}
            loading={loadingMyRequests}
            rowSelection={{
              selectedRowKeys: selectedRequestKeys,
              onChange: (keys) => setSelectedRequestKeys(keys as string[])
            }}
            pagination={{ pageSize: 5 }}
            columns={[
              { title: '企业', dataIndex: 'org_name' },
              {
                title: '状态',
                dataIndex: 'status',
                render: (v: string) => joinRequestStatusTag(v)
              },
              {
                title: '申请理由',
                dataIndex: 'reason',
                render: (v: string) => (
                  <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                    {v || '-'}
                  </div>
                )
              },
              {
                title: '驳回理由',
                dataIndex: 'review_reason',
                render: (v: string) => (
                  <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                    {v || '-'}
                  </div>
                )
              },
              {
                title: '申请时间',
                dataIndex: 'created_at',
                render: (v: string) => v ? new Date(v).toLocaleString() : '-'
              }
            ]}
          />
        </Card>
      </Space>
      <Modal
        open={applyOpen}
        title="申请加入企业"
        okText="提交申请"
        cancelText="取消"
        confirmLoading={submittingApply}
        onOk={submitJoinRequest}
        onCancel={() => setApplyOpen(false)}
      >
        <Space direction="vertical" size={8} style={{ width: '100%', paddingBottom: 12 }}>
          <Text type="secondary">申请理由（选填）</Text>
          <TextArea
            value={applyReason}
            onChange={(e) => setApplyReason(e.target.value)}
            rows={4}
            maxLength={300}
            showCount
            placeholder="请简要说明申请加入企业的原因"
          />
        </Space>
      </Modal>
    </div>
  )
}

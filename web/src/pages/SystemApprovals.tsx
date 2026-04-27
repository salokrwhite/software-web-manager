import { useEffect, useState } from 'react'
import { Button, Card, Grid, Space, Table, Tag, Typography, message, Modal, Form, Input, Switch, Tabs, Tooltip } from 'antd'
import api from '../api/client'

const { Title, Text } = Typography

type SystemOrgItem = {
  id: string
  name: string
  owner_email: string
  status: string
  created_at: string
}

type SystemAppItem = {
  id: string
  name: string
  slug: string
  org_name: string
  owner_email: string
  status: string
  created_at: string
  submitted_at?: string
  rejection_reason?: string | null
  submit_note?: string
}

type SystemReleaseItem = {
  id: string
  version: string
  status: string
  submitted_at?: string
  created_at: string
  app_id: string
  app_name: string
  app_slug: string
  org_name: string
  owner_email: string
  submit_note?: string
}

type ApprovalView = 'orgs' | 'apps'

export default function SystemApprovals({ view }: { view?: ApprovalView }) {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [items, setItems] = useState<SystemOrgItem[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedOrgKeys, setSelectedOrgKeys] = useState<string[]>([])
  const [materialsOpen, setMaterialsOpen] = useState(false)
  const [materialsLoading, setMaterialsLoading] = useState(false)
  const [materials, setMaterials] = useState<any[]>([])
  const [materialsOrg, setMaterialsOrg] = useState<SystemOrgItem | null>(null)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [rejecting, setRejecting] = useState(false)
  const [rejectOrg, setRejectOrg] = useState<SystemOrgItem | null>(null)
  const [rejectForm] = Form.useForm()
  const [appItems, setAppItems] = useState<SystemAppItem[]>([])
  const [appLoading, setAppLoading] = useState(false)
  const [selectedAppLogIds, setSelectedAppLogIds] = useState<string[]>([])
  const [appRejectOpen, setAppRejectOpen] = useState(false)
  const [appRejecting, setAppRejecting] = useState(false)
  const [appRejectTarget, setAppRejectTarget] = useState<SystemAppItem | null>(null)
  const [appRejectForm] = Form.useForm()
  const [releaseItems, setReleaseItems] = useState<SystemReleaseItem[]>([])
  const [releaseLoading, setReleaseLoading] = useState(false)
  const [selectedReleaseKeys, setSelectedReleaseKeys] = useState<string[]>([])
  const [noteOpen, setNoteOpen] = useState(false)
  const [noteTitle, setNoteTitle] = useState('')
  const [noteContent, setNoteContent] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/system/orgs')
      setItems(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载审核列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadApps = async () => {
    setAppLoading(true)
    try {
      const res = await api.get('/api/system/apps', { params: { org_type: 'personal' } })
      setAppItems(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载应用审核列表失败')
    } finally {
      setAppLoading(false)
    }
  }

  const loadReleases = async () => {
    setReleaseLoading(true)
    try {
      const res = await api.get('/api/system/releases', { params: { org_type: 'personal' } })
      setReleaseItems(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载版本审核列表失败')
    } finally {
      setReleaseLoading(false)
    }
  }

  useEffect(() => {
    if (view === 'orgs') {
      load()
      return
    }
    if (view === 'apps') {
      loadApps()
      loadReleases()
      return
    }
    load()
    loadApps()
    loadReleases()
  }, [view])

  const approveOrg = async (orgId: string) => {
    try {
      await api.post(`/api/system/orgs/${orgId}/approve`)
      message.success('已审批通过')
      load()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '审批失败')
    }
  }

  const openMaterials = async (org: SystemOrgItem) => {
    setMaterialsOrg(org)
    setMaterialsOpen(true)
    setMaterialsLoading(true)
    try {
      const res = await api.get(`/api/system/orgs/${org.id}/materials`)
      setMaterials(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载材料失败')
      setMaterials([])
    } finally {
      setMaterialsLoading(false)
    }
  }

  const formatSize = (size: number) => {
    if (!size) return '-'
    if (size < 1024) return `${size} B`
    if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`
    return `${(size / (1024 * 1024)).toFixed(1)} MB`
  }

  const statusTag = (status: string) => {
    const s = (status || '').toLowerCase()
    if (s === 'active') return <Tag color="green">已通过</Tag>
    if (s === 'pending') return <Tag color="orange">待审核</Tag>
    if (s === 'rejected') return <Tag color="red">已驳回</Tag>
    if (s === 'disabled') return <Tag color="red">已禁用</Tag>
    return <Tag>{status}</Tag>
  }

  const releaseStatusTag = (status: string) => {
    const s = (status || '').toLowerCase()
    if (s === 'in_review') return <Tag color="orange">待审核</Tag>
    if (s === 'approved') return <Tag color="green">已通过</Tag>
    if (s === 'rejected') return <Tag color="red">已驳回</Tag>
    if (s === 'published') return <Tag color="blue">已发布</Tag>
    return <Tag>{status}</Tag>
  }

  const openNote = (title: string, note?: string | null) => {
    setNoteTitle(title)
    const value = (note || '').trim()
    setNoteContent(value === '' ? '无' : value)
    setNoteOpen(true)
  }

  const openReject = (org: SystemOrgItem) => {
    setRejectOrg(org)
    setRejectOpen(true)
    rejectForm.setFieldsValue({ reason: '', allow_resubmit: true })
  }

  const submitReject = async () => {
    if (!rejectOrg) return
    try {
      const values = await rejectForm.validateFields()
      setRejecting(true)
      await api.post(`/api/system/orgs/${rejectOrg.id}/reject`, values)
      message.success('已驳回企业申请')
      setRejectOpen(false)
      setRejectOrg(null)
      rejectForm.resetFields()
      load()
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || '驳回失败')
    } finally {
      setRejecting(false)
    }
  }

  const approveApp = async (appId: string) => {
    try {
      await api.post(`/api/system/apps/${appId}/approve`)
      message.success('应用已通过审核')
      loadApps()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '审批失败')
    }
  }

  const openAppReject = (app: SystemAppItem) => {
    setAppRejectTarget(app)
    setAppRejectOpen(true)
    appRejectForm.setFieldsValue({ reason: '' })
  }

  const submitAppReject = async () => {
    if (!appRejectTarget) return
    try {
      const values = await appRejectForm.validateFields()
      setAppRejecting(true)
      await api.post(`/api/system/apps/${appRejectTarget.id}/reject`, values)
      message.success('已驳回应用申请')
      setAppRejectOpen(false)
      setAppRejectTarget(null)
      appRejectForm.resetFields()
      loadApps()
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || '驳回失败')
    } finally {
      setAppRejecting(false)
    }
  }

  const handleBatchDeleteOrgLogs = () => {
    if (selectedOrgKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中企业的审核日志？',
      content: '仅删除企业审批审计日志，保留当前审核结果与组织状态。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/audit-logs/org-approvals/batch-delete', { ids: selectedOrgKeys })
          message.success('企业审批日志已删除，审核结果保留')
          setSelectedOrgKeys([])
          await load()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  const handleBatchDeleteAppLogs = () => {
    if (selectedAppLogIds.length === 0) return
    Modal.confirm({
      title: '确认删除选中应用的审核日志？',
      content: '仅删除应用审批审计日志，保留当前审核结果与应用状态。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/audit-logs/app-approvals/batch-delete', { ids: selectedAppLogIds })
          message.success('应用审批日志已删除，审核结果保留')
          setSelectedAppLogIds([])
          await loadApps()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  const handleBatchDeleteReleaseLogs = () => {
    if (selectedReleaseKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中版本的审核日志？',
      content: '仅删除版本审批审计日志，保留当前审核结果与版本状态。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/audit-logs/release-approvals/batch-delete', { ids: selectedReleaseKeys })
          message.success('版本审批日志已删除，审核结果保留')
          setSelectedReleaseKeys([])
          await loadReleases()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  const approveRelease = async (releaseId: string) => {
    try {
      await api.post(`/api/system/releases/${releaseId}/approve`)
      message.success('版本已通过审核')
      loadReleases()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '审批失败')
    }
  }

  const rejectRelease = async (releaseId: string) => {
    Modal.confirm({
      title: '确认驳回该版本？',
      okText: '驳回',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post(`/api/system/releases/${releaseId}/reject`)
          message.success('已驳回版本')
          loadReleases()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '驳回失败')
        }
      }
    })
  }

  const orgTable = (
    <>
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
        <Text type="secondary">已选 {selectedOrgKeys.length} 条</Text>
        <Button danger disabled={selectedOrgKeys.length === 0 || loading} onClick={handleBatchDeleteOrgLogs} style={isMobile ? { width: '100%' } : undefined}>
          批量删除审批日志
        </Button>
      </div>
      <Table
        rowKey={(row) => row.id}
        dataSource={items}
        loading={loading}
        size={isMobile ? 'small' : 'middle'}
        scroll={isMobile ? { x: 980 } : { x: 1080 }}
        rowSelection={{
          selectedRowKeys: selectedOrgKeys,
          onChange: (keys) => setSelectedOrgKeys(keys as string[])
        }}
        pagination={{
          pageSize: 10,
          size: isMobile ? 'small' : 'default',
          responsive: true,
          showSizeChanger: !isMobile
        }}
        columns={[
          { title: '名称', dataIndex: 'name', width: 180 },
          { title: '管理员邮箱', dataIndex: 'owner_email', width: 220 },
          { title: '状态', dataIndex: 'status', width: 110, render: (v: string) => statusTag(v) },
          { title: '申请时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
          {
            title: '操作',
            width: 180,
            fixed: isMobile ? undefined : 'right',
            render: (_: any, row: SystemOrgItem) => (
              <Space size={[6, 6]} wrap>
                <Button size="small" onClick={() => openMaterials(row)}>材料</Button>
                {row.status === 'pending' && (
                  <>
                    <Button size="small" type="primary" onClick={() => approveOrg(row.id)}>{isMobile ? '通过' : '审批'}</Button>
                    <Button size="small" danger onClick={() => openReject(row)}>{isMobile ? '拒绝' : '驳回'}</Button>
                  </>
                )}
              </Space>
            )
          }
        ]}
      />
    </>
  )

  const appTable = (
    <>
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
        <Text type="secondary">已选 {selectedAppLogIds.length} 条</Text>
        <Button danger disabled={selectedAppLogIds.length === 0 || appLoading} onClick={handleBatchDeleteAppLogs} style={isMobile ? { width: '100%' } : undefined}>
          批量删除审批日志
        </Button>
      </div>
      <Table
        rowKey={(row) => row.id}
        dataSource={appItems}
        loading={appLoading}
        size={isMobile ? 'small' : 'middle'}
        scroll={isMobile ? { x: 1400 } : { x: 1500 }}
        rowSelection={{
          selectedRowKeys: selectedAppLogIds,
          onChange: (keys) => setSelectedAppLogIds(keys as string[])
        }}
        pagination={{
          pageSize: 10,
          size: isMobile ? 'small' : 'default',
          responsive: true,
          showSizeChanger: !isMobile
        }}
        columns={[
          { title: '应用名称', dataIndex: 'name', width: 180 },
          { title: 'Slug', dataIndex: 'slug', width: 160 },
          { title: '所属组织', dataIndex: 'org_name', width: 180 },
          { title: '管理员邮箱', dataIndex: 'owner_email', width: 220 },
          { title: '状态', dataIndex: 'status', width: 110, render: (v: string) => statusTag(v) },
          {
            title: '提交时间',
            dataIndex: 'submitted_at',
            width: 180,
            render: (_: any, row: SystemAppItem) => {
              const value = row.submitted_at || row.created_at
              return value ? new Date(value).toLocaleString() : '-'
            }
          },
          {
            title: '驳回原因',
            dataIndex: 'rejection_reason',
            width: 180,
            render: (v: string) => v ? (
              <Tooltip title={v}>
                <span>{v.length > 12 ? `${v.slice(0, 12)}...` : v}</span>
              </Tooltip>
            ) : '-'
          },
          {
            title: '备注',
            dataIndex: 'submit_note',
            width: 100,
            render: (_: any, row: SystemAppItem) => (
              row.submit_note ? (
                <Button size="small" onClick={() => openNote(`应用备注 - ${row.name}`, row.submit_note)}>详情</Button>
              ) : (
                <Text type="secondary">无</Text>
              )
            )
          },
          {
            title: '操作',
            width: 160,
            fixed: isMobile ? undefined : 'right',
            render: (_: any, row: SystemAppItem) => (
              <Space size={[6, 6]} wrap>
                {row.status === 'pending' && (
                  <>
                    <Button size="small" type="primary" onClick={() => approveApp(row.id)}>{isMobile ? '通过' : '审批'}</Button>
                    <Button size="small" danger onClick={() => openAppReject(row)}>{isMobile ? '拒绝' : '驳回'}</Button>
                  </>
                )}
              </Space>
            )
          }
        ]}
      />
    </>
  )

  const releaseTable = (
    <>
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
        <Text type="secondary">已选 {selectedReleaseKeys.length} 条</Text>
        <Button danger disabled={selectedReleaseKeys.length === 0 || releaseLoading} onClick={handleBatchDeleteReleaseLogs} style={isMobile ? { width: '100%' } : undefined}>
          批量删除审批日志
        </Button>
      </div>
      <Table
        rowKey={(row) => row.id}
        dataSource={releaseItems}
        loading={releaseLoading}
        size={isMobile ? 'small' : 'middle'}
        scroll={isMobile ? { x: 1450 } : { x: 1540 }}
        rowSelection={{
          selectedRowKeys: selectedReleaseKeys,
          onChange: (keys) => setSelectedReleaseKeys(keys as string[])
        }}
        pagination={{
          pageSize: 10,
          size: isMobile ? 'small' : 'default',
          responsive: true,
          showSizeChanger: !isMobile
        }}
        columns={[
          { title: '应用名称', dataIndex: 'app_name', width: 180 },
          { title: 'Slug', dataIndex: 'app_slug', width: 160 },
          { title: '版本号', dataIndex: 'version', width: 120 },
          { title: '所属组织', dataIndex: 'org_name', width: 180 },
          { title: '管理员邮箱', dataIndex: 'owner_email', width: 220 },
          { title: '状态', dataIndex: 'status', width: 110, render: (v: string) => releaseStatusTag(v) },
          {
            title: '提交时间',
            dataIndex: 'submitted_at',
            width: 180,
            render: (_: any, row: SystemReleaseItem) => {
              const value = row.submitted_at || row.created_at
              return value ? new Date(value).toLocaleString() : '-'
            }
          },
          {
            title: '备注',
            dataIndex: 'submit_note',
            width: 100,
            render: (_: any, row: SystemReleaseItem) => (
              row.submit_note ? (
                <Button size="small" onClick={() => openNote(`版本备注 - ${row.app_name} ${row.version}`, row.submit_note)}>详情</Button>
              ) : (
                <Text type="secondary">无</Text>
              )
            )
          },
          {
            title: '操作',
            width: 160,
            fixed: isMobile ? undefined : 'right',
            render: (_: any, row: SystemReleaseItem) => (
              row.status === 'in_review' ? (
                <Space size={[6, 6]} wrap>
                  <Button size="small" type="primary" onClick={() => approveRelease(row.id)}>{isMobile ? '通过' : '审批'}</Button>
                  <Button size="small" danger onClick={() => rejectRelease(row.id)}>{isMobile ? '拒绝' : '驳回'}</Button>
                </Space>
              ) : (
                <Text type="secondary">-</Text>
              )
            )
          }
        ]}
      />
    </>
  )

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>审核中心</Title>
        <Text type="secondary">审核企业注册与个人应用/版本申请</Text>
      </Space>

      {view ? (
        view === 'orgs' ? (
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>{orgTable}</Card>
        ) : (
          <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
            <Tabs
              defaultActiveKey="apps"
              size={isMobile ? 'small' : 'middle'}
              tabBarGutter={isMobile ? 12 : 24}
              tabBarStyle={{ overflowX: 'auto' }}
              items={[
                { key: 'apps', label: '应用审核', children: appTable },
                { key: 'releases', label: '版本审核', children: releaseTable }
              ]}
            />
          </Card>
        )
      ) : (
        <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
          <Tabs
            defaultActiveKey="orgs"
            size={isMobile ? 'small' : 'middle'}
            tabBarGutter={isMobile ? 12 : 24}
            tabBarStyle={{ overflowX: 'auto' }}
            items={[
              {
                key: 'orgs',
                label: '企业审核',
                children: orgTable
              },
              {
                key: 'apps',
                label: '应用审核',
                children: (
                  <Tabs
                    defaultActiveKey="apps"
                    size={isMobile ? 'small' : 'middle'}
                    tabBarGutter={isMobile ? 12 : 24}
                    tabBarStyle={{ overflowX: 'auto' }}
                    items={[
                      { key: 'apps', label: '应用审核', children: appTable },
                      { key: 'releases', label: '版本审核', children: releaseTable }
                    ]}
                  />
                )
              }
            ]}
          />
        </Card>
      )}

      <Modal
        title={materialsOrg ? `企业材料 - ${materialsOrg.name}` : '企业材料'}
        open={materialsOpen}
        onCancel={() => { setMaterialsOpen(false); setMaterialsOrg(null); setMaterials([]) }}
        footer={null}
        width={isMobile ? 'calc(100vw - 24px)' : 720}
      >
        <Table
          rowKey={(row) => row.id}
          dataSource={materials}
          loading={materialsLoading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 760 } : undefined}
          pagination={{
            pageSize: 5,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile
          }}
          columns={[
            { title: '文件名', dataIndex: 'file_name', width: 240 },
            { title: '类型', dataIndex: 'content_type', width: 170 },
            { title: '大小', dataIndex: 'size', width: 100, render: (v: number) => formatSize(v) },
            { title: '上传时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
            {
              title: '下载',
              width: 80,
              render: (_: any, row: any) => row.download_url ? (
                <a href={row.download_url} target="_blank" rel="noreferrer">下载</a>
              ) : '-'
            }
          ]}
        />
      </Modal>

      <Modal
        title={rejectOrg ? `驳回企业 - ${rejectOrg.name}` : '驳回企业'}
        open={rejectOpen}
        onOk={submitReject}
        confirmLoading={rejecting}
        onCancel={() => { setRejectOpen(false); setRejectOrg(null); rejectForm.resetFields() }}
        okText="确认驳回"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={rejectForm} initialValues={{ allow_resubmit: true }}>
          <Form.Item name="reason" label="驳回理由(可选)">
            <Input.TextArea rows={4} placeholder="请输入驳回原因，申请人将看到此内容" />
          </Form.Item>
          <Form.Item name="allow_resubmit" label="允许二次提交" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={appRejectTarget ? `驳回应用 - ${appRejectTarget.name}` : '驳回应用'}
        open={appRejectOpen}
        onOk={submitAppReject}
        confirmLoading={appRejecting}
        onCancel={() => { setAppRejectOpen(false); setAppRejectTarget(null); appRejectForm.resetFields() }}
        okText="确认驳回"
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={appRejectForm}>
          <Form.Item name="reason" label="驳回理由(可选)">
            <Input.TextArea rows={4} placeholder="请输入驳回原因，申请人将看到此内容" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={noteTitle || '备注'}
        open={noteOpen}
        footer={null}
        onCancel={() => { setNoteOpen(false); setNoteTitle(''); setNoteContent('') }}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Text style={{ whiteSpace: 'pre-wrap' }}>{noteContent || '无'}</Text>
      </Modal>
    </div>
  )
}

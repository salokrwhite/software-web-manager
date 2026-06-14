import { useEffect, useState } from 'react'
import { Button, Card, Descriptions, Drawer, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

type OrgItem = {
  id: string
  name: string
}

type SystemAppItem = {
  id: string
  name: string
  slug: string
  org_id: string
  org_name: string
  org_status: string
  org_type: string
  owner_email?: string
  status: string
  submitted_at?: string | null
  rejection_reason?: string | null
  submit_note?: string
  created_at: string
  release_count: number
  member_count: number
  device_count: number
}

export default function SystemApps() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [orgId, setOrgId] = useState<string>('')
  const [keyword, setKeyword] = useState<string>('')
  const [items, setItems] = useState<SystemAppItem[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [detailApp, setDetailApp] = useState<SystemAppItem | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  const openDetail = (row: SystemAppItem) => {
    setDetailApp(row)
    setDetailOpen(true)
  }

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

  const loadApps = async () => {
    setLoading(true)
    try {
      const params: any = {}
      if (orgId) params.org_id = orgId
      if (keyword) params.q = keyword
      const res = await api.get('/api/system/apps', { params })
      setItems(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载应用失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
  }, [])

  useEffect(() => {
    loadApps()
  }, [orgId])

  const statusTag = (status: string) => {
    const s = (status || '').toLowerCase()
    if (s === 'active') return <Tag color="green">已通过</Tag>
    if (s === 'pending') return <Tag color="orange">待审核</Tag>
    if (s === 'disabled') return <Tag color="red">已禁用</Tag>
    return <Tag>{status}</Tag>
  }

  const appStatusTag = (status: string) => {
    const s = (status || 'active').toLowerCase()
    if (s === 'active') return <Tag color="green">已启用</Tag>
    if (s === 'pending') return <Tag color="orange">待审核</Tag>
    if (s === 'rejected') return <Tag color="red">已驳回</Tag>
    if (s === 'disabled') return <Tag color="red">已禁用</Tag>
    return <Tag>{status}</Tag>
  }

  const disableApp = (row: SystemAppItem) => {
    Modal.confirm({
      title: '确认禁用该应用？',
      content: '禁用后该应用的客户端将无法通过签名校验，更新检查、心跳等接口将被拒绝。可随时重新启用。',
      okText: '禁用',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post(`/api/system/apps/${row.id}/disable`)
          message.success('应用已禁用')
          loadApps()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '禁用失败')
        }
      }
    })
  }

  const enableApp = async (row: SystemAppItem) => {
    try {
      await api.post(`/api/system/apps/${row.id}/enable`)
      message.success('应用已启用')
      loadApps()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '启用失败')
    }
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中应用？',
      content: '删除后将清理应用及其关联数据，且不可恢复。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/apps/batch-delete', { ids: selectedRowKeys })
          message.success('删除成功')
          setSelectedRowKeys([])
          loadApps()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>全局应用管理</Title>
        <Text type="secondary">查看所有企业应用</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          style={{ width: '100%', justifyContent: 'space-between' }}
        >
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary">企业筛选：</Text>
            <Select
              value={orgId || 'all'}
              style={{ width: isMobile ? '100%' : 240 }}
              onChange={(value) => setOrgId(value === 'all' ? '' : value)}
            >
              <Option value="all">全部企业</Option>
              {orgs.map((org) => (
                <Option key={org.id} value={org.id}>{org.name}</Option>
              ))}
            </Select>
          </Space>
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Input.Search
              placeholder="搜索应用名称或 slug"
              allowClear
              onSearch={() => loadApps()}
              onChange={(e) => setKeyword(e.target.value)}
              style={{ width: isMobile ? '100%' : 260 }}
            />
            <Button onClick={loadApps} style={isMobile ? { width: '100%' } : undefined}>查询</Button>
          </Space>
        </Space>
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
          rowKey={(row) => row.id}
          dataSource={items}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 1330 } : { x: 1410 }}
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
            { title: '应用名称', dataIndex: 'name', width: 180 },
            { title: 'Slug', dataIndex: 'slug', width: 160 },
            {
              title: '企业',
              dataIndex: 'org_name',
              width: 180,
              render: (_: string, row: SystemAppItem) => (
                row.org_type?.toLowerCase() === 'personal' ? '-' : (row.org_name || '-')
              )
            },
            {
              title: '企业状态',
              dataIndex: 'org_status',
              width: 120,
              render: (_: string, row: SystemAppItem) => (
                row.org_type?.toLowerCase() === 'personal' ? '-' : statusTag(row.org_status)
              )
            },
            {
              title: '应用状态',
              dataIndex: 'status',
              width: 110,
              render: (_: string, row: SystemAppItem) => appStatusTag(row.status)
            },
            { title: '版本数', dataIndex: 'release_count', width: 90 },
            { title: '成员数', dataIndex: 'member_count', width: 90 },
            { title: '设备数', dataIndex: 'device_count', width: 90 },
            { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
            {
              title: '操作',
              width: 200,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: SystemAppItem) => {
                const appStatus = (row.status || 'active').toLowerCase()
                return (
                  <Space size={[6, 6]} wrap>
                    <Button size="small" onClick={() => openDetail(row)}>{isMobile ? '详情' : '查看详情'}</Button>
                    {appStatus === 'disabled' ? (
                      <Button size="small" onClick={() => enableApp(row)}>启用</Button>
                    ) : appStatus === 'active' ? (
                      <Button size="small" danger onClick={() => disableApp(row)}>禁用</Button>
                    ) : null}
                  </Space>
                )
              }
            }
          ]}
        />
      </Card>

      <Drawer
        title="应用详情"
        placement="right"
        width={isMobile ? '100%' : 460}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
      >
        {detailApp && (
          <Descriptions column={1} bordered size="small" labelStyle={{ width: 96 }}>
            <Descriptions.Item label="应用名称">{detailApp.name}</Descriptions.Item>
            <Descriptions.Item label="Slug">{detailApp.slug}</Descriptions.Item>
            <Descriptions.Item label="应用状态">{appStatusTag(detailApp.status)}</Descriptions.Item>
            <Descriptions.Item label="企业">
              {detailApp.org_type?.toLowerCase() === 'personal' ? '个人' : (detailApp.org_name || '-')}
            </Descriptions.Item>
            <Descriptions.Item label="企业状态">
              {detailApp.org_type?.toLowerCase() === 'personal' ? '-' : statusTag(detailApp.org_status)}
            </Descriptions.Item>
            <Descriptions.Item label="负责人">{detailApp.owner_email || '-'}</Descriptions.Item>
            <Descriptions.Item label="版本数">{detailApp.release_count}</Descriptions.Item>
            <Descriptions.Item label="成员数">{detailApp.member_count}</Descriptions.Item>
            <Descriptions.Item label="设备数">{detailApp.device_count}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {detailApp.created_at ? new Date(detailApp.created_at).toLocaleString() : '-'}
            </Descriptions.Item>
            {detailApp.submitted_at && (
              <Descriptions.Item label="提交时间">{new Date(detailApp.submitted_at).toLocaleString()}</Descriptions.Item>
            )}
            {detailApp.submit_note && (
              <Descriptions.Item label="提交备注">{detailApp.submit_note}</Descriptions.Item>
            )}
            {detailApp.rejection_reason && (
              <Descriptions.Item label="驳回原因">
                <Text type="danger">{detailApp.rejection_reason}</Text>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}

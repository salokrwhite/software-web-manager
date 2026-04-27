import { useEffect, useState } from 'react'
import { Button, Card, Grid, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import api, { storeTokens } from '../api/client'

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
  created_at: string
  release_count: number
  member_count: number
  device_count: number
}

export default function SystemApps() {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [orgId, setOrgId] = useState<string>('')
  const [keyword, setKeyword] = useState<string>('')
  const [items, setItems] = useState<SystemAppItem[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

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
    if (s === 'active') return <Tag color="green">active</Tag>
    if (s === 'pending') return <Tag color="orange">pending</Tag>
    if (s === 'disabled') return <Tag color="red">disabled</Tag>
    return <Tag>{status}</Tag>
  }

  const impersonateOrg = async (orgIdValue: string, appId?: string) => {
    try {
      const res = await api.post('/api/system/impersonate', { org_id: orgIdValue, role: 'owner' })
      sessionStorage.setItem('system_backup_access_token', sessionStorage.getItem('access_token') || '')
      sessionStorage.setItem('system_backup_refresh_token', sessionStorage.getItem('refresh_token') || '')
      sessionStorage.setItem('system_backup_access_token_expires_at', sessionStorage.getItem('access_token_expires_at') || '')
      sessionStorage.setItem('system_backup_org_id', sessionStorage.getItem('org_id') || '')
      sessionStorage.setItem('system_backup_role', sessionStorage.getItem('role') || '')
      sessionStorage.setItem('impersonating', 'true')
      sessionStorage.setItem('impersonation_org_id', orgIdValue)
      storeTokens(res.data.tokens)
      if (res.data.org_id) {
        sessionStorage.setItem('org_id', res.data.org_id)
      }
      if (res.data.role) {
        sessionStorage.setItem('role', res.data.role)
      }
      message.success('已进入企业后台')
      if (appId) {
        navigate(`/apps/${appId}`)
      } else {
        navigate('/apps')
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '冒充失败')
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
        <Text type="secondary">查看所有企业应用，操作需冒充进入企业后台</Text>
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
          scroll={isMobile ? { x: 1240 } : { x: 1320 }}
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
            { title: '版本数', dataIndex: 'release_count', width: 90 },
            { title: '成员数', dataIndex: 'member_count', width: 90 },
            { title: '设备数', dataIndex: 'device_count', width: 90 },
            { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString() : '-' },
            {
              title: '操作',
              width: 220,
              fixed: isMobile ? undefined : 'right',
              render: (_: any, row: SystemAppItem) => (
                <Space size={[6, 6]} wrap>
                  <Button size="small" onClick={() => impersonateOrg(row.org_id, row.id)}>{isMobile ? '详情' : '查看详情'}</Button>
                  <Button size="small" onClick={() => impersonateOrg(row.org_id)}>{isMobile ? '进入' : '进入企业后台'}</Button>
                </Space>
              )
            }
          ]}
        />
      </Card>
    </div>
  )
}

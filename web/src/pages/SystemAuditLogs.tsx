import { Button, Card, DatePicker, Form, Grid, Input, Modal, Select, Space, Table, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import api from '../api/client'
import dayjs from 'dayjs'
import {
  formatAction,
  formatTargetType,
  normalizeAuditActionQuery,
  normalizeAuditTargetTypeQuery
} from '../utils/audit'

const { Title, Text } = Typography
const { RangePicker } = DatePicker
const { Option } = Select

type OrgItem = {
  id: string
  name: string
}

export default function SystemAuditLogs() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [logs, setLogs] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [selectedRows, setSelectedRows] = useState<any[]>([])
  const [pageSize, setPageSize] = useState(10)
  const [queryParams, setQueryParams] = useState<any>({})
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

  const loadLogs = async (params?: any) => {
    setLoading(true)
    try {
      const res = await api.get('/api/system/audit-logs', { params })
      setLogs(res.data.items || [])
      setSelectedRowKeys([])
      setSelectedRows([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载审计日志失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
    loadLogs()
  }, [])

  const onSearch = async () => {
    const values = await form.validateFields()
    const params: any = {
      org_id: values.org_id === 'all' ? undefined : values.org_id,
      action: normalizeAuditActionQuery(values.action),
      target_type: normalizeAuditTargetTypeQuery(values.target_type)
    }
    if (values.range?.length === 2) {
      params.from = values.range[0].format('YYYY-MM-DD')
      params.to = values.range[1].format('YYYY-MM-DD')
    }
    setQueryParams(params)
    loadLogs(params)
  }

  const hasSelection = selectedRowKeys.length > 0

  const toCsvValue = (value: any) => {
    const text = value === null || value === undefined ? '' : String(value)
    const escaped = text.replace(/"/g, '""')
    if (/[",\n]/.test(escaped)) {
      return `"${escaped}"`
    }
    return escaped
  }

  const handleExportCsv = () => {
    if (!hasSelection) {
      message.warning('请先选择日志')
      return
    }
    const headers = ['时间', '企业', '用户', '动作', '目标类型', '目标ID', 'IP']
    const rows = selectedRows.map((log) => [
      log.created_at ? new Date(log.created_at).toLocaleString() : '',
      log.org_name || '',
      log.user_email || '',
      formatAction(log.action),
      formatTargetType(log.target_type),
      log.target_id || '',
      log.ip_address || ''
    ])
    const csv = [headers, ...rows].map((row) => row.map(toCsvValue).join(',')).join('\r\n')
    const csvWithBom = `\uFEFF${csv}`
    const blob = new Blob([csvWithBom], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `system-audit-logs-${dayjs().format('YYYYMMDD-HHmmss')}.csv`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
    message.success('已导出 CSV')
  }

  const handleDelete = () => {
    if (!hasSelection) {
      message.warning('请先选择日志')
      return
    }
    Modal.confirm({
      title: '确认删除选中的日志？',
      content: `将删除 ${selectedRowKeys.length} 条系统日志，此操作不可恢复。`,
      okText: '确认',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/system/audit-logs/delete', { ids: selectedRowKeys })
          message.success('删除成功')
          setSelectedRowKeys([])
          setSelectedRows([])
          await loadLogs(queryParams)
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>系统审计日志</Title>
        <Text type="secondary">查看全局关键操作的审计记录</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Form
          layout={isMobile ? 'vertical' : 'inline'}
          form={form}
          initialValues={{ org_id: 'all', range: [dayjs().subtract(7, 'day'), dayjs()] }}
          style={isMobile ? { width: '100%' } : undefined}
        >
          <Form.Item name="org_id">
            <Select style={{ width: isMobile ? '100%' : 220 }}>
              <Option value="all">全部企业</Option>
              {orgs.map((org) => (
                <Option key={org.id} value={org.id}>{org.name}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="action">
            <Input placeholder="动作 (如 system.org.approve / 系统审核通过企业)" style={{ width: isMobile ? '100%' : 220 }} />
          </Form.Item>
          <Form.Item name="target_type">
            <Input placeholder="目标类型 (如 org / 企业)" style={{ width: isMobile ? '100%' : 180 }} />
          </Form.Item>
          <Form.Item name="range">
            <RangePicker style={{ width: isMobile ? '100%' : undefined }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={onSearch} style={isMobile ? { width: '100%' } : undefined}>查询</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Space direction={isMobile ? 'vertical' : 'horizontal'} style={{ marginBottom: 12, width: isMobile ? '100%' : undefined }}>
          <Button danger disabled={!hasSelection || loading} onClick={handleDelete} style={isMobile ? { width: '100%' } : undefined}>批量删除</Button>
          <Button disabled={!hasSelection || loading} onClick={handleExportCsv} style={isMobile ? { width: '100%' } : undefined}>导出 CSV</Button>
          <Text type="secondary">已选 {selectedRowKeys.length} 条</Text>
        </Space>
        <Table
          rowKey="id"
          dataSource={logs}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 980 } : { x: 1060 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys, rows) => {
              setSelectedRowKeys(keys as string[])
              setSelectedRows(rows)
            }
          }}
          pagination={{
            pageSize,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50'],
            onShowSizeChange: (_, size) => setPageSize(size)
          }}
          columns={[
            { title: '时间', dataIndex: 'created_at', width: 180, render: (d: string) => d ? new Date(d).toLocaleString() : '-' },
            { title: '企业', dataIndex: 'org_name', width: 180 },
            { title: '用户', dataIndex: 'user_email', width: 220 },
            { title: '动作', dataIndex: 'action', width: 210, render: (v: string) => formatAction(v) },
            { title: '目标类型', dataIndex: 'target_type', width: 140, render: (v: string) => formatTargetType(v) },
            { title: 'IP', dataIndex: 'ip_address', width: 150 }
          ]}
        />
      </Card>
    </div>
  )
}

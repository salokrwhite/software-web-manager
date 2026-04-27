import { Button, Card, DatePicker, Form, Grid, Input, Modal, Space, Table, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import type { Key } from 'react'
import api from '../api/client'
import dayjs from 'dayjs'
import {
  formatAction,
  formatTargetId,
  formatTargetType,
  normalizeAuditActionQuery,
  normalizeAuditTargetTypeQuery
} from '../utils/audit'

const { Title, Text } = Typography
const { RangePicker } = DatePicker

export default function AuditLogs() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [logs, setLogs] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [pageSize, setPageSize] = useState(10)
  const [queryParams, setQueryParams] = useState<any>({})
  const [form] = Form.useForm()

  const loadLogs = async (params?: any) => {
    setLoading(true)
    try {
      const res = await api.get('/api/audit-logs', { params })
      setLogs(res.data.items || [])
      setSelectedRowKeys([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载审计日志失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadLogs()
  }, [])

  const onSearch = async () => {
    const values = await form.validateFields()
    const params: any = {
      action: normalizeAuditActionQuery(values.action),
      target_type: normalizeAuditTargetTypeQuery(values.target_type),
      target_id: values.target_id
    }
    if (values.range?.length === 2) {
      params.from = values.range[0].format('YYYY-MM-DD')
      params.to = values.range[1].format('YYYY-MM-DD')
    }
    setQueryParams(params)
    loadLogs(params)
  }

  const hasSelection = selectedRowKeys.length > 0

  const handleDelete = () => {
    if (!hasSelection) {
      message.warning('请先选择日志')
      return
    }
    Modal.confirm({
      title: '确认删除选中的日志？',
      content: `将删除 ${selectedRowKeys.length} 条审计日志，此操作不可恢复。`,
      okText: '确认',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.post('/api/audit-logs/delete', { ids: selectedRowKeys.map((key) => String(key)) })
          message.success('删除成功')
          setSelectedRowKeys([])
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
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>审计日志</Title>
        <Text type="secondary">查看关键操作的审计记录</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Form
          layout={isMobile ? 'vertical' : 'inline'}
          form={form}
          initialValues={{ range: [dayjs().subtract(7, 'day'), dayjs()] }}
          style={isMobile ? { width: '100%' } : undefined}
        >
          <Form.Item name="action">
            <Input placeholder="动作 (如 release.publish / 发布版本到渠道)" style={{ width: isMobile ? '100%' : 220 }} />
          </Form.Item>
          <Form.Item name="target_type">
            <Input placeholder="目标类型 (如 release / 版本)" style={{ width: isMobile ? '100%' : 180 }} />
          </Form.Item>
          <Form.Item name="target_id">
            <Input placeholder="目标 ID" style={{ width: isMobile ? '100%' : 220 }} />
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
          <Text type="secondary">已选 {selectedRowKeys.length} 条</Text>
        </Space>
        <Table
          rowKey={(row) => row.id || row.ID}
          dataSource={logs}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          scroll={isMobile ? { x: 980 } : { x: 1100 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys)
          }}
          pagination={{
            pageSize,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50', '100'],
            onShowSizeChange: (_, size) => setPageSize(size),
            showTotal: (total) => `共 ${total} 条`
          }}
          columns={[
            {
              title: '时间',
              dataIndex: 'created_at',
              width: 180,
              render: (_: any, record: any) => {
                const value = record.created_at || record.CreatedAt
                return value ? new Date(value).toLocaleString() : '-'
              }
            },
            {
              title: '动作',
              dataIndex: 'action',
              width: 220,
              render: (_: any, record: any) => formatAction(record.action || record.Action)
            },
            {
              title: '目标类型',
              dataIndex: 'target_type',
              width: 140,
              render: (_: any, record: any) => formatTargetType(record.target_type || record.TargetType)
            },
            {
              title: '目标ID',
              dataIndex: 'target_id',
              width: 220,
              render: (_: any, record: any) => formatTargetId(record.target_id || record.TargetID)
            },
            {
              title: '用户',
              dataIndex: 'user_email',
              width: 220,
              render: (_: any, record: any) => record.user_email || record.UserEmail || record.user_id || record.UserID || '-'
            },
            {
              title: 'IP',
              dataIndex: 'ip_address',
              width: 150,
              render: (_: any, record: any) => record.ip_address || record.IPAddress || '-'
            }
          ]}
        />
      </Card>
    </div>
  )
}

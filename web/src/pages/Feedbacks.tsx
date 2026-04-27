import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Grid,
  Image,
  Input,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  message
} from 'antd'
import { ReloadOutlined, SaveOutlined, SearchOutlined } from '@ant-design/icons'
import { useEffect, useMemo, useState } from 'react'
import dayjs from 'dayjs'
import api from '../api/client'

const { Text, Title, Paragraph } = Typography
const { RangePicker } = DatePicker
const { TextArea } = Input

type AppItem = {
  id: string
  name: string
  slug?: string
  feedback_enabled?: boolean
}

type FeedbackStatus = 'open' | 'processing' | 'resolved' | 'closed'

type FeedbackItem = {
  id: string
  content: string
  rating?: number
  contact?: string
  device_id?: string
  app_version?: string
  channel_code?: string
  status: FeedbackStatus
  internal_note?: string
  handled_by?: string
  handled_at?: string
  created_at: string
  updated_at?: string
  attachment_count?: number
}

type FeedbackAttachment = {
  id: string
  file_name: string
  content_type: string
  size: number
  download_url: string
  created_at: string
}

type FeedbackDetail = FeedbackItem & {
  metadata_json?: any
  Metadata?: any
}

const statusOptions: Array<{ value: FeedbackStatus; label: string; color: string }> = [
  { value: 'open', label: '未处理', color: 'red' },
  { value: 'processing', label: '处理中', color: 'blue' },
  { value: 'resolved', label: '已解决', color: 'green' },
  { value: 'closed', label: '已关闭', color: 'default' }
]

const statusMap = Object.fromEntries(statusOptions.map((item) => [item.value, item]))

function pick(raw: any, snake: string, pascal: string, fallback?: any) {
  return raw?.[snake] ?? raw?.[pascal] ?? fallback
}

function normalizeFeedback(raw: any): FeedbackItem {
  const handledBy = pick(raw, 'handled_by', 'HandledBy')
  return {
    id: String(pick(raw, 'id', 'ID', '')),
    content: pick(raw, 'content', 'Content', ''),
    rating: pick(raw, 'rating', 'Rating'),
    contact: pick(raw, 'contact', 'Contact', ''),
    device_id: pick(raw, 'device_id', 'DeviceID', ''),
    app_version: pick(raw, 'app_version', 'AppVersion', ''),
    channel_code: pick(raw, 'channel_code', 'ChannelCode', ''),
    status: pick(raw, 'status', 'Status', 'open'),
    internal_note: pick(raw, 'internal_note', 'InternalNote', ''),
    handled_by: handledBy ? String(handledBy) : '',
    handled_at: pick(raw, 'handled_at', 'HandledAt'),
    created_at: pick(raw, 'created_at', 'CreatedAt', ''),
    updated_at: pick(raw, 'updated_at', 'UpdatedAt', ''),
    attachment_count: Number(pick(raw, 'attachment_count', 'AttachmentCount', 0))
  }
}

function normalizeDetail(raw: any): FeedbackDetail {
  return {
    ...normalizeFeedback(raw),
    metadata_json: pick(raw, 'metadata_json', 'Metadata')
  }
}

function formatTime(value?: string) {
  return value ? new Date(value).toLocaleString() : '-'
}

function formatBytes(size?: number) {
  if (!size) return '0 B'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

export default function Feedbacks() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [apps, setApps] = useState<AppItem[]>([])
  const [appsLoading, setAppsLoading] = useState(false)
  const [selectedAppId, setSelectedAppId] = useState<string>('')
  const [items, setItems] = useState<FeedbackItem[]>([])
  const [total, setTotal] = useState(0)
  const [ready, setReady] = useState(true)
  const [statusCounts, setStatusCounts] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [rating, setRating] = useState<number | undefined>(undefined)
  const [status, setStatus] = useState<FeedbackStatus | undefined>(undefined)
  const [hasAttachment, setHasAttachment] = useState<boolean | undefined>(undefined)
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().add(-30, 'day'),
    dayjs()
  ])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [detail, setDetail] = useState<FeedbackDetail | null>(null)
  const [detailStatus, setDetailStatus] = useState<FeedbackStatus>('open')
  const [detailNote, setDetailNote] = useState('')
  const [attachments, setAttachments] = useState<FeedbackAttachment[]>([])

  const selectedApp = useMemo(() => apps.find((app) => app.id === selectedAppId), [apps, selectedAppId])
  const feedbackEnabled = selectedApp?.feedback_enabled !== false

  const loadApps = async () => {
    setAppsLoading(true)
    try {
      const res = await api.get('/api/apps')
      const list = (res.data.items || []).map((item: any) => ({
        id: item.ID || item.id,
        name: item.Name || item.name,
        slug: item.Slug || item.slug,
        feedback_enabled: item.FeedbackEnabled ?? item.feedback_enabled ?? true
      }))
      setApps(list)
      if (!selectedAppId && list.length > 0) {
        setSelectedAppId(list[0].id)
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载应用失败')
    } finally {
      setAppsLoading(false)
    }
  }

  const loadFeedback = async (targetAppId = selectedAppId, nextPage = page, nextSize = pageSize) => {
    if (!targetAppId) return
    setLoading(true)
    try {
      const params: any = {
        page: nextPage,
        page_size: nextSize
      }
      if (keyword.trim()) params.keyword = keyword.trim()
      if (typeof rating === 'number') params.rating = rating
      if (status) params.status = status
      if (typeof hasAttachment === 'boolean') params.has_attachment = hasAttachment
      if (dateRange?.[0] && dateRange?.[1]) {
        params.from = dateRange[0].format('YYYY-MM-DD')
        params.to = dateRange[1].format('YYYY-MM-DD')
      }
      const res = await api.get(`/api/apps/${targetAppId}/feedback`, { params })
      setReady(res.data.ready !== false)
      setItems((res.data.items || []).map(normalizeFeedback))
      setTotal(res.data.total || 0)
      setStatusCounts(res.data.status_counts || {})
      setPage(nextPage)
      setPageSize(nextSize)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载反馈失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadApps()
  }, [])

  useEffect(() => {
    if (!selectedAppId) return
    loadFeedback(selectedAppId, 1, pageSize)
  }, [selectedAppId])

  const openDetail = async (record: FeedbackItem) => {
    if (!selectedAppId) return
    setDetailOpen(true)
    setDetailLoading(true)
    try {
      const res = await api.get(`/api/apps/${selectedAppId}/feedback/${record.id}`)
      const nextDetail = normalizeDetail(res.data.feedback || {})
      setDetail(nextDetail)
      setDetailStatus(nextDetail.status || 'open')
      setDetailNote(nextDetail.internal_note || '')
      setAttachments(res.data.attachments || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载反馈详情失败')
    } finally {
      setDetailLoading(false)
    }
  }

  const saveDetail = async () => {
    if (!selectedAppId || !detail) return
    setSaving(true)
    try {
      const res = await api.patch(`/api/apps/${selectedAppId}/feedback/${detail.id}`, {
        status: detailStatus,
        internal_note: detailNote
      })
      const nextDetail = normalizeDetail(res.data.feedback || {})
      setDetail((prev) => ({ ...(prev || nextDetail), ...nextDetail }))
      message.success('反馈处理信息已保存')
      loadFeedback(selectedAppId, page, pageSize)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const isImageAttachment = (attachment: FeedbackAttachment) => {
    const type = (attachment.content_type || '').toLowerCase()
    if (type.startsWith('image/')) return true
    const name = (attachment.file_name || '').toLowerCase()
    return /\.(png|jpe?g|gif|webp|bmp|svg)$/.test(name)
  }

  const metadataText = useMemo(() => {
    const metadata = detail?.metadata_json ?? detail?.Metadata
    if (!metadata) return ''
    if (typeof metadata === 'string') {
      try {
        return JSON.stringify(JSON.parse(metadata), null, 2)
      } catch {
        return metadata
      }
    }
    return JSON.stringify(metadata, null, 2)
  }, [detail])

  const columns = useMemo(() => ([
    {
      title: '时间',
      dataIndex: 'created_at',
      width: isMobile ? 180 : 160,
      render: (v: string) => formatTime(v)
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: FeedbackStatus) => {
        const meta = statusMap[v] || statusMap.open
        return <Tag color={meta.color}>{meta.label}</Tag>
      }
    },
    {
      title: '评分',
      dataIndex: 'rating',
      width: 80,
      render: (v: number) => (typeof v === 'number' ? <Tag color="gold">{v} 分</Tag> : '-')
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true
    },
    {
      title: '版本',
      dataIndex: 'app_version',
      width: 120,
      render: (v: string) => v || '-'
    },
    {
      title: '渠道',
      dataIndex: 'channel_code',
      width: 120,
      render: (v: string) => v || '-'
    },
    {
      title: '设备',
      dataIndex: 'device_id',
      width: 160,
      ellipsis: true
    },
    {
      title: '附件',
      dataIndex: 'attachment_count',
      width: 80,
      render: (v: number) => (v ? <Tag color="cyan">{v}</Tag> : 0)
    },
    {
      title: '处理',
      dataIndex: 'handled_at',
      width: 160,
      render: (v: string) => formatTime(v)
    },
    {
      title: '操作',
      width: 90,
      render: (_: any, record: FeedbackItem) => (
        <Button size="small" onClick={() => openDetail(record)}>{isMobile ? '详情' : '查看'}</Button>
      )
    }
  ]), [isMobile])

  return (
    <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
      <Row justify="space-between" align={isMobile ? 'top' : 'middle'} gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={24} lg={12}>
          <Space direction="vertical" size={4}>
            <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>用户反馈</Title>
            <Text type="secondary">集中处理 SDK 上报的用户反馈、截图和上下文信息</Text>
          </Space>
        </Col>
        <Col xs={24} lg={12} style={{ textAlign: isMobile ? 'left' : 'right' }}>
          <Space wrap>
            <Select
              placeholder="选择应用"
              loading={appsLoading}
              value={selectedAppId || undefined}
              onChange={(value) => setSelectedAppId(value)}
              style={{ width: isMobile ? '100%' : 240 }}
              options={apps.map((app) => ({ label: app.name, value: app.id }))}
            />
            <Button icon={<ReloadOutlined />} onClick={() => loadFeedback(selectedAppId, page, pageSize)} loading={loading}>
              刷新
            </Button>
          </Space>
        </Col>
      </Row>

      {!selectedAppId ? (
        <Empty description="暂无应用，请先创建应用" />
      ) : (
        <>
          {!ready && (
            <Alert
              type="warning"
              showIcon
              message="反馈功能未初始化"
              description="请先执行最新数据库迁移，完成 feedbacks 闭环字段初始化后再使用此页面。"
              style={{ marginBottom: 16 }}
            />
          )}
          {!feedbackEnabled && (
            <Alert
              type="warning"
              showIcon
              message="当前应用的新反馈上报已关闭"
              description="历史反馈仍可查看和处理；SDK 继续上报时会收到 feedback_disabled 拒绝响应。"
              style={{ marginBottom: 16 }}
            />
          )}

          <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
            {statusOptions.map((item) => (
              <Col xs={12} md={6} key={item.value}>
                <Card size="small" bordered>
                  <Statistic title={item.label} value={statusCounts[item.value] || 0} />
                </Card>
              </Col>
            ))}
          </Row>

          <Row gutter={[12, 12]} style={{ marginBottom: 16 }} align="middle">
            <Col xs={24} md={8} lg={5}>
              <Input
                placeholder="关键词（内容/联系方式）"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onPressEnter={() => loadFeedback(selectedAppId, 1, pageSize)}
              />
            </Col>
            <Col xs={12} md={4} lg={3}>
              <Select
                allowClear
                placeholder="评分"
                value={rating}
                style={{ width: '100%' }}
                onChange={(value) => setRating(value)}
                options={[1, 2, 3, 4, 5].map((v) => ({ label: `${v} 分`, value: v }))}
              />
            </Col>
            <Col xs={12} md={5} lg={4}>
              <Select
                allowClear
                placeholder="状态"
                value={status}
                style={{ width: '100%' }}
                onChange={(value) => setStatus(value)}
                options={statusOptions.map(({ value, label }) => ({ value, label }))}
              />
            </Col>
            <Col xs={12} md={5} lg={4}>
              <Select
                allowClear
                placeholder="附件"
                value={hasAttachment}
                style={{ width: '100%' }}
                onChange={(value) => setHasAttachment(value)}
                options={[
                  { label: '有附件', value: true },
                  { label: '无附件', value: false }
                ]}
              />
            </Col>
            <Col xs={24} md={10} lg={6}>
              <RangePicker
                value={dateRange}
                onChange={(range) => {
                  if (range?.[0] && range?.[1]) setDateRange([range[0], range[1]])
                }}
                style={{ width: '100%' }}
              />
            </Col>
            <Col xs={24} lg={2}>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={() => loadFeedback(selectedAppId, 1, pageSize)}
                loading={loading}
                style={{ width: isMobile ? '100%' : undefined }}
              >
                查询
              </Button>
            </Col>
          </Row>

          <Table
            rowKey="id"
            loading={loading}
            size={isMobile ? 'small' : 'middle'}
            scroll={isMobile ? { x: 1280 } : { x: 1320 }}
            dataSource={items}
            columns={columns}
            pagination={{
              current: page,
              pageSize,
              total,
              size: isMobile ? 'small' : 'default',
              responsive: true,
              showSizeChanger: !isMobile,
              onChange: (nextPage, nextSize) => loadFeedback(selectedAppId, nextPage, nextSize || pageSize)
            }}
          />
        </>
      )}

      <Drawer
        title="反馈详情"
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={isMobile ? '100%' : 640}
      >
        {detailLoading ? (
          <Text type="secondary">加载中...</Text>
        ) : detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions size="small" column={1} bordered>
              <Descriptions.Item label="提交时间">{formatTime(detail.created_at)}</Descriptions.Item>
              <Descriptions.Item label="评分">{typeof detail.rating === 'number' ? `${detail.rating} 分` : '-'}</Descriptions.Item>
              <Descriptions.Item label="联系方式">{detail.contact || '-'}</Descriptions.Item>
              <Descriptions.Item label="设备ID">{detail.device_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="渠道/版本">{detail.channel_code || '-'} / {detail.app_version || '-'}</Descriptions.Item>
              <Descriptions.Item label="附件数量">{detail.attachment_count || 0}</Descriptions.Item>
            </Descriptions>

            <div>
              <Text type="secondary">反馈内容</Text>
              <Paragraph style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>{detail.content || '-'}</Paragraph>
            </div>

            <div>
              <Text type="secondary">处理状态</Text>
              <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
                <Select
                  value={detailStatus}
                  onChange={(value) => setDetailStatus(value)}
                  options={statusOptions.map(({ value, label }) => ({ value, label }))}
                  style={{ width: '100%' }}
                />
                <TextArea
                  value={detailNote}
                  onChange={(e) => setDetailNote(e.target.value)}
                  rows={4}
                  maxLength={5000}
                  showCount
                  placeholder="内部处理备注，仅后台可见"
                />
                <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={saveDetail}>
                  保存处理信息
                </Button>
              </Space>
            </div>

            <Divider style={{ margin: '4px 0' }} />

            <div>
              <Text type="secondary">Metadata</Text>
              {metadataText ? (
                <pre style={{ marginTop: 8, padding: 12, background: '#f6f8fa', borderRadius: 6, overflowX: 'auto' }}>{metadataText}</pre>
              ) : (
                <div style={{ marginTop: 8 }}>无 metadata</div>
              )}
            </div>

            <div>
              <Text type="secondary">附件</Text>
              {attachments.length === 0 ? (
                <div style={{ marginTop: 8 }}>无附件</div>
              ) : (
                <Image.PreviewGroup>
                  <Space wrap style={{ marginTop: 8 }}>
                    {attachments.map((file) => (
                      isImageAttachment(file) && file.download_url ? (
                        <Image
                          key={file.id}
                          width={isMobile ? 96 : 132}
                          src={file.download_url}
                          alt={file.file_name}
                          style={{ borderRadius: 6 }}
                        />
                      ) : (
                        <Button key={file.id} href={file.download_url} target="_blank">
                          {file.file_name} ({formatBytes(file.size)})
                        </Button>
                      )
                    ))}
                  </Space>
                </Image.PreviewGroup>
              )}
            </div>
          </Space>
        ) : (
          <Text type="secondary">暂无详情</Text>
        )}
      </Drawer>
    </Card>
  )
}

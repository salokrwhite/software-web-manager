import { Button, Card, Divider, Grid, Image, Input, List, Modal, Space, Steps, Tag, Typography, Upload, message } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { CheckCircleOutlined, ClockCircleOutlined, SyncOutlined, UploadOutlined } from '@ant-design/icons'
import api from '../api/client'
import ticketWS, { WsStatus } from '../utils/ws'
import { formatTicketStatus, getTicketStepIndex, getTicketStepsStatus, TICKET_STEPS } from '../utils/ticket'

const { Title, Text, Paragraph } = Typography

type TicketAttachment = {
  id: string
  file_name: string
  content_type: string
  size: number
  download_url: string
  created_at: string
}

type TicketDetail = {
  id: string
  org_id: string
  title: string
  description: string
  status: string
  assignee_type: string
  assignee_user_id?: string | null
  assignee_email?: string
  created_by: string
  created_by_email?: string
  in_progress_at?: string | null
  resolved_at?: string | null
  created_at: string
  updated_at: string
  attachments: TicketAttachment[]
}

type TicketMessage = {
  id: string
  ticket_id: string
  sender_type: string
  user_id: string
  user_email?: string
  content: string
  created_at: string
  attachments: TicketAttachment[]
}

const formatDateTime = (value?: string | null) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const formatStepDateTime = (value?: string | null) => {
  if (!value) return '-'
  const dt = new Date(value)
  if (Number.isNaN(dt.getTime())) return String(value)
  const pad = (num: number) => String(num).padStart(2, '0')
  return `${dt.getFullYear()}/${pad(dt.getMonth() + 1)}/${pad(dt.getDate())} ${pad(dt.getHours())}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}`
}

export default function TicketDetail() {
  const navigate = useNavigate()
  const { id } = useParams()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [detail, setDetail] = useState<TicketDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [messages, setMessages] = useState<TicketMessage[]>([])
  const [messageLoading, setMessageLoading] = useState(false)
  const [messageContent, setMessageContent] = useState('')
  const [messageFiles, setMessageFiles] = useState<any[]>([])
  const [sending, setSending] = useState(false)
  const [wsStatus, setWsStatus] = useState<WsStatus>('idle')
  const role = (sessionStorage.getItem('role') || '').toLowerCase()
  const canDelete = role === 'admin' || role === 'owner'

  const goBack = () => {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      navigate(-1)
      return
    }
    navigate('/tickets')
  }

  const loadDetail = async () => {
    if (!id) return
    setLoading(true)
    try {
      const res = await api.get(`/api/tickets/${id}`)
      setDetail(res.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载工单详情失败')
    } finally {
      setLoading(false)
    }
  }

  const loadMessages = async () => {
    if (!id) return
    setMessageLoading(true)
    try {
      const res = await api.get(`/api/tickets/${id}/messages`)
      setMessages(res.data.items || [])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载对话失败')
    } finally {
      setMessageLoading(false)
    }
  }

  useEffect(() => {
    loadDetail()
    loadMessages()
  }, [id])

  useEffect(() => {
    if (!id) return
    const offEvent = ticketWS.onEvent((event) => {
      if (!event || event.ticket_id !== id) return
      if (event.type === 'ticket.message.created' && event.payload) {
        const msg = event.payload as TicketMessage
        setMessages((prev) => (prev.some((item) => item.id === msg.id) ? prev : [...prev, msg]))
      }
      if (event.type === 'ticket.status.updated' && event.payload) {
        const payload = event.payload as Partial<TicketDetail>
        setDetail((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            status: payload.status || prev.status,
            in_progress_at: payload.in_progress_at ?? prev.in_progress_at,
            resolved_at: payload.resolved_at ?? prev.resolved_at,
            updated_at: payload.updated_at || prev.updated_at
          }
        })
      }
    })
    const offStatus = ticketWS.onStatus((status) => setWsStatus(status))
    ticketWS.subscribe(id)
    return () => {
      ticketWS.unsubscribe(id)
      offEvent()
      offStatus()
    }
  }, [id])

  const wrapNoBreak = (text: string) => (
    <span style={isMobile ? undefined : { display: 'inline-flex' }}>{text}</span>
  )

  const renderInfoItem = (label: string, value?: string) => (
    <div style={{ marginBottom: isMobile ? 10 : 12 }}>
      <Text type="secondary" style={{ display: 'block' }}>{label}</Text>
      <Text>{value || '-'}</Text>
    </div>
  )

  const renderStatusTag = (status: string) => {
    const normalized = (status || '').toLowerCase()
    if (normalized === 'resolved') {
      return (
        <Tag color="success" style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <CheckCircleOutlined />
          {formatTicketStatus(status)}
        </Tag>
      )
    }
    if (normalized === 'in_progress') {
      return (
        <Tag color="processing" style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <SyncOutlined spin />
          {formatTicketStatus(status)}
        </Tag>
      )
    }
    return (
      <Tag color="default" style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        <ClockCircleOutlined />
        {formatTicketStatus(status)}
      </Tag>
    )
  }

  const stepItems = (detailInfo: TicketDetail | null) => {
    const createdAt = formatStepDateTime(detailInfo?.created_at)
    const progressAt = formatStepDateTime(detailInfo?.in_progress_at)
    const resolvedAt = formatStepDateTime(detailInfo?.resolved_at)
    return TICKET_STEPS.map((step) => {
      if (step.key === 'submitted') {
        return { title: step.title, description: detailInfo?.created_at ? wrapNoBreak(`提交于 ${createdAt}`) : '等待提交' }
      }
      if (step.key === 'in_progress') {
        return { title: step.title, description: detailInfo?.in_progress_at ? wrapNoBreak(`开始于 ${progressAt}`) : '等待处理' }
      }
      return { title: step.title, description: detailInfo?.resolved_at ? wrapNoBreak(`完成于 ${resolvedAt}`) : '等待完成' }
    })
  }

  const isImageAttachment = (attachment: TicketAttachment) => {
    const type = (attachment.content_type || '').toLowerCase()
    if (type.startsWith('image/')) return true
    const name = (attachment.file_name || '').toLowerCase()
    return /\.(png|jpe?g|gif|webp|bmp|svg)$/.test(name)
  }

  const getSenderLabel = (msg: TicketMessage) => {
    if ((msg.sender_type || '').toLowerCase() === 'system') return '系统管理员'
    return msg.user_email || '企业成员'
  }

  const beforeUpload = (file: any) => {
    if (file.size > 20 * 1024 * 1024) {
      message.error('附件大小不能超过 20MB')
      return Upload.LIST_IGNORE
    }
    if (messageFiles.length >= 5) {
      message.error('最多上传 5 个附件')
      return Upload.LIST_IGNORE
    }
    return false
  }

  const sendMessage = async () => {
    if (!id) return
    const trimmed = messageContent.trim()
    if (!trimmed && messageFiles.length === 0) {
      message.warning('请输入内容或上传附件')
      return
    }
    const oversized = messageFiles.find((file) => (file.size || file.originFileObj?.size || 0) > 20 * 1024 * 1024)
    if (oversized) {
      message.error('附件大小不能超过 20MB')
      return
    }
    const formData = new FormData()
    if (trimmed) {
      formData.append('content', trimmed)
    }
    messageFiles.forEach((file) => {
      const raw = file.originFileObj || file
      formData.append('attachments', raw)
    })
    setSending(true)
    try {
      await api.post(`/api/tickets/${id}/messages`, formData)
      setMessageContent('')
      setMessageFiles([])
      await loadMessages()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '发送失败')
    } finally {
      setSending(false)
    }
  }

  const isResolved = detail?.status?.toLowerCase() === 'resolved'
  const handleDelete = () => {
    if (!detail) return
    if (!canDelete) {
      message.error('当前账号无权限删除工单')
      return
    }
    Modal.confirm({
      title: '确认删除该工单？',
      content: '删除后将清理工单、附件及聊天记录，且不可恢复。',
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        await api.delete(`/api/tickets/${detail.id}`)
        message.success('工单已删除')
        navigate('/tickets')
      }
    })
  }

  const handleClose = () => {
    if (!detail) return
    Modal.confirm({
      title: '确认关闭该工单？',
      content: '关闭后将标记为已解决，如需继续处理请重新提交工单。',
      okText: '关闭工单',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.patch(`/api/tickets/${detail.id}/close`)
          message.success('工单已关闭')
          await loadDetail()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '关闭工单失败')
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>工单详情</Title>
        <Text type="secondary">查看工单的详细信息与附件</Text>
      </Space>

      {loading ? (
        <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
          <Text type="secondary">加载中...</Text>
        </Card>
      ) : detail ? (
        <div
          style={{
            display: 'flex',
            flexDirection: isMobile ? 'column' : 'row',
            gap: isMobile ? 12 : 16,
            alignItems: 'flex-start',
            flexWrap: 'wrap'
          }}
        >
          <Card style={{ flex: 1, minWidth: 0, width: '100%', borderRadius: isMobile ? 10 : 12 }}>
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <Button
                  type="text"
                  onClick={goBack}
                  icon={(
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M15 6L9 12L15 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  )}
                />
                <Title
                  level={isMobile ? 5 : 4}
                  style={{
                    margin: 0,
                    marginTop: isMobile ? 0 : -3,
                    display: 'inline-flex',
                    alignItems: 'center',
                    maxWidth: '100%'
                  }}
                >
                  {detail.title || '工单详情'}
                </Title>
                {renderStatusTag(detail.status)}
              </div>
              <Steps
                current={getTicketStepIndex(detail.status)}
                status={getTicketStepsStatus(detail.status)}
                direction={isMobile ? 'vertical' : 'horizontal'}
                size={isMobile ? 'small' : 'default'}
                items={stepItems(detail)}
              />

              <div>
                <Text type="secondary">描述</Text>
                <Paragraph style={{ marginTop: 8, whiteSpace: 'pre-wrap' }}>
                  {detail.description || '无描述'}
                </Paragraph>
              </div>

              <div>
                <Text type="secondary">附件</Text>
                {detail.attachments?.length ? (
                  <List
                    dataSource={detail.attachments}
                    renderItem={(item) => (
                      <List.Item>
                        <Space direction="vertical" size={6} style={{ width: '100%' }}>
                          {isImageAttachment(item) && item.download_url && (
                            <Image
                              width={isMobile ? 110 : 160}
                              src={item.download_url}
                              alt={item.file_name}
                              style={{ borderRadius: 6 }}
                            />
                          )}
                          <Space direction="vertical" size={2}>
                            <Text>{item.file_name}</Text>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {Math.ceil(item.size / 1024)} KB · {formatDateTime(item.created_at)}
                            </Text>
                          </Space>
                        {item.download_url ? (
                          isImageAttachment(item) ? null : (
                            <Button type="link" href={item.download_url} target="_blank" style={{ padding: 0 }}>
                              下载
                            </Button>
                          )
                        ) : (
                          <Text type="secondary">无下载链接</Text>
                        )}
                        </Space>
                      </List.Item>
                    )}
                  />
                ) : (
                  <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>无附件</Text>
                )}
              </div>

              <Divider />

              <div>
                <Title level={5} style={{ margin: 0 }}>问题沟通</Title>
                <Space size={8} wrap>
                  <Text type="secondary">可在此补充说明或上传附件</Text>
                  {wsStatus !== 'open' && (
                    <Text type="secondary">
                      {wsStatus === 'connecting' || wsStatus === 'idle'
                        ? '实时连接中...'
                        : '实时连接已断开，正在重连...'}
                    </Text>
                  )}
                </Space>
              </div>

              <Card size="small" style={{ background: '#fafafa', borderRadius: isMobile ? 8 : 10 }}>
                <List
                  dataSource={messages}
                  loading={messageLoading}
                  locale={{ emptyText: '暂无对话内容' }}
                  renderItem={(msg) => {
                    const images = (msg.attachments || []).filter((a) => isImageAttachment(a) && a.download_url)
                    const files = (msg.attachments || []).filter((a) => !isImageAttachment(a) || !a.download_url)
                    return (
                      <List.Item>
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                          <Space size={8} wrap>
                            <Tag color={msg.sender_type === 'system' ? 'blue' : 'green'}>
                              {msg.sender_type === 'system' ? '系统' : '企业'}
                            </Tag>
                            <Text strong>{getSenderLabel(msg)}</Text>
                            <Text type="secondary">{formatDateTime(msg.created_at)}</Text>
                          </Space>
                          {msg.content && (
                            <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{msg.content}</Paragraph>
                          )}
                          {images.length > 0 && (
                            <Image.PreviewGroup>
                              <Space wrap>
                                {images.map((img) => (
                                  <Image
                                    key={img.id}
                                    width={isMobile ? 96 : 120}
                                    src={img.download_url}
                                    alt={img.file_name}
                                    style={{ borderRadius: 6 }}
                                  />
                                ))}
                              </Space>
                            </Image.PreviewGroup>
                          )}
                          {files.length > 0 && (
                            <Space direction="vertical" size={4}>
                              {files.map((file) => (
                                file.download_url ? (
                                  <Button
                                    key={file.id}
                                    type="link"
                                    href={file.download_url}
                                    target="_blank"
                                    style={{ padding: 0, height: 'auto' }}
                                  >
                                    {file.file_name}
                                  </Button>
                                ) : (
                                  <Text key={file.id} type="secondary">
                                    {file.file_name}（无下载链接）
                                  </Text>
                                )
                              ))}
                            </Space>
                          )}
                        </Space>
                      </List.Item>
                    )
                  }}
                />
              </Card>

              <Card size="small" style={{ borderRadius: isMobile ? 8 : 10 }}>
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <Input.TextArea
                    rows={3}
                    value={messageContent}
                    onChange={(e) => setMessageContent(e.target.value)}
                    placeholder="请输入回复内容"
                    disabled={isResolved}
                  />
                  <div
                    style={{
                      display: 'flex',
                      flexDirection: isMobile ? 'column' : 'row',
                      alignItems: isMobile ? 'stretch' : 'center',
                      justifyContent: 'space-between',
                      gap: 12,
                      flexWrap: 'wrap'
                    }}
                  >
                    <Space size={8} wrap style={isMobile ? { width: '100%' } : undefined}>
                    <Upload
                      multiple
                      beforeUpload={beforeUpload}
                      fileList={messageFiles}
                      onChange={({ fileList }) => setMessageFiles(fileList)}
                      disabled={isResolved}
                    >
                      <Button icon={<UploadOutlined />} disabled={isResolved} style={isMobile ? { width: '100%' } : undefined}>上传附件</Button>
                    </Upload>
                    <Text type="secondary">最多 5 个附件，单个不超过 20MB</Text>
                    </Space>
                  <Button
                    type="primary"
                    onClick={sendMessage}
                    loading={sending}
                    disabled={isResolved}
                    style={isMobile ? { width: '100%' } : undefined}
                  >
                    发送
                  </Button>
                </div>
              </Space>
            </Card>

            </Space>
          </Card>

          <Card title="相关信息" style={{ width: isMobile ? '100%' : 280, maxWidth: '100%', borderRadius: isMobile ? 10 : 12 }}>
            {renderInfoItem('工单ID', detail.id)}
            {renderInfoItem('提交时间', formatDateTime(detail.created_at))}
            {renderInfoItem('提交账号', detail.created_by_email || detail.created_by)}
            <Button danger block onClick={handleDelete} disabled={!canDelete} style={{ marginTop: 8 }}>
              删除工单
            </Button>
            <Button
              type="primary"
              onClick={handleClose}
              disabled={isResolved}
              block
              style={{ marginTop: 8 }}
            >
              关闭工单
            </Button>
          </Card>
        </div>
      ) : (
        <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
          <Text type="secondary">暂无详情</Text>
        </Card>
      )}
    </div>
  )
}

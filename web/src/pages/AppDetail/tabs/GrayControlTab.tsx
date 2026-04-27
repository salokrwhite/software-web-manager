import {
  Alert,
  Button,
  Card,
  Col,
  Checkbox,
  DatePicker,
  Empty,
  Form,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  message,
  Statistic,
  Typography
} from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { useState } from 'react'
import api from '../../../api/client'
import ReleasePolicySummaryCard from '../components/ReleasePolicySummaryCard'

const { RangePicker } = DatePicker
const { Option } = Select
const { Text } = Typography

type GrayControlTabProps = {
  appId: string
  releaseChannels: any[]
  releases: any[]
  channels: any[]
  isLocked: boolean
  reload: () => void
}

export default function GrayControlTab({
  appId,
  releaseChannels,
  releases,
  channels,
  isLocked,
  reload
}: GrayControlTabProps) {
  const [releaseChannelOpen, setReleaseChannelOpen] = useState(false)
  const [editingReleaseChannel, setEditingReleaseChannel] = useState<any>(null)
  const [grayCreateOpen, setGrayCreateOpen] = useState(false)
  const [grayMetricsOpen, setGrayMetricsOpen] = useState(false)
  const [grayMetricsLoading, setGrayMetricsLoading] = useState(false)
  const [grayMetricsData, setGrayMetricsData] = useState<any>(null)
  const [grayMetricsRange, setGrayMetricsRange] = useState<any>([dayjs().add(-30, 'day'), dayjs()])
  const [grayMetricsTarget, setGrayMetricsTarget] = useState<any>(null)
  const [grayAdjustOpen, setGrayAdjustOpen] = useState(false)
  const [grayAdjustTarget, setGrayAdjustTarget] = useState<any>(null)
  const [grayAdjustValue, setGrayAdjustValue] = useState<number>(10)

  const [releaseChannelForm] = Form.useForm()
  const [grayCreateForm] = Form.useForm()

  const createReleaseChannel = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await grayCreateForm.validateFields()
      const payload: any = {
        release_id: values.release_id,
        channel_code: values.channel_code,
        rollout_percent: values.rollout_percent || 100,
        paused: values.paused || false,
        whitelist: values.whitelist || [],
        rollout_start_at: values.rollout_window ? values.rollout_window[0].toISOString() : null,
        rollout_end_at: values.rollout_window ? values.rollout_window[1].toISOString() : null
      }
      await api.post(`/api/apps/${appId}/release-channels`, payload)
      message.success('灰度策略已创建')
      setGrayCreateOpen(false)
      grayCreateForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    }
  }

  const updateReleaseChannel = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await releaseChannelForm.validateFields()
      const payload: any = {
        rollout_percent: values.rollout_percent,
        paused: values.paused || false,
        whitelist: values.whitelist || [],
        rollout_start_at: values.rollout_window ? values.rollout_window[0].toISOString() : null,
        rollout_end_at: values.rollout_window ? values.rollout_window[1].toISOString() : null
      }
      await api.patch(`/api/release-channels/${editingReleaseChannel.id}`, payload)
      message.success('更新成功')
      setReleaseChannelOpen(false)
      releaseChannelForm.resetFields()
      setEditingReleaseChannel(null)
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
    }
  }

  const toggleReleaseChannelPaused = async (record: any, paused: boolean) => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      await api.patch(`/api/release-channels/${record.id}`, { paused })
      message.success(paused ? '已暂停' : '已恢复')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '操作失败')
    }
  }

  const setReleaseChannelRollout = async (record: any, percent: number) => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      await api.patch(`/api/release-channels/${record.id}`, { rollout_percent: percent })
      message.success(`灰度已调整为 ${percent}%`)
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '调整失败')
    }
  }

  const openAdjustRollout = (record: any) => {
    setGrayAdjustTarget(record)
    setGrayAdjustValue(10)
    setGrayAdjustOpen(true)
  }

  const applyAdjustRollout = async () => {
    if (!grayAdjustTarget) return
    const nextPercent = Math.min(100, Math.max(1, (grayAdjustTarget.rollout_percent || 0) + grayAdjustValue))
    try {
      await api.patch(`/api/release-channels/${grayAdjustTarget.id}`, { rollout_percent: nextPercent })
      message.success(`灰度已调整为 ${nextPercent}%`)
      setGrayAdjustOpen(false)
      setGrayAdjustTarget(null)
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '调整失败')
    }
  }

  const loadGrayMetrics = async (record: any, range: any[] = grayMetricsRange) => {
    if (!record) return
    setGrayMetricsLoading(true)
    try {
      const params = {
        from: range[0].format('YYYY-MM-DD'),
        to: range[1].format('YYYY-MM-DD')
      }
      const res = await api.get(`/api/apps/${appId}/release-channels/${record.id}/metrics`, { params })
      setGrayMetricsData(res.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载指标失败')
    } finally {
      setGrayMetricsLoading(false)
    }
  }

  const openGrayMetrics = async (record: any) => {
    setGrayMetricsTarget(record)
    setGrayMetricsOpen(true)
    await loadGrayMetrics(record)
  }

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <Text type="secondary">灰度策略用于分批发布与控制范围</Text>
        </Col>
        <Col>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setGrayCreateOpen(true)} disabled={isLocked}>
              创建灰度策略
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <ReleasePolicySummaryCard
        releaseChannels={releaseChannels}
        title="发布策略摘要（流量视图）"
      />
      {releaseChannels.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="暂无灰度策略，创建后即可控制发布范围"
        >
          <Button type="primary" onClick={() => setGrayCreateOpen(true)} disabled={isLocked}>
            创建灰度策略
          </Button>
        </Empty>
      ) : (
        <Table
          rowKey="id"
          dataSource={releaseChannels}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: '渠道', dataIndex: 'channel_code', render: (c: string) => <Tag>{c}</Tag> },
            { title: '版本', dataIndex: 'release_version' },
            { title: '状态', dataIndex: 'status', render: (s: string) => s === 'active' ? <Tag color="success">active</Tag> : <Tag>{s}</Tag> },
            { title: '灰度', dataIndex: 'rollout_percent', render: (v: number) => `${v}%` },
            {
              title: '时间窗',
              render: (_: any, record: any) => {
                if (!record.rollout_start_at || !record.rollout_end_at) return '-'
                const start = dayjs(record.rollout_start_at).format('YYYY-MM-DD')
                const end = dayjs(record.rollout_end_at).format('YYYY-MM-DD')
                return `${start} ~ ${end}`
              }
            },
            { title: '暂停', dataIndex: 'paused', render: (v: boolean) => v ? <Tag color="orange">已暂停</Tag> : <Tag color="green">运行中</Tag> },
            {
              title: '操作',
              render: (_: any, record: any) => (
                <Space>
                  <Button size="small" onClick={() => openGrayMetrics(record)}>指标</Button>
                  <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
                    <Button size="small" disabled={isLocked} onClick={() => {
                      setEditingReleaseChannel(record)
                      releaseChannelForm.setFieldsValue({
                        rollout_percent: record.rollout_percent,
                        paused: record.paused,
                        whitelist: record.whitelist || [],
                        rollout_window: record.rollout_start_at && record.rollout_end_at
                          ? [dayjs(record.rollout_start_at), dayjs(record.rollout_end_at)]
                          : null
                      })
                      setReleaseChannelOpen(true)
                    }}>
                      编辑
                    </Button>
                  </Tooltip>
                  <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
                    <Button size="small" disabled={isLocked} onClick={() => toggleReleaseChannelPaused(record, !record.paused)}>
                      {record.paused ? '恢复' : '暂停'}
                    </Button>
                  </Tooltip>
                  <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
                    <Button size="small" disabled={isLocked} onClick={() => setReleaseChannelRollout(record, 100)}>
                      全量
                    </Button>
                  </Tooltip>
                  <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
                    <Button size="small" disabled={isLocked} onClick={() => openAdjustRollout(record)}>
                      加量
                    </Button>
                  </Tooltip>
                </Space>
              )
            }
          ]}
        />
      )}

      <Modal
        title="创建灰度策略"
        open={grayCreateOpen}
        onOk={createReleaseChannel}
        onCancel={() => { setGrayCreateOpen(false); grayCreateForm.resetFields() }}
        width={560}
      >
        <Form layout="vertical" form={grayCreateForm} style={{ marginTop: 16 }}>
          <Form.Item name="release_id" label="选择版本" rules={[{ required: true, message: '请选择版本' }]}>
            <Select placeholder="选择已审核/已发布版本">
              {releases
                .filter((r) => ['approved', 'published'].includes((r.status || '').toLowerCase()))
                .map((r) => (
                  <Option key={r.id} value={r.id}>{r.version}</Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name="channel_code" label="选择渠道" rules={[{ required: true, message: '请选择渠道' }]}>
            <Select placeholder="选择渠道">
              {channels.map((c) => (
                <Option key={c.code} value={c.code}>{c.name} ({c.code})</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="rollout_percent" label="灰度百分比" initialValue={100}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} size="large" />
          </Form.Item>
          <Form.Item name="rollout_window" label="灰度时间窗">
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="paused" valuePropName="checked">
            <Checkbox>创建后先暂停</Checkbox>
          </Form.Item>
          <Form.Item name="whitelist" label="白名单设备ID">
            <Select mode="tags" placeholder="输入设备ID，回车确认" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={grayMetricsTarget ? `灰度指标 - ${grayMetricsTarget.channel_code} / ${grayMetricsTarget.release_version}` : '灰度指标'}
        open={grayMetricsOpen}
        onCancel={() => { setGrayMetricsOpen(false); setGrayMetricsTarget(null); setGrayMetricsData(null) }}
        footer={null}
        width={760}
      >
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Row justify="space-between" align="middle">
            <Col>
              <Text type="secondary">时间范围</Text>
            </Col>
            <Col>
              <RangePicker
                value={grayMetricsRange}
                onChange={(range) => {
                  if (!range) return
                  setGrayMetricsRange(range)
                  if (grayMetricsTarget) {
                    loadGrayMetrics(grayMetricsTarget, range)
                  }
                }}
              />
            </Col>
          </Row>
          {grayMetricsData && !grayMetricsData.has_release_events && (
            <Alert type="info" message="请升级客户端以支持指标" showIcon />
          )}
          <Row gutter={[16, 16]}>
            {[
              { key: 'update_available', label: '发现更新' },
              { key: 'download_completed', label: '下载完成' },
              { key: 'install_completed', label: '安装完成' },
              { key: 'update_failed', label: '更新失败' },
              { key: 'app_started', label: '应用启动' }
            ].map((item) => (
              <Col xs={12} md={8} key={item.key}>
                <Card size="small" style={{ borderRadius: 8 }}>
                  <Statistic
                    title={item.label}
                    value={grayMetricsData?.summary?.[item.key] || 0}
                  />
                </Card>
              </Col>
            ))}
          </Row>
          <Card size="small" title="时间趋势" style={{ borderRadius: 8 }}>
            <Table
              rowKey="date"
              size="small"
              pagination={false}
              loading={grayMetricsLoading}
              dataSource={grayMetricsData?.timeline || []}
              columns={[
                { title: '日期', dataIndex: 'date' },
                { title: '发现更新', dataIndex: 'update_available' },
                { title: '下载完成', dataIndex: 'download_completed' },
                { title: '安装完成', dataIndex: 'install_completed' },
                { title: '更新失败', dataIndex: 'update_failed' },
                { title: '应用启动', dataIndex: 'app_started' }
              ]}
            />
          </Card>
        </Space>
      </Modal>

      <Modal
        title="灰度加量"
        open={grayAdjustOpen}
        onOk={applyAdjustRollout}
        onCancel={() => { setGrayAdjustOpen(false); setGrayAdjustTarget(null) }}
        width={400}
      >
        <Form layout="vertical">
          <Form.Item label="加量比例(%)">
            <InputNumber
              min={1}
              max={100}
              value={grayAdjustValue}
              onChange={(v) => setGrayAdjustValue(Number(v) || 0)}
              style={{ width: '100%' }}
            />
          </Form.Item>
          {grayAdjustTarget && (
            <Text type="secondary">当前 {grayAdjustTarget.rollout_percent || 0}%，将调整为不超过 100%</Text>
          )}
        </Form>
      </Modal>

      <Modal
        title="更新灰度策略"
        open={releaseChannelOpen}
        onOk={updateReleaseChannel}
        onCancel={() => { setReleaseChannelOpen(false); releaseChannelForm.resetFields(); setEditingReleaseChannel(null) }}
        width={560}
      >
        <Form layout="vertical" form={releaseChannelForm} style={{ marginTop: 16 }}>
          <Form.Item name="rollout_percent" label="灰度百分比">
            <InputNumber min={1} max={100} style={{ width: '100%' }} size="large" />
          </Form.Item>
          <Form.Item name="paused" valuePropName="checked">
            <Checkbox>暂停更新</Checkbox>
          </Form.Item>
          <Form.Item name="rollout_window" label="灰度时间窗">
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="whitelist" label="白名单设备ID">
            <Select mode="tags" placeholder="输入设备ID，回车确认" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

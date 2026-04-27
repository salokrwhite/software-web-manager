import {
  Card,
  Avatar,
  Button,
  Checkbox,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Upload,
  message,
  Grid,
  theme,
  Typography
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  RocketOutlined,
  RollbackOutlined,
  UploadOutlined
} from '@ant-design/icons'
import { useState } from 'react'
import dayjs from 'dayjs'
import { useNavigate } from 'react-router-dom'
import api from '../../../api/client'
import ReleasePolicySummaryCard from '../components/ReleasePolicySummaryCard'
import { getStatusTag } from '../utils/statusTag'
import { buildRegionRulesFromTemplates } from '../utils/region'
import { buildTargetingRules, summarizeTargetingRules } from '../utils/targeting'

const { Text } = Typography
const { Option } = Select
const { RangePicker } = DatePicker

type ReleasesTabProps = {
  appId: string
  channels: any[]
  releases: any[]
  releaseChannels: any[]
  regionTemplates: any[]
  activeRegionTemplateId: string
  regionEnabled: boolean
  isLocked: boolean
  isPersonal: boolean
  canReviewRelease: boolean
  reload: () => void
  setReleases: (updater: any) => void
  loading: boolean
}

export default function ReleasesTab({
  appId,
  channels,
  releases,
  releaseChannels,
  regionTemplates,
  activeRegionTemplateId,
  regionEnabled,
  isLocked,
  isPersonal,
  canReviewRelease,
  reload,
  setReleases,
  loading
}: ReleasesTabProps) {
  const navigate = useNavigate()
  const { token } = theme.useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const formControlSize = isMobile ? 'middle' : 'large'

  const [publishOpen, setPublishOpen] = useState(false)
  const [publishReleaseId, setPublishReleaseId] = useState<string | null>(null)
  const [reviewOpen, setReviewOpen] = useState(false)
  const [reviewAction, setReviewAction] = useState<'submit' | 'approve' | 'reject'>('submit')
  const [reviewReleaseId, setReviewReleaseId] = useState<string | null>(null)
  const [rollbackOpen, setRollbackOpen] = useState(false)
  const [rollbackReleaseId, setRollbackReleaseId] = useState<string | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadReleaseId, setUploadReleaseId] = useState<string | null>(null)
  const [uploadMode, setUploadMode] = useState<'create' | 'replace'>('create')
  const [uploadFile, setUploadFile] = useState<any>(null)
  const [editReleaseOpen, setEditReleaseOpen] = useState(false)
  const [editingRelease, setEditingRelease] = useState<any>(null)

  const [publishForm] = Form.useForm()
  const [reviewForm] = Form.useForm()
  const [rollbackForm] = Form.useForm()
  const [uploadForm] = Form.useForm()
  const [editReleaseForm] = Form.useForm()

  const activeChannelsForRelease = (releaseId: string) => {
    return releaseChannels.filter((rc) => rc.release_id === releaseId && rc.status === 'active')
  }

  const activeAppRegionTemplate = regionTemplates.find((tpl: any) => tpl.id === activeRegionTemplateId) || regionTemplates[0]

  const buildRegionRulesPayload = (mode: string, templateId?: string) => {
    if (mode !== 'template') {
      return undefined
    }
    const selected = regionTemplates.find((tpl: any) => tpl.id === templateId)
    if (!selected) {
      return undefined
    }
    return buildRegionRulesFromTemplates([selected], selected.id)
  }

  const openPublishModal = (releaseId: string) => {
    const defaultChannel = channels.find((c: any) => c?.is_default)
    setPublishReleaseId(releaseId)
    setPublishOpen(true)
    publishForm.setFieldsValue({
      channel_code: defaultChannel?.code,
      rollout_percent: 100,
      paused: false,
      mandatory: false,
      whitelist: [],
      user_ids: [],
      device_ids: [],
      platforms: [],
      archs: [],
      min_version: '',
      max_version: '',
      region_mode: 'inherit',
      region_template_id: activeAppRegionTemplate?.id
    })
  }

  const publishRelease = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await publishForm.validateFields()
      const payload: any = {
        channel_code: values.channel_code,
        rollout_percent: values.rollout_percent,
        mandatory: !!values.mandatory,
        paused: !!values.paused,
        whitelist: values.whitelist || []
      }
      if (values.rollout_window && values.rollout_window.length === 2) {
        payload.rollout_start_at = values.rollout_window[0].toISOString()
        payload.rollout_end_at = values.rollout_window[1].toISOString()
      }
      const targetingRules = buildTargetingRules(values)
      if (Object.keys(targetingRules).length > 0) {
        payload.targeting_rules = targetingRules
      }
      const regionRules = buildRegionRulesPayload(values.region_mode, values.region_template_id)
      if (regionRules) {
        payload.region_rules = regionRules
      }
      await api.post(`/api/releases/${publishReleaseId}/publish`, payload)
      message.success('发布成功')
      setPublishOpen(false)
      publishForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '发布失败')
    }
  }

  const submitReview = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await reviewForm.validateFields()
      const url = `/api/releases/${reviewReleaseId}/${reviewAction}`
      await api.post(url, values)
      message.success('操作成功')
      setReviewOpen(false)
      reviewForm.resetFields()
      reload()
    } catch (err: any) {
      const code = err?.response?.data?.error
      if (code === 'artifact_required') {
        message.warning('请先上传新版本软件后再提交审核')
        return
      }
      message.error(code || '操作失败')
    }
  }

  const rollbackRelease = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await rollbackForm.validateFields()
      await api.post(`/api/releases/${rollbackReleaseId}/rollback`, values)
      message.success('回滚成功')
      setRollbackOpen(false)
      rollbackForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '回滚失败')
    }
  }

  const handleUpload = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    if (!uploadFile || !uploadReleaseId) return
    try {
      const values = await uploadForm.validateFields()
      const form = new FormData()
      form.append('file', uploadFile)
      form.append('platform', values.platform)
      form.append('arch', values.arch)
      form.append('file_type', values.file_type)
      if (values.signature) {
        form.append('signature', values.signature)
      }
      if (uploadMode === 'replace') {
        form.append('replace', 'true')
      }
      await api.post(`/api/releases/${uploadReleaseId}/artifacts`, form)
      message.success('上传成功')
      setUploadOpen(false)
      setUploadFile(null)
      setUploadReleaseId(null)
      setUploadMode('create')
      uploadForm.resetFields()
      if (uploadReleaseId) {
        setReleases((prev: any[]) => prev.map((r) => (
          r.id === uploadReleaseId ? { ...r, artifact_count: Math.max(Number(r.artifact_count || 0), 1) } : r
        )))
      }
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '上传失败')
    }
  }

  const openUploadModal = (releaseId: string, mode: 'create' | 'replace' = 'create') => {
    setUploadMode(mode)
    setUploadReleaseId(releaseId)
    setUploadOpen(true)
    setUploadFile(null)
    uploadForm.resetFields()
  }

  const openSubmitReview = (record: any) => {
    if (isPersonal) {
      const count = Number(record?.artifact_count || 0)
      const externalLink = String(record?.external_download_url || record?.ExternalDownloadURL || '').trim()
      const hasExternalLink = Boolean(externalLink)
      if (count <= 0 && !hasExternalLink) {
        message.warning('请先上传新版本软件后再提交审核')
        return
      }
    }
    setReviewAction('submit')
    setReviewReleaseId(record.id)
    setReviewOpen(true)
  }

  const deleteRelease = (record: any) => {
    Modal.confirm({
      title: '确认删除版本',
      content: `确定要删除版本 ${record.version} 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.delete(`/api/releases/${record.id}`)
          message.success('删除成功')
          reload()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  const openEditRelease = (record: any) => {
    setEditingRelease(record)
    setEditReleaseOpen(true)
    editReleaseForm.setFieldsValue({
      version: record.version || '',
      version_code: record.version_code != null ? String(record.version_code) : ''
    })
  }

  const submitEditRelease = async () => {
    if (!editingRelease) return
    try {
      const values = await editReleaseForm.validateFields()
      const payload: any = { ...values }
      if (payload.version_code !== undefined && payload.version_code !== null && String(payload.version_code).trim() !== '') {
        payload.version_code = Number(payload.version_code)
      } else {
        delete payload.version_code
      }
      await api.patch(`/api/releases/${editingRelease.id}`, payload)
      message.success('更新成功')
      setEditReleaseOpen(false)
      setEditingRelease(null)
      editReleaseForm.resetFields()
      reload()
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || '更新失败')
    }
  }

  return (
    <>
      <Row
        justify="space-between"
        align={isMobile ? 'top' : 'middle'}
        style={{ marginBottom: isMobile ? 12 : 16 }}
        gutter={isMobile ? [12, 12] : undefined}
      >
        <Col xs={24} sm={12}>
          <Text type="secondary">共 {releases.length} 个版本</Text>
        </Col>
        <Col xs={24} sm={12} style={isMobile ? undefined : { textAlign: 'right' }}>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              disabled={isLocked}
              style={isMobile ? { width: '100%' } : undefined}
              onClick={() => navigate(`/apps/${appId}/releases/new`)}
            >
              新建版本
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <ReleasePolicySummaryCard
        releaseChannels={releaseChannels}
        title="发布策略摘要（版本视图）"
      />
      <Table
        rowKey="id"
        dataSource={releases}
        loading={loading}
        size={isMobile ? 'small' : 'middle'}
        scroll={isMobile ? { x: 960 } : { x: 1100 }}
        pagination={{
          pageSize: 10,
          size: isMobile ? 'small' : 'default',
          responsive: true,
          showSizeChanger: !isMobile
        }}
        columns={[
          {
            title: '版本',
            dataIndex: 'version',
            width: isMobile ? 150 : 180,
            render: (v: string) => (
              <Space size={isMobile ? 6 : 8}>
                <Avatar size={isMobile ? 22 : 26} icon={<RocketOutlined />} style={{ background: token.colorPrimary }} />
                <Text strong style={isMobile ? { fontSize: 13 } : undefined}>{v}</Text>
              </Space>
            )
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 110,
            render: (s: string) => getStatusTag(s)
          },
          {
            title: '发布渠道',
            width: isMobile ? 210 : 240,
            render: (_: any, record: any) => (
              <Space size={[4, 4]} wrap>
                {activeChannelsForRelease(record.id).length > 0
                  ? activeChannelsForRelease(record.id).map((rc) => (
                    <Tag key={rc.id} color="success">{rc.channel_code}</Tag>
                  ))
                  : <Text type="secondary">-</Text>}
              </Space>
            )
          },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 140,
            render: (d: string) => new Date(d).toLocaleDateString()
          },
          {
            title: '操作',
            width: isMobile ? 300 : 460,
            fixed: isMobile ? undefined : 'right',
            render: (_: any, record: any) => {
              const lockOps = isLocked
              const hasArtifacts = Number(record?.artifact_count || 0) > 0
              const externalLink = String(record?.external_download_url || record?.ExternalDownloadURL || '').trim()
              const isExternalOnly = externalLink.length > 0
              const status = (record.status || '').toLowerCase()
              // Backend allows uploading artifacts for approved releases as well.
              const canUpload = status === 'draft' || status === 'rejected' || status === 'approved'
              const uploadDisabled = lockOps || !canUpload || isExternalOnly
              return (
                <Space size={[6, 6]} wrap>
                  {isPersonal && (record.status === 'draft' || record.status === 'rejected') && (
                    <Button
                      size="small"
                      disabled={lockOps}
                      onClick={() => openSubmitReview(record)}
                    >
                      {isMobile ? '提交' : '提交审核'}
                    </Button>
                  )}
                  {record.status === 'in_review' && (
                    canReviewRelease ? (
                      <Space size={4}>
                        <Button size="small" type="primary" disabled={lockOps} onClick={() => { setReviewAction('approve'); setReviewReleaseId(record.id); setReviewOpen(true) }}>通过</Button>
                        <Button size="small" danger disabled={lockOps} onClick={() => { setReviewAction('reject'); setReviewReleaseId(record.id); setReviewOpen(true) }}>拒绝</Button>
                      </Space>
                    ) : null
                  )}
                  {record.status === 'approved' && (
                    <Button size="small" type="primary" disabled={lockOps} onClick={() => openPublishModal(record.id)}>
                      发布
                    </Button>
                  )}
                  {record.status === 'published' && (
                    <Button size="small" icon={<RollbackOutlined />} disabled={lockOps} onClick={() => { setRollbackReleaseId(record.id); setRollbackOpen(true) }}>
                      {!isMobile && '回滚'}
                    </Button>
                  )}
                  <Tooltip title={isExternalOnly ? '该版本已配置外部下载链接，不能上传软件包' : ''}>
                    <Button size="small" icon={<UploadOutlined />} disabled={uploadDisabled} onClick={() => openUploadModal(record.id, hasArtifacts ? 'replace' : 'create')}>
                      {!isMobile && '上传'}
                    </Button>
                  </Tooltip>
                  <Button size="small" icon={<EditOutlined />} onClick={() => openEditRelease(record)}>
                    {!isMobile && '编辑'}
                  </Button>
                  <Button size="small" danger icon={<DeleteOutlined />} disabled={lockOps} onClick={() => deleteRelease(record)}>
                    {!isMobile && '删除'}
                  </Button>
                </Space>
              )
            }
          }
        ]}
      />

      <Modal
        title="发布版本"
        open={publishOpen}
        onOk={publishRelease}
        onCancel={() => { setPublishOpen(false); publishForm.resetFields() }}
        width={isMobile ? 'calc(100vw - 24px)' : 980}
      >
        <Form layout="vertical" form={publishForm} style={{ marginTop: 16 }}>
          <Form.Item name="channel_code" label="发布渠道" rules={[{ required: true }]}>
            <Select placeholder="请选择发布渠道" size={formControlSize}>
              {channels.map((c: any) => (
                <Option key={c.id} value={c.code}>
                  {c.name} ({c.code})
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Row gutter={isMobile ? [12, 0] : 16}>
            <Col xs={24} md={12}>
              <Form.Item name="rollout_percent" label="灰度百分比" initialValue={100}>
                <InputNumber min={1} max={100} style={{ width: '100%' }} size={formControlSize} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="rollout_window" label="生效时间窗">
                <RangePicker showTime style={{ width: '100%' }} size={formControlSize} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={isMobile ? [12, 0] : 16}>
            <Col xs={24} md={12}>
              <Form.Item name="mandatory" valuePropName="checked">
                <Checkbox>强制更新</Checkbox>
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="paused" valuePropName="checked">
                <Checkbox>暂停该通道更新</Checkbox>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="whitelist" label="白名单设备ID">
            <Select mode="tags" placeholder="输入设备ID，回车确认" size={formControlSize} />
          </Form.Item>

          <Row gutter={16} align="top">
            <Col xs={24} md={12}>
              <Card size="small" title="目标规则（结构化）" style={{ borderRadius: 8 }}>
                <Form.Item name="platforms" label="平台">
                  <Select
                    mode="multiple"
                    placeholder="选择平台"
                    size={formControlSize}
                    options={[
                      { label: 'Windows', value: 'windows' },
                      { label: 'macOS', value: 'mac' },
                      { label: 'Linux', value: 'linux' },
                      { label: 'Android', value: 'android' },
                      { label: 'iOS', value: 'ios' },
                      { label: 'Universal', value: 'universal' }
                    ]}
                  />
                </Form.Item>
                <Form.Item name="archs" label="架构">
                  <Select
                    mode="multiple"
                    placeholder="选择架构"
                    size={formControlSize}
                    options={[
                      { label: 'x64', value: 'x64' },
                      { label: 'x86', value: 'x86' },
                      { label: 'arm64', value: 'arm64' },
                      { label: 'arm', value: 'arm' },
                      { label: 'universal', value: 'universal' }
                    ]}
                  />
                </Form.Item>
                <Form.Item name="user_ids" label="用户ID">
                  <Select mode="tags" placeholder="输入用户ID" size={formControlSize} />
                </Form.Item>
                <Form.Item name="device_ids" label="设备ID">
                  <Select mode="tags" placeholder="输入设备ID" size={formControlSize} />
                </Form.Item>
                <Row gutter={8}>
                  <Col xs={24} md={12}>
                    <Form.Item name="min_version" label="最低版本">
                      <Input placeholder="例如：1.0.0" size={formControlSize} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="max_version" label="最高版本">
                      <Input placeholder="例如：2.0.0" size={formControlSize} />
                    </Form.Item>
                  </Col>
                </Row>
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card size="small" title="区域规则（并排联动）" style={{ borderRadius: 8 }}>
                <Form.Item name="region_mode" label="区域策略模式" initialValue="inherit">
                  <Select
                    size={formControlSize}
                    options={[
                      { label: '继承应用级区域规则', value: 'inherit' },
                      { label: '使用指定区域模板', value: 'template' }
                    ]}
                  />
                </Form.Item>
                <Form.Item shouldUpdate noStyle>
                  {({ getFieldValue }) => {
                    const mode = getFieldValue('region_mode')
                    if (mode !== 'template') {
                      return (
                        <Text type="secondary">
                          当前发布通道将继承应用级区域规则：{regionEnabled ? (activeAppRegionTemplate?.name || '未配置模板') : '未启用'}
                        </Text>
                      )
                    }
                    return (
                      <Form.Item
                        name="region_template_id"
                        label="选择区域模板"
                        rules={[{ required: true, message: '请选择区域模板' }]}
                      >
                        <Select
                          placeholder="请选择区域模板"
                          size={formControlSize}
                          options={regionTemplates.map((tpl: any) => ({
                            label: tpl.name,
                            value: tpl.id
                          }))}
                          notFoundContent="暂无模板，请先到“发布策略（人群）”维护"
                        />
                      </Form.Item>
                    )
                  }}
                </Form.Item>
              </Card>
              <Card size="small" title="最终生效范围预览" style={{ borderRadius: 8, marginTop: 12 }}>
                <Form.Item shouldUpdate noStyle>
                  {({ getFieldValue }) => {
                    const previewValues = {
                      user_ids: getFieldValue('user_ids'),
                      device_ids: getFieldValue('device_ids'),
                      platforms: getFieldValue('platforms'),
                      archs: getFieldValue('archs'),
                      min_version: getFieldValue('min_version'),
                      max_version: getFieldValue('max_version')
                    }
                    const targetingSummary = summarizeTargetingRules(previewValues)
                    const regionMode = getFieldValue('region_mode') || 'inherit'
                    const regionTemplateId = getFieldValue('region_template_id')
                    const selectedTemplate = regionTemplates.find((tpl: any) => tpl.id === regionTemplateId)
                    const rolloutPercent = Number(getFieldValue('rollout_percent') || 100)
                    const paused = !!getFieldValue('paused')
                    const whitelist = getFieldValue('whitelist') || []
                    const rolloutWindow = getFieldValue('rollout_window')
                    const timeText = rolloutWindow && rolloutWindow.length === 2
                      ? `${dayjs(rolloutWindow[0]).format('YYYY-MM-DD HH:mm')} ~ ${dayjs(rolloutWindow[1]).format('YYYY-MM-DD HH:mm')}`
                      : '长期有效'
                    const regionText = regionMode === 'template'
                      ? `通道覆盖：${selectedTemplate?.name || '未选择模板'}`
                      : `继承应用级：${regionEnabled ? (activeAppRegionTemplate?.name || '未配置模板') : '未启用'}`
                    return (
                      <Space direction="vertical" size={6} style={{ width: '100%' }}>
                        <Text>时间层：{timeText}</Text>
                        <Text>人群层(区域)：{regionText}</Text>
                        <Text>
                          人群层(目标)：
                          {targetingSummary.length > 0 ? targetingSummary.join(' / ') : '不限'}
                        </Text>
                        <Text>
                          流量层：灰度 {rolloutPercent}% / 白名单 {Array.isArray(whitelist) ? whitelist.length : 0} / {paused ? '已暂停' : '运行中'}
                        </Text>
                        <Text type="secondary">
                          实际命中 = 满足时间层 AND 满足区域规则 AND 满足目标规则 AND 命中灰度或白名单
                        </Text>
                      </Space>
                    )
                  }}
                </Form.Item>
              </Card>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        title="提交/审批"
        open={reviewOpen}
        onOk={submitReview}
        onCancel={() => { setReviewOpen(false); reviewForm.resetFields() }}
        width={isMobile ? 'calc(100vw - 24px)' : 480}
      >
        <Form layout="vertical" form={reviewForm} style={{ marginTop: 16 }}>
          <Form.Item name="note" label="备注">
            <Input.TextArea rows={3} placeholder="可选备注" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="回滚版本"
        open={rollbackOpen}
        onOk={rollbackRelease}
        onCancel={() => { setRollbackOpen(false); rollbackForm.resetFields() }}
        width={isMobile ? 'calc(100vw - 24px)' : 480}
      >
        <Form layout="vertical" form={rollbackForm} style={{ marginTop: 16 }}>
          <Form.Item name="channel_code" label="渠道" rules={[{ required: true }]}>
            <Input placeholder="例如：stable" size={formControlSize} />
          </Form.Item>
          <Form.Item name="release_id" label="目标版本(可选)">
            <Select allowClear placeholder="选择版本" size={formControlSize}>
              {releases.map((r) => (
                <Option key={r.id} value={r.id}>{r.version}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={uploadMode === 'replace' ? '修改上传软件包' : '上传软件包'}
        open={uploadOpen}
        onOk={handleUpload}
        onCancel={() => {
          setUploadOpen(false)
          setUploadFile(null)
          setUploadReleaseId(null)
          setUploadMode('create')
          uploadForm.resetFields()
        }}
        width={isMobile ? 'calc(100vw - 24px)' : 520}
      >
        <Form layout="vertical" form={uploadForm} style={{ marginTop: 16 }}>
          <Form.Item name="platform" label="平台" rules={[{ required: true }]} initialValue="windows">
            <Select size={formControlSize}>
              <Option value="universal">通用</Option>
              <Option value="windows">Windows</Option>
              <Option value="mac">macOS</Option>
              <Option value="linux">Linux</Option>
              <Option value="android">Android</Option>
              <Option value="ios">iOS</Option>
            </Select>
          </Form.Item>
          <Form.Item name="arch" label="架构" rules={[{ required: true }]} initialValue="x64">
            <Select size={formControlSize}>
              <Option value="universal">通用</Option>
              <Option value="x64">x64 (64位)</Option>
              <Option value="x86">x86 (32位)</Option>
              <Option value="arm64">ARM64</Option>
              <Option value="arm">ARM</Option>
            </Select>
          </Form.Item>
          <Form.Item name="file_type" label="文件类型" rules={[{ required: true }]} initialValue="exe">
            <Select size={formControlSize}>
              <Option value="universal">通用</Option>
              <Option value="exe">EXE (Windows 可执行文件)</Option>
              <Option value="msi">MSI (Windows 安装包)</Option>
              <Option value="dmg">DMG (macOS 安装包)</Option>
              <Option value="pkg">PKG (macOS 安装包)</Option>
              <Option value="deb">DEB (Debian/Ubuntu)</Option>
              <Option value="rpm">RPM (RedHat/CentOS)</Option>
              <Option value="apk">APK (Android)</Option>
              <Option value="ipa">IPA (iOS)</Option>
              <Option value="zip">ZIP (压缩包)</Option>
              <Option value="tar.gz">TAR.GZ (压缩包)</Option>
            </Select>
          </Form.Item>
          <Form.Item label="软件包文件" required>
            <Upload
              beforeUpload={(file) => { setUploadFile(file); return false }}
              onRemove={() => setUploadFile(null)}
              maxCount={1}
            >
              <Button icon={<UploadOutlined />}>选择文件</Button>
            </Upload>
            {uploadFile && <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>已选择: {uploadFile.name}</Text>}
          </Form.Item>
          <Form.Item name="signature" label="签名 (可选)">
            <Input.TextArea rows={3} placeholder="文件签名，用于完整性验证" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingRelease ? `编辑版本 - ${editingRelease.version}` : '编辑版本'}
        open={editReleaseOpen}
        onOk={submitEditRelease}
        onCancel={() => { setEditReleaseOpen(false); setEditingRelease(null); editReleaseForm.resetFields() }}
        width={isMobile ? 'calc(100vw - 24px)' : 480}
        okText="保存"
      >
        <Form layout="vertical" form={editReleaseForm} style={{ marginTop: 16 }}>
          <Form.Item name="version" label="版本号" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input size={formControlSize} />
          </Form.Item>
          <Form.Item
            name="version_code"
            label="内部版本号"
            rules={[
              {
                validator: (_, value) => {
                  if (value === undefined || value === null || String(value).trim() === '') {
                    return Promise.resolve()
                  }
                  if (/^\d+$/.test(String(value).trim())) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('请输入数字'))
                }
              }
            ]}
          >
            <Input placeholder="例如：1001" size={formControlSize} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

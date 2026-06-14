import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
  Typography
} from 'antd'
import dayjs from 'dayjs'
import { useState } from 'react'
import api from '../../../api/client'
import ReleasePolicySummaryCard from '../components/ReleasePolicySummaryCard'
import FeatureGuide, { GuideTag } from '../components/FeatureGuide'
import { getStatusTag } from '../utils/statusTag'

const { RangePicker } = DatePicker
const { Text } = Typography

type ReleasePlanTabProps = {
  releases: any[]
  releaseTemplates: any[]
  releaseChannels: any[]
  isLocked: boolean
  reload: () => void
}

export default function ReleasePlanTab({
  releases,
  releaseTemplates,
  releaseChannels,
  isLocked,
  reload
}: ReleasePlanTabProps) {
  const [templateOpen, setTemplateOpen] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<any>(null)
  const [templateForm] = Form.useForm()

  const templateMap = new Map(releaseTemplates.map((t) => [t.id, t]))

  const openTemplateModal = (template?: any) => {
    setEditingTemplate(template || null)
    setTemplateOpen(true)
    templateForm.resetFields()
    if (template) {
      templateForm.setFieldsValue({
        name: template.name,
        schedule_at: template.schedule_at ? dayjs(template.schedule_at) : null,
        window_range: template.window_start && template.window_end ? [dayjs(template.window_start), dayjs(template.window_end)] : null,
        emergency: !!template.emergency
      })
    }
  }

  const saveTemplate = async () => {
    try {
      const values = await templateForm.validateFields()
      const payload = {
        name: values.name,
        schedule_at: values.schedule_at ? values.schedule_at.toISOString() : null,
        window_start: values.window_range ? values.window_range[0].toISOString() : null,
        window_end: values.window_range ? values.window_range[1].toISOString() : null,
        emergency: !!values.emergency
      }
      if (editingTemplate) {
        await api.patch(`/api/release-templates/${editingTemplate.id}`, payload)
      } else {
        await api.post('/api/release-templates', payload)
      }
      message.success('保存成功')
      setTemplateOpen(false)
      setEditingTemplate(null)
      templateForm.resetFields()
      reload()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  const deleteTemplate = (template: any) => {
    Modal.confirm({
      title: '确认删除模板',
      content: `确定要删除模板 ${template.name} 吗？`,
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.delete(`/api/release-templates/${template.id}`)
          message.success('删除成功')
          reload()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        }
      }
    })
  }

  const assignTemplate = async (releaseId: string, templateId?: string) => {
    try {
      await api.patch(`/api/releases/${releaseId}/template`, { template_id: templateId || '' })
      message.success('已更新发布模板')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
    }
  }

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24}>
        <FeatureGuide
          storageKey="release-plan"
          title="发布模板"
          summary={
            <>
              发布模板用来<Text strong>预先约定「什么时候发」</Text>：可以设置定时发布、限定发布时间窗，
              或开启紧急发布立即生效。把这些时间规则存成模板后，给版本绑定模板，到点就会自动按规则发布。
            </>
          }
          steps={[
            {
              title: '新建一个模板',
              description: <>在下方「模板管理」点<GuideTag>新建模板</GuideTag>，填好名称，设置预约发布时间或时间窗。</>
            },
            {
              title: '给版本绑定模板',
              description: <>在「版本发布计划」里，为目标版本的「发布模板」列选择刚建好的模板。</>
            },
            {
              title: '发布并自动生效',
              description: <>对版本点发布后：普通模板会到「预约时间」自动激活并在时间窗内下发；开启了紧急发布则立即生效。</>
            }
          ]}
          tips={[
            <>「预约发布」是到某个时间点才开始发；「时间窗」是限定只在这段时间区间内允许下发。</>,
            <>开启「紧急发布」会忽略预约时间和时间窗，发布后立刻对用户生效，适合修复线上故障。</>,
            <>一个模板可以被多个版本复用，修改模板会影响所有绑定它的版本。</>
          ]}
        />
      </Col>
      <Col xs={24}>
        <ReleasePolicySummaryCard
          releaseChannels={releaseChannels}
          title="发布策略摘要（模板视图）"
        />
      </Col>
      <Col xs={24}>
        <Alert
          type="info"
          showIcon
          message="模板已支持预约调度"
          description="版本绑定模板后点击“发布”：非紧急模板会按 schedule_at 到点激活，并在窗口期内下发；开启紧急发布则立即生效，忽略模板预约时间与时间窗。"
        />
      </Col>
      <Col xs={24}>
        <Card title="版本发布计划" style={{ borderRadius: 12 }}>
          <Table
            rowKey="id"
            dataSource={releases}
            pagination={{ pageSize: 5 }}
            columns={[
              { title: '版本', dataIndex: 'version' },
              { title: '状态', dataIndex: 'status', render: (s: string) => getStatusTag(s) },
              {
                title: '发布模板',
                render: (_: any, record: any) => (
                  <Select
                    style={{ width: 180 }}
                    placeholder="选择模板"
                    allowClear
                    options={releaseTemplates.map((t) => ({ label: t.name, value: t.id }))}
                    value={record.release_template_id || undefined}
                    onChange={(value) => assignTemplate(record.id, value)}
                    disabled={isLocked}
                  />
                )
              },
              {
                title: '预约发布',
                render: (_: any, record: any) => {
                  const tpl = templateMap.get(record.release_template_id)
                  return tpl?.schedule_at ? new Date(tpl.schedule_at).toLocaleString() : '-'
                }
              },
              {
                title: '时间窗',
                render: (_: any, record: any) => {
                  const tpl = templateMap.get(record.release_template_id)
                  if (!tpl?.window_start || !tpl?.window_end) return '-'
                  return `${new Date(tpl.window_start).toLocaleString()} ~ ${new Date(tpl.window_end).toLocaleString()}`
                }
              },
              {
                title: '紧急发布',
                render: (_: any, record: any) => {
                  const tpl = templateMap.get(record.release_template_id)
                  return tpl?.emergency ? <Tag color="red">开启</Tag> : <Tag>关闭</Tag>
                }
              }
            ]}
          />
        </Card>
      </Col>
      <Col xs={24}>
        <Card
          title="模板管理"
          style={{ borderRadius: 12 }}
          extra={<Button type="primary" disabled={isLocked} onClick={() => openTemplateModal()}>新建模板</Button>}
        >
          <Table
            rowKey="id"
            dataSource={releaseTemplates}
            pagination={{ pageSize: 5 }}
            columns={[
              { title: '名称', dataIndex: 'name' },
              {
                title: '预约发布',
                render: (_: any, record: any) => record.schedule_at ? new Date(record.schedule_at).toLocaleString() : '-'
              },
              {
                title: '时间窗',
                render: (_: any, record: any) => {
                  if (!record.window_start || !record.window_end) return '-'
                  return `${new Date(record.window_start).toLocaleString()} ~ ${new Date(record.window_end).toLocaleString()}`
                }
              },
              {
                title: '紧急发布',
                render: (_: any, record: any) => record.emergency ? <Tag color="red">开启</Tag> : <Tag>关闭</Tag>
              },
              {
                title: '操作',
                render: (_: any, record: any) => (
                  <Space>
                    <Button size="small" disabled={isLocked} onClick={() => openTemplateModal(record)}>编辑</Button>
                    <Button size="small" danger disabled={isLocked} onClick={() => deleteTemplate(record)}>删除</Button>
                  </Space>
                )
              }
            ]}
          />
        </Card>
      </Col>

      <Modal
        title={editingTemplate ? `编辑模板 - ${editingTemplate.name}` : '新建发布模板'}
        open={templateOpen}
        onOk={saveTemplate}
        onCancel={() => { setTemplateOpen(false); setEditingTemplate(null); templateForm.resetFields() }}
        width={520}
        okText="保存"
        cancelText="取消"
      >
        <Form layout="vertical" form={templateForm} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item shouldUpdate noStyle>
            {({ getFieldValue }) => {
              const emergency = !!getFieldValue('emergency')
              return (
                <>
                  <Form.Item name="schedule_at" label="预约发布">
                    <DatePicker
                      showTime
                      style={{ width: '100%' }}
                      placeholder="选择发布时间"
                      disabled={emergency}
                    />
                  </Form.Item>
                  <Form.Item name="window_range" label="时间窗">
                    <RangePicker
                      showTime
                      style={{ width: '100%' }}
                      placeholder={['开始时间', '结束时间']}
                      disabled={emergency}
                    />
                  </Form.Item>
                  {emergency && (
                    <div style={{ marginBottom: 8 }}>
                      <Text type="secondary">当前发布模式为实时发布暂不可用此功能。</Text>
                    </div>
                  )}
                  <Form.Item name="emergency" label="紧急发布开关" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </>
              )
            }}
          </Form.Item>
        </Form>
      </Modal>
    </Row>
  )
}

import { Alert, Button, Card, Form, Grid, Input, Select, Space, Typography, Upload, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

const maxAttachmentSize = 20 * 1024 * 1024
const maxAttachments = 5

type MemberOption = {
  id: string
  email: string
}

export default function TicketSubmit() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [members, setMembers] = useState<MemberOption[]>([])
  const [fileList, setFileList] = useState<any[]>([])
  const [submitting, setSubmitting] = useState(false)

  const role = (sessionStorage.getItem('role') || '').toLowerCase()
  const canSubmit = role === 'admin' || role === 'owner'
  const orgId = sessionStorage.getItem('org_id') || ''
  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()
  const isPersonal = orgType === 'personal'

  const loadMembers = async () => {
    if (!orgId || isPersonal) return
    try {
      const res = await api.get(`/api/orgs/${orgId}/members`)
      const items = (res.data.items || []).map((m: any) => ({
        id: m.UserID || m.user_id,
        email: m.Email || m.email
      }))
      setMembers(items)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载成员失败')
    }
  }

  useEffect(() => {
    loadMembers()
  }, [orgId, isPersonal])

  const beforeUpload = (file: any) => {
    if (file.size > maxAttachmentSize) {
      message.error('附件大小不能超过 20MB')
      return Upload.LIST_IGNORE
    }
    if (fileList.length >= maxAttachments) {
      message.error(`最多上传 ${maxAttachments} 个附件`)
      return Upload.LIST_IGNORE
    }
    return false
  }

  const submit = async () => {
    if (!canSubmit) {
      message.error('当前账号无权限提交工单')
      return
    }
    try {
      const values = await form.validateFields()
      if (fileList.length > maxAttachments) {
        message.error(`最多上传 ${maxAttachments} 个附件`)
        return
      }
      const oversized = fileList.find((file) => (file.size || file.originFileObj?.size || 0) > maxAttachmentSize)
      if (oversized) {
        message.error('附件大小不能超过 20MB')
        return
      }
      const formData = new FormData()
      formData.append('title', values.title.trim())
      formData.append('description', values.description || '')
      const assigneeType = isPersonal ? 'system' : values.assignee_type
      formData.append('assignee_type', assigneeType)
      if (!isPersonal && assigneeType === 'user') {
        formData.append('assignee_user_id', values.assignee_user_id)
      }
      fileList.forEach((file) => {
        const raw = file.originFileObj || file
        formData.append('attachments', raw)
      })
      setSubmitting(true)
      await api.post('/api/tickets', formData)
      message.success('工单已提交')
      navigate('/tickets')
    } catch (err: any) {
      if (err?.errorFields) return
      message.error(err?.response?.data?.error || '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>提交工单</Title>
        <Text type="secondary">提交问题并派发到系统管理员或子用户</Text>
      </Space>

      {!canSubmit && (
        <Alert
          type="warning"
          message="当前账号无权限提交工单，仅管理员可提交。"
          style={{ marginBottom: 16 }}
        />
      )}

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Form layout="vertical" form={form} initialValues={{ assignee_type: 'system' }} disabled={!canSubmit}>
          <Form.Item
            name="title"
            label="工单标题"
            rules={[{ required: true, message: '请输入工单标题' }]}
          >
            <Input placeholder="请输入标题" maxLength={100} />
          </Form.Item>
          <Form.Item name="description" label="工单描述">
            <Input.TextArea rows={5} placeholder="请输入详细描述" />
          </Form.Item>
          {!isPersonal && (
            <>
              <Form.Item
                name="assignee_type"
                label="派发对象"
                rules={[{ required: true, message: '请选择派发对象' }]}
              >
                <Select>
                  <Option value="system">系统管理员</Option>
                  <Option value="user">子用户</Option>
                </Select>
              </Form.Item>
              <Form.Item shouldUpdate noStyle>
                {({ getFieldValue }) => {
                  if (getFieldValue('assignee_type') !== 'user') return null
                  return (
                    <Form.Item
                      name="assignee_user_id"
                      label="选择子用户"
                      rules={[{ required: true, message: '请选择子用户' }]}
                    >
                      <Select
                        placeholder="请选择子用户"
                        showSearch
                        optionFilterProp="children"
                      >
                        {members.map((member) => (
                          <Option key={member.id} value={member.id}>{member.email}</Option>
                        ))}
                      </Select>
                    </Form.Item>
                  )
                }}
              </Form.Item>
            </>
          )}
          <Form.Item label="附件上传">
            <Upload
              multiple
              beforeUpload={beforeUpload}
              fileList={fileList}
              onChange={({ fileList: nextList }) => setFileList(nextList)}
            >
              <Button icon={<UploadOutlined />} style={isMobile ? { width: '100%' } : undefined}>选择附件</Button>
            </Upload>
            <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
              最多 {maxAttachments} 个附件，单个文件不超过 20MB
            </Text>
          </Form.Item>
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Button onClick={() => navigate('/tickets')} style={isMobile ? { width: '100%' } : undefined}>返回列表</Button>
            <Button
              type="primary"
              onClick={submit}
              loading={submitting}
              disabled={!canSubmit}
              style={isMobile ? { width: '100%' } : undefined}
            >
              提交工单
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  )
}

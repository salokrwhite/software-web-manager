import { Button, Card, Form, Grid, Input, Select, Space, Typography, Upload, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

const maxAttachmentSize = 20 * 1024 * 1024
const maxAttachments = 5

type OrgItem = {
  id: string
  name: string
}

export default function SystemTicketSubmit() {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [form] = Form.useForm()
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [fileList, setFileList] = useState<any[]>([])
  const [submitting, setSubmitting] = useState(false)

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

  useEffect(() => {
    loadOrgs()
  }, [])

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
      formData.append('org_id', values.org_id)
      formData.append('title', values.title.trim())
      formData.append('description', values.description || '')
      fileList.forEach((file) => {
        const raw = file.originFileObj || file
        formData.append('attachments', raw)
      })
      setSubmitting(true)
      await api.post('/api/system/tickets', formData)
      message.success('工单已提交')
      navigate('/system/tickets')
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
        <Text type="secondary">系统管理员提交工单并派发到系统队列</Text>
      </Space>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Form layout="vertical" form={form}>
          <Form.Item
            name="org_id"
            label="选择企业"
            rules={[{ required: true, message: '请选择企业' }]}
          >
            <Select placeholder="请选择企业">
              {orgs.map((org) => (
                <Option key={org.id} value={org.id}>{org.name}</Option>
              ))}
            </Select>
          </Form.Item>
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
              默认派发至系统队列，最多 {maxAttachments} 个附件，单个文件不超过 20MB
            </Text>
          </Form.Item>
          <Space direction={isMobile ? 'vertical' : 'horizontal'} style={isMobile ? { width: '100%' } : undefined}>
            <Button onClick={() => navigate('/system/tickets')} style={isMobile ? { width: '100%' } : undefined}>返回列表</Button>
            <Button type="primary" onClick={submit} loading={submitting} style={isMobile ? { width: '100%' } : undefined}>
              提交工单
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  )
}

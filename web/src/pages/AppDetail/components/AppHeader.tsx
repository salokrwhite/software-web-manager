import {
  Button,
  Form,
  Input,
  Modal,
  Row,
  Col,
  Grid,
  Space,
  Tooltip,
  Typography,
  message
} from 'antd'
import {
  ArrowLeftOutlined,
  EditOutlined,
  KeyOutlined,
  RocketOutlined
} from '@ant-design/icons'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../../../api/client'

const { Title, Text } = Typography

type AppHeaderProps = {
  app: any
  isLocked: boolean
  canEdit: boolean
  lockReason?: 'pending' | 'rejected' | ''
  onReload: () => void
}

export default function AppHeader({ app, isLocked, canEdit, lockReason = '', onReload }: AppHeaderProps) {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [editOpen, setEditOpen] = useState(false)
  const [publicKeyOpen, setPublicKeyOpen] = useState(false)
  const [editForm] = Form.useForm()
  const [publicKeyForm] = Form.useForm()

  const lockTip = lockReason === 'rejected'
    ? '已驳回，仅可编辑'
    : lockReason === 'pending'
      ? '待审核，无法操作'
      : ''

  const updateAppInfo = async () => {
    if (!canEdit) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await editForm.validateFields()
      await api.patch(`/api/apps/${app.id}`, values)
      message.success('保存成功')
      setEditOpen(false)
      editForm.resetFields()
      onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  const updatePublicKey = async () => {
    if (!canEdit) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await publicKeyForm.validateFields()
      await api.patch(`/api/apps/${app.id}`, {
        public_key: values.public_key || ''
      })
      message.success('公钥已更新')
      setPublicKeyOpen(false)
      publicKeyForm.resetFields()
      onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    }
  }

  return (
    <>
      <Row
        justify="space-between"
        align={isMobile ? 'top' : 'middle'}
        style={{ marginBottom: isMobile ? 16 : 24 }}
        gutter={isMobile ? [12, 12] : undefined}
      >
        <Col xs={24} lg={12}>
          <Space
            size={isMobile ? 10 : 16}
            direction={isMobile ? 'vertical' : 'horizontal'}
            style={isMobile ? { width: '100%' } : undefined}
          >
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/apps')} style={isMobile ? { width: '100%' } : undefined}>
              返回
            </Button>
            <Space direction="vertical" size={4}>
              <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>{app.name}</Title>
              <Text type="secondary" style={{ fontSize: isMobile ? 12 : 14 }}>
                {app.slug} · 创建于 {new Date(app.created_at).toLocaleDateString()}
              </Text>
            </Space>
          </Space>
        </Col>
        <Col xs={24} lg={12}>
          <Space
            direction={isMobile ? 'vertical' : 'horizontal'}
            style={isMobile ? { width: '100%' } : { width: '100%', justifyContent: 'flex-end' }}
            size={isMobile ? 8 : 8}
            wrap={!isMobile}
          >
            <Tooltip title={!canEdit ? '待审核，无法操作' : ''}>
              <Button
                icon={<EditOutlined />}
                disabled={!canEdit}
                style={isMobile ? { width: '100%' } : undefined}
                onClick={() => {
                  setEditOpen(true)
                  editForm.setFieldsValue({
                    name: app.name || '',
                    slug: app.slug || '',
                    description: app.description || ''
                  })
                }}
              >
                编辑
              </Button>
            </Tooltip>
            <Tooltip title={!canEdit ? '待审核，无法操作' : ''}>
              <Button
                icon={<KeyOutlined />}
                disabled={!canEdit}
                style={isMobile ? { width: '100%' } : undefined}
                onClick={() => {
                  setPublicKeyOpen(true)
                  publicKeyForm.setFieldsValue({
                    public_key: app.public_key || ''
                  })
                }}
              >
                设置公钥
              </Button>
            </Tooltip>
            <Tooltip title={isLocked ? lockTip : ''}>
              <Button
                type="primary"
                icon={<RocketOutlined />}
                disabled={isLocked}
                style={isMobile ? { width: '100%' } : undefined}
                onClick={() => navigate(`/apps/${app.id}/releases/new`)}
              >
                发布新版本
              </Button>
            </Tooltip>
          </Space>
        </Col>
      </Row>

      <Modal
        title="编辑应用"
        open={editOpen}
        onOk={updateAppInfo}
        onCancel={() => { setEditOpen(false); editForm.resetFields() }}
        okText="确定"
        cancelText="取消"
        width={isMobile ? 'calc(100vw - 32px)' : 520}
      >
        <Form layout="vertical" form={editForm} style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="应用名称"
            rules={[{ required: true, message: '请输入应用名称' }]}
          >
            <Input placeholder="例如：MyApp" />
          </Form.Item>
          <Form.Item
            name="slug"
            label="应用标识"
            rules={[
              { required: true, message: '请输入应用标识' },
              { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符' }
            ]}
          >
            <Input placeholder="例如：my-app" />
          </Form.Item>
          <Form.Item name="description" label="应用描述">
            <Input.TextArea rows={3} placeholder="简要描述您的应用" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="设置公钥"
        open={publicKeyOpen}
        onOk={updatePublicKey}
        onCancel={() => { setPublicKeyOpen(false); publicKeyForm.resetFields() }}
        okText="保存"
        cancelText="取消"
        width={isMobile ? 'calc(100vw - 32px)' : 560}
      >
        <Form layout="vertical" form={publicKeyForm} style={{ marginTop: 16 }}>
          <Form.Item name="public_key" label="公钥（Ed25519）">
            <Input.TextArea rows={5} placeholder="用于客户端更新包签名验签，可留空" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

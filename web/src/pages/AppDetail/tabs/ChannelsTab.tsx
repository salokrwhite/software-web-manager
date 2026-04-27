import {
  Avatar,
  Button,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme
} from 'antd'
import { DeleteOutlined, GlobalOutlined, PlusOutlined } from '@ant-design/icons'
import { useState } from 'react'
import api from '../../../api/client'

const { Text } = Typography

type ChannelsTabProps = {
  appId: string
  channels: any[]
  isLocked: boolean
  reload: () => void
}

function ChannelsTab({ appId, channels, isLocked, reload }: ChannelsTabProps) {
  const { token } = theme.useToken()
  const [channelOpen, setChannelOpen] = useState(false)
  const [channelForm] = Form.useForm()

  const createChannel = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await channelForm.validateFields()
      await api.post(`/api/apps/${appId}/channels`, values)
      message.success('创建成功')
      setChannelOpen(false)
      channelForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    }
  }

  const deleteChannel = (record: any) => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    if (record?.is_default) {
      message.warning('默认渠道不能删除')
      return
    }
    Modal.confirm({
      title: '确认删除渠道',
      content: `确定要删除渠道 ${record?.name || ''} 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      onOk: async () => {
        try {
          await api.delete(`/api/apps/${appId}/channels/${record.id}`)
          message.success('删除成功')
          reload()
        } catch (err: any) {
          const code = err?.response?.data?.error
          if (code === 'default_channel_cannot_delete') {
            message.warning('默认渠道不能删除')
            return
          }
          if (code === 'channel_last_one_cannot_delete') {
            message.warning('至少保留一个渠道')
            return
          }
          if (code === 'channel_in_use') {
            message.warning('渠道已被发布策略使用，不能删除')
            return
          }
          message.error(code || '删除失败')
        }
      }
    })
  }

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <Text type="secondary">管理应用分发渠道</Text>
        </Col>
        <Col>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setChannelOpen(true)} disabled={isLocked}>
              新建渠道
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <Table
        rowKey="id"
        dataSource={channels}
        pagination={false}
        columns={[
          {
            title: '渠道名称',
            dataIndex: 'name',
            render: (n: string) => (
              <Space>
                <Avatar size="small" icon={<GlobalOutlined />} style={{ background: token.colorSuccess }} />
                <Text strong>{n}</Text>
              </Space>
            )
          },
          { title: 'Code', dataIndex: 'code', render: (c: string) => <Tag>{c}</Tag> },
          {
            title: '默认渠道',
            dataIndex: 'is_default',
            render: (v: boolean) => v ? <Tag color="success">是</Tag> : <Tag>否</Tag>
          },
          { title: '最低版本', dataIndex: 'min_supported_version' },
          {
            title: '操作',
            render: (_: any, record: any) => {
              const onlyOneLeft = channels.length <= 1
              const disabled = isLocked || !!record?.is_default || onlyOneLeft
              let tip = ''
              if (isLocked) {
                tip = '待审核，无法操作'
              } else if (record?.is_default) {
                tip = '默认渠道不可删除'
              } else if (onlyOneLeft) {
                tip = '至少保留一个渠道'
              }
              return (
                <Tooltip title={tip}>
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    disabled={disabled}
                    onClick={() => deleteChannel(record)}
                  >
                    删除
                  </Button>
                </Tooltip>
              )
            }
          }
        ]}
      />

      <Modal
        title="新建渠道"
        open={channelOpen}
        onOk={createChannel}
        onCancel={() => { setChannelOpen(false); channelForm.resetFields() }}
        width={480}
      >
        <Form layout="vertical" form={channelForm} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="渠道名称" rules={[{ required: true }]}>
            <Input placeholder="例如：App Store" size="large" />
          </Form.Item>
          <Form.Item name="code" label="渠道代码" rules={[{ required: true }]}>
            <Input placeholder="例如：app-store" size="large" />
          </Form.Item>
          <Form.Item name="min_supported_version" label="最低支持版本">
            <Input placeholder="例如：1.0.0" size="large" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export default ChannelsTab

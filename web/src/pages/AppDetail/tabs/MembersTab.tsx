import {
  Button,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tooltip,
  Typography,
  message
} from 'antd'
import { TeamOutlined } from '@ant-design/icons'
import { useState } from 'react'
import api from '../../../api/client'

const { Text } = Typography
const { Option } = Select

type MembersTabProps = {
  appId: string
  appMembers: any[]
  isLocked: boolean
  reload: () => void
}

export default function MembersTab({ appId, appMembers, isLocked, reload }: MembersTabProps) {
  const [appMemberOpen, setAppMemberOpen] = useState(false)
  const [appMemberForm] = Form.useForm()

  const addAppMember = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await appMemberForm.validateFields()
      await api.post(`/api/apps/${appId}/members`, values)
      message.success('添加成功')
      setAppMemberOpen(false)
      appMemberForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '添加失败')
    }
  }

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <Text type="secondary">应用成员与角色</Text>
        </Col>
        <Col>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button type="primary" icon={<TeamOutlined />} onClick={() => setAppMemberOpen(true)} disabled={isLocked}>
              添加成员
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <Table
        rowKey={(row) => `${row.app_id}-${row.user_id}`}
        dataSource={appMembers}
        pagination={false}
        columns={[
          { title: '用户ID', dataIndex: 'user_id' },
          { title: '角色', dataIndex: 'role' },
          { title: '加入时间', dataIndex: 'created_at', render: (d: string) => d ? new Date(d).toLocaleDateString() : '-' }
        ]}
      />

      <Modal
        title="添加应用成员"
        open={appMemberOpen}
        onOk={addAppMember}
        onCancel={() => { setAppMemberOpen(false); appMemberForm.resetFields() }}
        width={480}
      >
        <Form layout="vertical" form={appMemberForm} style={{ marginTop: 16 }}>
          <Form.Item name="user_email" label="邮箱" rules={[{ required: true }]}>
            <Input placeholder="成员邮箱" size="large" />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true }]}>
            <Select>
              <Option value="admin">admin</Option>
              <Option value="dev">dev</Option>
              <Option value="viewer">viewer</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

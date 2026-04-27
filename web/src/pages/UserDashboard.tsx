import { Card, Space, Typography } from 'antd'

const { Title, Text } = Typography

export default function UserDashboard() {
  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>仪表盘</Title>
        <Text type="secondary">个人主页</Text>
      </Space>

      <Card>
        <Space direction="vertical" size={8}>
          <Title level={5} style={{ margin: 0 }}>尚未加入企业组织</Title>
          <Text type="secondary">
            您当前未加入任何企业组织，请联系企业管理员邀请加入后即可使用完整功能。
          </Text>
        </Space>
      </Card>
    </div>
  )
}

import { Card, Empty, Space, Typography } from 'antd'

const { Title, Text } = Typography

function MaintenanceTab({
  appId,
  app,
  isLocked
}: {
  appId: string
  app: any
  isLocked: boolean
}) {
  // 预留页面：维护模式功能后续实现，这里仅占位。
  void appId
  void app
  void isLocked

  return (
    <Card style={{ borderRadius: 12 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          <Title level={5} style={{ marginBottom: 4 }}>维护模式</Title>
          <Text type="secondary">用于在升级、检修期间临时停服并向客户端下发维护提示</Text>
        </div>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={<Text type="secondary">功能开发中，敬请期待</Text>}
        />
      </Space>
    </Card>
  )
}

export { MaintenanceTab }
export default MaintenanceTab

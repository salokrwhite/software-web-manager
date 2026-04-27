import { Card, Col, Row, Statistic, theme, Grid } from 'antd'
import { AppstoreOutlined, GlobalOutlined, KeyOutlined, RocketOutlined } from '@ant-design/icons'

type AppStatsProps = {
  releasesCount: number
  channelsCount: number
  appSecretsCount: number
  activeReleaseChannelsCount: number
}

export function AppStats({
  releasesCount,
  channelsCount,
  appSecretsCount,
  activeReleaseChannelsCount
}: AppStatsProps) {
  const { token } = theme.useToken()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg

  return (
    <Row gutter={isMobile ? [12, 12] : [24, 24]} style={{ marginBottom: isMobile ? 16 : 24 }}>
      <Col xs={24} sm={12} lg={6}>
        <Card
          style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          styles={{ body: { padding: isMobile ? 16 : 24 } }}
        >
          <Statistic
            title="总版本数"
            value={releasesCount}
            prefix={<RocketOutlined style={{ color: token.colorPrimary }} />}
            valueStyle={{ fontSize: isMobile ? 22 : 24 }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card
          style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          styles={{ body: { padding: isMobile ? 16 : 24 } }}
        >
          <Statistic
            title="渠道数"
            value={channelsCount}
            prefix={<GlobalOutlined style={{ color: token.colorSuccess }} />}
            valueStyle={{ fontSize: isMobile ? 22 : 24 }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card
          style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          styles={{ body: { padding: isMobile ? 16 : 24 } }}
        >
          <Statistic
            title="应用密钥"
            value={appSecretsCount}
            prefix={<KeyOutlined style={{ color: token.colorInfo }} />}
            valueStyle={{ fontSize: isMobile ? 22 : 24 }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card
          style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          styles={{ body: { padding: isMobile ? 16 : 24 } }}
        >
          <Statistic
            title="发布渠道"
            value={activeReleaseChannelsCount}
            prefix={<AppstoreOutlined style={{ color: token.colorWarning }} />}
            valueStyle={{ fontSize: isMobile ? 22 : 24 }}
          />
        </Card>
      </Col>
    </Row>
  )
}

export default AppStats

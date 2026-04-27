import { Card, Col, Row, Typography } from 'antd'

const { Text } = Typography

type AttributesTabProps = {
  app: any
}

export default function AttributesTab({ app }: AttributesTabProps) {
  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} sm={12}>
        <Card size="small" title="应用ID">
          <Text>{app?.id || '-'}</Text>
        </Card>
      </Col>
      <Col xs={24} sm={12}>
        <Card size="small" title="创建时间">
          <Text>{app?.created_at ? new Date(app.created_at).toLocaleString() : '-'}</Text>
        </Card>
      </Col>
    </Row>
  )
}

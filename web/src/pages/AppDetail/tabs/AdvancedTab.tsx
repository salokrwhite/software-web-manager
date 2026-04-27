import { Button, Card, Col, InputNumber, Row, Space, Switch, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import api from '../../../api/client'

const { Text, Title } = Typography

function AdvancedTab({
  appId,
  app,
  isLocked,
  onReload
}: {
  appId: string
  app: any
  isLocked: boolean
  onReload?: () => void
}) {
  const [feedbackEnabled, setFeedbackEnabled] = useState(false)
  const [onlineEnabled, setOnlineEnabled] = useState(false)
  const [heartbeatInterval, setHeartbeatInterval] = useState<number>(60)
  const [savingFeedback, setSavingFeedback] = useState(false)
  const [savingOnline, setSavingOnline] = useState(false)
  const [savingHeartbeat, setSavingHeartbeat] = useState(false)

  useEffect(() => {
    if (!appId) return
    setFeedbackEnabled(app?.feedback_enabled === true)
    setOnlineEnabled(app?.online_enabled === true)
    setHeartbeatInterval(app?.heartbeat_interval_seconds ?? 60)
  }, [appId, app?.feedback_enabled, app?.online_enabled, app?.heartbeat_interval_seconds])

  const handleToggleFeedback = async (checked: boolean) => {
    if (!appId) return
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    setFeedbackEnabled(checked)
    setSavingFeedback(true)
    try {
      await api.patch(`/api/apps/${appId}`, { feedback_enabled: checked })
      message.success('已更新')
      if (onReload) onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
      setFeedbackEnabled(app?.feedback_enabled === true)
    } finally {
      setSavingFeedback(false)
    }
  }

  const handleToggleOnline = async (checked: boolean) => {
    if (!appId) return
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    setOnlineEnabled(checked)
    setSavingOnline(true)
    try {
      await api.patch(`/api/apps/${appId}`, { online_enabled: checked })
      message.success('已更新')
      if (onReload) onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '更新失败')
      setOnlineEnabled(app?.online_enabled === true)
    } finally {
      setSavingOnline(false)
    }
  }

  const saveHeartbeatInterval = async () => {
    if (!appId) return
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    setSavingHeartbeat(true)
    try {
      await api.patch(`/api/apps/${appId}`, { heartbeat_interval_seconds: heartbeatInterval })
      message.success('已保存')
      if (onReload) onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
      setHeartbeatInterval(app?.heartbeat_interval_seconds ?? 60)
    } finally {
      setSavingHeartbeat(false)
    }
  }

  return (
    <Card style={{ borderRadius: 12 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          <Title level={5} style={{ marginBottom: 4 }}>高级选项</Title>
          <Text type="secondary">用于控制应用的高级功能开关</Text>
        </div>
        <Row gutter={[16, 16]}>
          <Col xs={24}>
            <Card size="small" style={{ borderRadius: 8 }}>
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Row align="middle" justify="space-between">
                  <Col>
                    <Space direction="vertical" size={2}>
                      <Text strong>实时在线设备</Text>
                      <Text type="secondary">控制在线设备统计与实时展示</Text>
                    </Space>
                  </Col>
                  <Col>
                    <Switch checked={onlineEnabled} loading={savingOnline} onChange={handleToggleOnline} disabled={isLocked} />
                  </Col>
                </Row>
                <div>
                  <Text strong>心跳配置（秒）</Text>
                  <div style={{ marginTop: 4 }}>
                    <Text type="secondary">建议 30–60 秒，过于频繁会增加服务压力。</Text>
                  </div>
                  <Space style={{ marginTop: 8 }}>
                    <Space.Compact>
                      <InputNumber
                        min={10}
                        max={3600}
                        value={heartbeatInterval}
                        onChange={(value) => {
                          if (typeof value === 'number') {
                            setHeartbeatInterval(value)
                          }
                        }}
                        disabled={isLocked}
                      />
                      <Button
                        disabled
                        style={{
                          pointerEvents: 'none',
                          color: 'rgba(0, 0, 0, 0.65)',
                          background: '#fafafa',
                          cursor: 'default'
                        }}
                      >
                        秒
                      </Button>
                    </Space.Compact>
                    <Button type="primary" loading={savingHeartbeat} onClick={saveHeartbeatInterval} disabled={isLocked}>
                      保存
                    </Button>
                  </Space>
                </div>
              </Space>
            </Card>
          </Col>
          <Col xs={24}>
            <Card size="small" style={{ borderRadius: 8 }}>
              <Row align="middle" justify="space-between">
                <Col>
                  <Space direction="vertical" size={2}>
                    <Text strong>用户反馈</Text>
                    <Text type="secondary">控制 SDK 上报的用户反馈是否启用</Text>
                  </Space>
                </Col>
                <Col>
                  <Switch checked={feedbackEnabled} loading={savingFeedback} onChange={handleToggleFeedback} disabled={isLocked} />
                </Col>
              </Row>
            </Card>
          </Col>
        </Row>
      </Space>
    </Card>
  )
}

export { AdvancedTab }
export default AdvancedTab

import { Button, Card, DatePicker, Input, Space, Tag, Typography, message } from 'antd'
import dayjs, { Dayjs } from 'dayjs'
import { useEffect, useState } from 'react'
import api from '../../../api/client'

const { Title, Text } = Typography

function MaintenanceTab({
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
  const [enabled, setEnabled] = useState(false)
  const [startAt, setStartAt] = useState<Dayjs | null>(null)
  const [maintenanceMessage, setMaintenanceMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const [now, setNow] = useState(dayjs())

  useEffect(() => {
    setEnabled(app?.maintenance_enabled === true)
    setStartAt(app?.maintenance_start_at ? dayjs(app.maintenance_start_at) : null)
    setMaintenanceMessage(app?.maintenance_message || '')
  }, [appId, app?.maintenance_enabled, app?.maintenance_start_at, app?.maintenance_message])

  useEffect(() => {
    const timer = setInterval(() => setNow(dayjs()), 1000)
    return () => clearInterval(timer)
  }, [])

  const status: 'off' | 'scheduled' | 'active' = !enabled
    ? 'off'
    : startAt && startAt.isAfter(now)
      ? 'scheduled'
      : 'active'

  const countdownText = () => {
    if (status !== 'scheduled' || !startAt) return ''
    const diff = Math.max(0, startAt.diff(now, 'second'))
    const h = Math.floor(diff / 3600)
    const m = Math.floor((diff % 3600) / 60)
    const s = diff % 60
    if (h > 0) return `${h} 小时 ${m} 分 ${s} 秒`
    if (m > 0) return `${m} 分 ${s} 秒`
    return `${s} 秒`
  }

  const handleSave = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    if (enabled && !startAt) {
      message.warning('请先选择维护开始时间')
      return
    }
    if (maintenanceMessage.length > 500) {
      message.warning('提示文案不能超过 500 字')
      return
    }
    setSaving(true)
    try {
      await api.patch(`/api/apps/${appId}`, {
        maintenance_enabled: enabled,
        maintenance_start_at: enabled && startAt ? startAt.toISOString() : '',
        maintenance_message: maintenanceMessage
      })
      message.success('维护模式已保存')
      if (onReload) onReload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const statusTag = status === 'off'
    ? <Tag>未开启</Tag>
    : status === 'scheduled'
      ? <Tag color="processing">已排期 · {countdownText()} 后进入维护</Tag>
      : <Tag color="error">维护中</Tag>

  return (
    <Card style={{ borderRadius: 12 }}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div>
          <Title level={5} style={{ marginBottom: 4 }}>维护模式</Title>
          <Text type="secondary">
            开启后客户端会弹窗提示「X 后进入维护」并倒计时；到点后客户端自动退出，用户重新打开时会看到「系统维护中」提示。
          </Text>
        </div>

        <Space>
          <Text strong>当前状态：</Text>
          {statusTag}
        </Space>

        <Space align="center" size={12}>
          <Button
            type={enabled ? 'primary' : 'default'}
            danger={enabled}
            disabled={isLocked}
            onClick={() => setEnabled((v) => !v)}
          >
            {enabled ? '关闭维护模式' : '开启维护模式'}
          </Button>
          <Text type="secondary">{enabled ? '维护模式已开启（保存后生效）' : '维护模式未开启'}</Text>
        </Space>

        <div>
          <Text strong>维护开始时间</Text>
          <div style={{ marginTop: 4, marginBottom: 8 }}>
            <Text type="secondary">客户端会按此时间倒计时，到点自动退出。</Text>
          </div>
          <DatePicker
            showTime
            value={startAt}
            disabled={isLocked || !enabled}
            onChange={(value) => setStartAt(value)}
            format="YYYY-MM-DD HH:mm:ss"
            placeholder="选择维护开始时间"
            style={{ width: 260 }}
            disabledDate={(d) => d && d.isBefore(dayjs().startOf('day'))}
          />
        </div>

        <div>
          <Text strong>提示文案</Text>
          <div style={{ marginTop: 4, marginBottom: 8 }}>
            <Text type="secondary">向用户展示的维护说明（≤ 500 字，可选）。</Text>
          </div>
          <Input.TextArea
            rows={3}
            maxLength={500}
            showCount
            value={maintenanceMessage}
            disabled={isLocked}
            onChange={(e) => setMaintenanceMessage(e.target.value)}
            placeholder="例如：系统将于今晚进行升级维护，预计 30 分钟，请提前保存。"
          />
        </div>

        <Space>
          <Button type="primary" loading={saving} onClick={handleSave} disabled={isLocked}>
            保存
          </Button>
        </Space>
      </Space>
    </Card>
  )
}

export { MaintenanceTab }
export default MaintenanceTab

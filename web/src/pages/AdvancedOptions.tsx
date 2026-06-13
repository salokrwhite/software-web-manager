import { Button, Card, Col, Grid, Input, Modal, Row, Select, Space, Table, Tabs, Tag, Typography, message } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

type AppItem = {
  id: string
  name: string
  online_enabled?: boolean
}

type OnlineDeviceItem = {
  id: string
  device_id: string
  platform: string
  arch: string
  os_version: string
  country: string
  app_version: string
  user_id: string
  last_ip: string
  last_seen_at: string
}

type BlockedDeviceItem = {
  id: string
  app_id: string
  device_id: string
  reason?: string
  blocked_at?: string
  blocked_by?: string
}

export default function AdvancedOptions() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [apps, setApps] = useState<AppItem[]>([])
  const [appId, setAppId] = useState<string>('')
  const [loadingApps, setLoadingApps] = useState(false)
  const [onlineCount, setOnlineCount] = useState<number | null>(null)
  const [onlineWindow, setOnlineWindow] = useState<number>(120)
  const [onlineStatus, setOnlineStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const [onlineUpdatedAt, setOnlineUpdatedAt] = useState<string>('')
  const [deviceItems, setDeviceItems] = useState<OnlineDeviceItem[]>([])
  const [deviceTotal, setDeviceTotal] = useState(0)
  const [devicePage, setDevicePage] = useState(1)
  const [devicePageSize, setDevicePageSize] = useState(10)
  const [deviceLoading, setDeviceLoading] = useState(false)
  const [blockingId, setBlockingId] = useState<string>('')
  const [batchBlocking, setBatchBlocking] = useState(false)
  const [selectedOnlineIds, setSelectedOnlineIds] = useState<string[]>([])
  const [blockedItems, setBlockedItems] = useState<BlockedDeviceItem[]>([])
  const [blockedTotal, setBlockedTotal] = useState(0)
  const [blockedPage, setBlockedPage] = useState(1)
  const [blockedPageSize, setBlockedPageSize] = useState(10)
  const [blockedLoading, setBlockedLoading] = useState(false)
  const [unblockingId, setUnblockingId] = useState<string>('')
  const [selectedBlockedIds, setSelectedBlockedIds] = useState<string[]>([])
  const [batchUnblocking, setBatchUnblocking] = useState(false)
  const [blockedKeyword, setBlockedKeyword] = useState('')
  const [manualBlockDeviceID, setManualBlockDeviceID] = useState('')
  const [manualBlockReason, setManualBlockReason] = useState('')
  const [addingBlocked, setAddingBlocked] = useState(false)
  const selectedApp = useMemo(() => apps.find((app) => app.id === appId), [apps, appId])
  const onlineEnabled = selectedApp?.online_enabled !== false

  const loadApps = async () => {
    setLoadingApps(true)
    try {
      const res = await api.get('/api/apps')
      const items = (res.data.items || []).map((raw: any) => ({
        id: raw.ID || raw.id,
        name: raw.Name || raw.name,
        online_enabled: raw.OnlineEnabled ?? raw.online_enabled ?? true
      }))
      setApps(items)
      if (!appId && items.length > 0) {
        setAppId(items[0].id)
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载应用失败')
    } finally {
      setLoadingApps(false)
    }
  }

  useEffect(() => {
    loadApps()
  }, [])

  useEffect(() => {
    if (!appId) return
    if (!onlineEnabled) {
      setOnlineCount(0)
      setOnlineStatus('disconnected')
      return
    }
    let canceled = false
    let source: EventSource | null = null
    let reconnectTimer: number | null = null

    const loadOnline = async () => {
      try {
        const res = await api.get(`/api/apps/${appId}/online`)
        if (canceled) return
        setOnlineCount(typeof res.data?.online === 'number' ? res.data.online : 0)
        if (typeof res.data?.window_seconds === 'number') {
          setOnlineWindow(res.data.window_seconds)
        }
        if (res.data?.server_time) {
          setOnlineUpdatedAt(res.data.server_time)
        }
      } catch (err: any) {
        if (!canceled) {
          message.error(err?.response?.data?.error || '加载在线设备失败')
        }
      }
    }

    const clearReconnectTimer = () => {
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    }

    const connectStream = async () => {
      clearReconnectTimer()
      if (source) {
        source.close()
        source = null
      }
      setOnlineStatus('connecting')
      try {
        const tokenRes = await api.get(`/api/apps/${appId}/online/stream-token`)
        if (canceled) return
        const streamToken = String(tokenRes?.data?.stream_token || '').trim()
        if (!streamToken) {
          setOnlineStatus('disconnected')
          return
        }
        const base = import.meta.env.VITE_API_BASE || 'http://localhost:8080'
        source = new EventSource(`${base}/api/apps/${appId}/online/stream?stream_token=${encodeURIComponent(streamToken)}`)
        source.onopen = () => {
          if (!canceled) setOnlineStatus('connected')
        }
        source.onmessage = (evt) => {
          if (canceled) return
          try {
            const data = JSON.parse(evt.data || '{}')
            if (typeof data.online === 'number') {
              setOnlineCount(data.online)
            }
            if (typeof data.window_seconds === 'number') {
              setOnlineWindow(data.window_seconds)
            }
            if (data.server_time) {
              setOnlineUpdatedAt(data.server_time)
            }
          } catch {
            // ignore malformed message
          }
        }
        source.onerror = () => {
          if (canceled) return
          setOnlineStatus('disconnected')
          if (source) {
            source.close()
            source = null
          }
          clearReconnectTimer()
          reconnectTimer = window.setTimeout(() => {
            void connectStream()
          }, 2000)
        }
      } catch {
        if (canceled) return
        setOnlineStatus('disconnected')
        clearReconnectTimer()
        reconnectTimer = window.setTimeout(() => {
          void connectStream()
        }, 5000)
      }
    }

    loadOnline()
    void connectStream()

    return () => {
      canceled = true
      clearReconnectTimer()
      if (source) source.close()
    }
  }, [appId, onlineEnabled])

  const loadOnlineDevices = async (page = devicePage, pageSize = devicePageSize) => {
    if (!appId) return
    setDeviceLoading(true)
    try {
      const res = await api.get(`/api/apps/${appId}/online/devices`, {
        params: { page, page_size: pageSize }
      })
      setDeviceItems(res.data.items || [])
      setDeviceTotal(res.data.total || 0)
      setSelectedOnlineIds([])
      if (typeof res.data?.window_seconds === 'number') {
        setOnlineWindow(res.data.window_seconds)
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载在线设备失败')
    } finally {
      setDeviceLoading(false)
    }
  }

  const loadBlockedDevices = async (page = blockedPage, pageSize = blockedPageSize, keyword = blockedKeyword) => {
    if (!appId) return
    setBlockedLoading(true)
    try {
      const res = await api.get(`/api/apps/${appId}/blocked-devices`, {
        params: { page, page_size: pageSize, q: keyword || undefined }
      })
      setBlockedItems(res.data.items || [])
      setBlockedTotal(res.data.total || 0)
      setSelectedBlockedIds([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载已禁用设备失败')
    } finally {
      setBlockedLoading(false)
    }
  }

  const getErrorMessage = (err: any, fallback: string) => {
    const payload = err?.response?.data?.error
    if (typeof payload === 'string' && payload.trim() !== '') return payload
    if (payload && typeof payload?.message === 'string' && payload.message.trim() !== '') return payload.message
    return fallback
  }

  const confirmBlockOnlineDevices = (items: OnlineDeviceItem[]) => {
    if (!appId || items.length === 0) return
    const isBatch = items.length > 1
    let reasonInput = ''
    Modal.confirm({
      title: isBatch ? `确认移除并下线这 ${items.length} 台设备？` : '确认移除并下线该设备？',
      content: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            {isBatch
              ? `选中的 ${items.length} 台设备将被永久禁用，客户端会收到下线指令并退出。`
              : `设备 ${items[0].device_id} 将被永久禁用，客户端会收到下线指令并退出。`}
          </div>
          <Input.TextArea
            rows={3}
            placeholder="请输入禁用原因（可选，不填默认 manual_remove）"
            onChange={(e) => {
              reasonInput = e.target.value
            }}
          />
        </Space>
      ),
      okText: isBatch ? '确认批量下线' : '确认下线',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        if (isBatch) {
          setBatchBlocking(true)
        } else {
          setBlockingId(items[0].id)
        }
        try {
          const reason = reasonInput?.trim() || 'manual_remove'
          const results = await Promise.allSettled(
            items.map((item) => api.post(`/api/devices/${item.id}/block`, { app_id: appId, reason }))
          )
          const successCount = results.filter((result) => result.status === 'fulfilled').length
          const failCount = items.length - successCount

          if (failCount === 0) {
            message.success(isBatch ? `已下线 ${successCount} 台设备，客户端将退出` : '已下线，客户端将退出')
          } else if (successCount === 0) {
            const firstRejected = results.find((result) => result.status === 'rejected')
            const errMsg = firstRejected && firstRejected.status === 'rejected'
              ? getErrorMessage(firstRejected.reason, '下线失败')
              : '下线失败'
            message.error(errMsg)
          } else {
            const firstRejected = results.find((result) => result.status === 'rejected')
            const errMsg = firstRejected && firstRejected.status === 'rejected'
              ? getErrorMessage(firstRejected.reason, '部分设备下线失败')
              : '部分设备下线失败'
            message.warning(`已下线 ${successCount} 台，失败 ${failCount} 台：${errMsg}`)
          }

          await Promise.all([
            loadOnlineDevices(devicePage, devicePageSize),
            loadBlockedDevices(blockedPage, blockedPageSize, blockedKeyword)
          ])
        } catch (err: any) {
          message.error(getErrorMessage(err, '下线失败'))
        } finally {
          setBlockingId('')
          setBatchBlocking(false)
        }
      }
    })
  }

  const blockOnlineDevice = (item: OnlineDeviceItem) => {
    confirmBlockOnlineDevices([item])
  }

  const blockSelectedOnlineDevices = () => {
    if (selectedOnlineIds.length === 0) return
    const selectedIdSet = new Set(selectedOnlineIds)
    const selectedItems = deviceItems.filter((item) => selectedIdSet.has(item.id))
    if (selectedItems.length === 0) {
      setSelectedOnlineIds([])
      message.warning('请选择要下线的在线设备')
      return
    }
    confirmBlockOnlineDevices(selectedItems)
  }

  const unblockDevice = async (item: BlockedDeviceItem) => {
    if (!appId) return
    setUnblockingId(item.id)
    try {
      await api.post(`/api/devices/${item.id}/unblock`, { app_id: appId })
      message.success('已恢复设备')
      await Promise.all([
        loadBlockedDevices(blockedPage, blockedPageSize, blockedKeyword),
        loadOnlineDevices(devicePage, devicePageSize)
      ])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '恢复失败')
    } finally {
      setUnblockingId('')
    }
  }

  const confirmBatchUnblock = () => {
    if (!appId || selectedBlockedIds.length === 0) return
    const selectedIdSet = new Set(selectedBlockedIds)
    const items = blockedItems.filter((item) => selectedIdSet.has(item.id))
    if (items.length === 0) {
      setSelectedBlockedIds([])
      message.warning('请选择要解禁的设备')
      return
    }
    Modal.confirm({
      title: `确认解禁这 ${items.length} 台设备？`,
      content: `选中的 ${items.length} 台设备将移出禁用列表，恢复后客户端可正常使用。`,
      okText: '确认批量解禁',
      cancelText: '取消',
      onOk: async () => {
        setBatchUnblocking(true)
        try {
          const results = await Promise.allSettled(
            items.map((item) => api.post(`/api/devices/${item.id}/unblock`, { app_id: appId }))
          )
          const successCount = results.filter((result) => result.status === 'fulfilled').length
          const failCount = items.length - successCount

          if (failCount === 0) {
            message.success(`已解禁 ${successCount} 台设备`)
          } else if (successCount === 0) {
            const firstRejected = results.find((result) => result.status === 'rejected')
            const errMsg = firstRejected && firstRejected.status === 'rejected'
              ? getErrorMessage(firstRejected.reason, '解禁失败')
              : '解禁失败'
            message.error(errMsg)
          } else {
            const firstRejected = results.find((result) => result.status === 'rejected')
            const errMsg = firstRejected && firstRejected.status === 'rejected'
              ? getErrorMessage(firstRejected.reason, '部分设备解禁失败')
              : '部分设备解禁失败'
            message.warning(`已解禁 ${successCount} 台，失败 ${failCount} 台：${errMsg}`)
          }

          setSelectedBlockedIds([])
          await Promise.all([
            loadBlockedDevices(blockedPage, blockedPageSize, blockedKeyword),
            loadOnlineDevices(devicePage, devicePageSize)
          ])
        } catch (err: any) {
          message.error(getErrorMessage(err, '解禁失败'))
        } finally {
          setBatchUnblocking(false)
        }
      }
    })
  }

  const addBlockedDevice = async () => {
    if (!appId) return
    const deviceID = manualBlockDeviceID.trim()
    if (!deviceID) {
      message.warning('请输入设备ID')
      return
    }
    setAddingBlocked(true)
    try {
      const reason = manualBlockReason.trim()
      await api.post(`/api/apps/${appId}/blocked-devices`, {
        device_id: deviceID,
        reason: reason || undefined
      })
      message.success('设备已加入禁用列表')
      setManualBlockDeviceID('')
      setManualBlockReason('')
      setBlockedPage(1)
      await Promise.all([
        loadBlockedDevices(1, blockedPageSize, blockedKeyword),
        loadOnlineDevices(devicePage, devicePageSize)
      ])
    } catch (err: any) {
      message.error(getErrorMessage(err, '添加封禁失败'))
    } finally {
      setAddingBlocked(false)
    }
  }

  useEffect(() => {
    if (!appId) return
    setSelectedOnlineIds([])
    setSelectedBlockedIds([])
    setManualBlockDeviceID('')
    setManualBlockReason('')
    setDevicePage(1)
    loadOnlineDevices(1, devicePageSize)
    setBlockedPage(1)
    loadBlockedDevices(1, blockedPageSize, '')
    setBlockedKeyword('')
  }, [appId, onlineEnabled])

  const onlineListActions = (
    <Space
      direction={isMobile ? 'vertical' : 'horizontal'}
      size={8}
      style={isMobile ? { width: '100%' } : undefined}
    >
      <Button
        danger
        onClick={blockSelectedOnlineDevices}
        disabled={!onlineEnabled || selectedOnlineIds.length === 0 || !!blockingId}
        loading={batchBlocking}
        style={isMobile ? { width: '100%' } : undefined}
      >
        批量移除并下线{selectedOnlineIds.length > 0 ? ` (${selectedOnlineIds.length})` : ''}
      </Button>
      <Button
        onClick={() => {
          loadOnlineDevices(1, devicePageSize)
          loadBlockedDevices(1, blockedPageSize, blockedKeyword)
        }}
        disabled={!onlineEnabled || batchBlocking}
        style={isMobile ? { width: '100%' } : undefined}
      >
        刷新
      </Button>
    </Space>
  )

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>在线设备</Title>
        <Text type="secondary">集中查看应用的实时在线设备</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <Space
          direction={isMobile ? 'vertical' : 'horizontal'}
          size={isMobile ? 8 : 12}
          style={{ width: '100%' }}
        >
          <Space direction={isMobile ? 'vertical' : 'horizontal'} size={8} style={isMobile ? { width: '100%' } : undefined}>
            <Text type="secondary">选择应用：</Text>
            <Select
              value={appId || undefined}
              style={{ width: isMobile ? '100%' : 260 }}
              loading={loadingApps}
              onChange={(value) => setAppId(value)}
              placeholder="请选择应用"
            >
              {apps.map((app) => (
                <Option key={app.id} value={app.id}>{app.name}</Option>
              ))}
            </Select>
          </Space>
        </Space>
      </Card>

      {!appId ? (
        <Card>
          <Text type="secondary">暂无应用，请先创建应用。</Text>
        </Card>
      ) : (
        <Row gutter={isMobile ? [12, 12] : [16, 16]}>
          <Col xs={24}>
            <Card title="实时在线设备" style={{ borderRadius: isMobile ? 10 : 12 }}>
              <Space direction="vertical" size={8}>
                <Space align="baseline">
                  <Text style={{ fontSize: isMobile ? 24 : 28, fontWeight: 600 }}>
                    {onlineEnabled ? (onlineCount === null ? '-' : onlineCount) : 0}
                  </Text>
                  <Text type="secondary">台</Text>
                </Space>
                <Text type="secondary">在线判定：最近 {onlineWindow} 秒内有心跳上报</Text>
                {onlineUpdatedAt && (
                  <Text type="secondary">更新时间：{new Date(onlineUpdatedAt).toLocaleTimeString()}</Text>
                )}
                <Tag color={!onlineEnabled ? 'default' : onlineStatus === 'connected' ? 'green' : onlineStatus === 'connecting' ? 'processing' : 'default'}>
                  {!onlineEnabled ? '已关闭' : onlineStatus === 'connected' ? '在线' : onlineStatus === 'connecting' ? '连接中' : '已断开'}
                </Tag>
              </Space>
            </Card>
          </Col>
          <Col xs={24}>
            <Card
              title="在线设备列表"
              style={{ borderRadius: isMobile ? 10 : 12 }}
              extra={isMobile ? undefined : onlineListActions}
            >
              {isMobile && <div style={{ marginBottom: 12 }}>{onlineListActions}</div>}
              <Tabs
                size={isMobile ? 'small' : 'middle'}
                tabBarGutter={isMobile ? 12 : 24}
                tabBarStyle={{ overflowX: 'auto', overflowY: 'hidden' }}
                items={[
                  {
                    key: 'online',
                    label: `在线设备 (${deviceTotal})`,
                    children: (
                      <Table
                        rowKey="id"
                        size={isMobile ? 'small' : 'middle'}
                        loading={deviceLoading}
                        scroll={isMobile ? { x: 980 } : { x: 1120 }}
                        dataSource={onlineEnabled ? deviceItems : []}
                        rowSelection={onlineEnabled
                          ? {
                              selectedRowKeys: selectedOnlineIds,
                              onChange: (keys) => setSelectedOnlineIds(keys.map((key) => String(key))),
                              getCheckboxProps: () => ({ disabled: batchBlocking || !!blockingId })
                            }
                          : undefined}
                        pagination={{
                          current: devicePage,
                          pageSize: devicePageSize,
                          total: deviceTotal,
                          size: isMobile ? 'small' : 'default',
                          responsive: true,
                          showSizeChanger: !isMobile,
                          onChange: (page, pageSize) => {
                            setSelectedOnlineIds([])
                            setDevicePage(page)
                            setDevicePageSize(pageSize)
                            loadOnlineDevices(page, pageSize)
                          }
                        }}
                        columns={[
                          { title: '设备ID', dataIndex: 'device_id', width: 220 },
                          { title: '平台', dataIndex: 'platform', width: 120 },
                          { title: '架构', dataIndex: 'arch', width: 100 },
                          { title: '版本', dataIndex: 'app_version', width: 120 },
                          { title: '用户ID', dataIndex: 'user_id', width: 160, render: (v: string) => v || '-' },
                          { title: 'IP', dataIndex: 'last_ip', width: 140, render: (v: string) => v || '-' },
                          {
                            title: '最后在线',
                            dataIndex: 'last_seen_at',
                            width: 180,
                            render: (v: string) => v ? new Date(v).toLocaleString() : '-'
                          },
                          {
                            title: '操作',
                            key: 'actions',
                            width: 140,
                            fixed: isMobile ? undefined : 'right',
                            render: (_: any, record: OnlineDeviceItem) => (
                              <Button
                                danger
                                size="small"
                                loading={blockingId === record.id}
                                disabled={batchBlocking}
                                onClick={() => blockOnlineDevice(record)}
                              >
                                {isMobile ? '下线' : '移除并下线'}
                              </Button>
                            )
                          }
                        ]}
                      />
                    )
                  },
                  {
                    key: 'blocked',
                    label: `已禁用设备 (${blockedTotal})`,
                    children: (
                      <>
                        <div
                          style={{
                            display: 'flex',
                            flexDirection: isMobile ? 'column' : 'row',
                            alignItems: isMobile ? 'stretch' : 'center',
                            justifyContent: 'space-between',
                            gap: 12,
                            marginBottom: 12
                          }}
                        >
                          <Space wrap style={isMobile ? { width: '100%' } : undefined}>
                            <Input
                              placeholder="按设备ID搜索"
                              value={blockedKeyword}
                              onChange={(e) => setBlockedKeyword(e.target.value)}
                              onPressEnter={() => {
                                setBlockedPage(1)
                                loadBlockedDevices(1, blockedPageSize, blockedKeyword)
                              }}
                              style={{ width: isMobile ? '100%' : 240 }}
                            />
                            <Button
                              onClick={() => {
                                setBlockedPage(1)
                                loadBlockedDevices(1, blockedPageSize, blockedKeyword)
                              }}
                              style={isMobile ? { width: '100%' } : undefined}
                            >
                              查询
                            </Button>
                            <Button
                              type="primary"
                              ghost
                              disabled={selectedBlockedIds.length === 0 || !!unblockingId}
                              loading={batchUnblocking}
                              onClick={confirmBatchUnblock}
                              style={isMobile ? { width: '100%' } : undefined}
                            >
                              批量解禁{selectedBlockedIds.length > 0 ? ` (${selectedBlockedIds.length})` : ''}
                            </Button>
                          </Space>
                          <Space
                            wrap
                            style={{
                              width: isMobile ? '100%' : undefined,
                              marginLeft: isMobile ? 0 : 'auto',
                              justifyContent: isMobile ? 'stretch' : 'flex-end'
                            }}
                          >
                            <Input
                              placeholder="输入设备ID并添加封禁"
                              value={manualBlockDeviceID}
                              onChange={(e) => setManualBlockDeviceID(e.target.value)}
                              onPressEnter={addBlockedDevice}
                              style={{ width: isMobile ? '100%' : 260 }}
                            />
                            <Input
                              placeholder="封禁原因（可选）"
                              value={manualBlockReason}
                              onChange={(e) => setManualBlockReason(e.target.value)}
                              onPressEnter={addBlockedDevice}
                              style={{ width: isMobile ? '100%' : 220 }}
                            />
                            <Button
                              danger
                              type="primary"
                              loading={addingBlocked}
                              onClick={addBlockedDevice}
                              style={isMobile ? { width: '100%' } : undefined}
                            >
                              添加封禁
                            </Button>
                          </Space>
                        </div>
                        <Table
                          rowKey="id"
                          size={isMobile ? 'small' : 'middle'}
                          loading={blockedLoading}
                          scroll={isMobile ? { x: 900 } : { x: 980 }}
                          dataSource={blockedItems}
                          rowSelection={{
                            selectedRowKeys: selectedBlockedIds,
                            onChange: (keys) => setSelectedBlockedIds(keys.map((key) => String(key))),
                            getCheckboxProps: () => ({ disabled: batchUnblocking || !!unblockingId })
                          }}
                          pagination={{
                            current: blockedPage,
                            pageSize: blockedPageSize,
                            total: blockedTotal,
                            size: isMobile ? 'small' : 'default',
                            responsive: true,
                            showSizeChanger: !isMobile,
                            onChange: (page, pageSize) => {
                              setSelectedBlockedIds([])
                              setBlockedPage(page)
                              setBlockedPageSize(pageSize)
                              loadBlockedDevices(page, pageSize, blockedKeyword)
                            }
                          }}
                          columns={[
                            { title: '设备ID', dataIndex: 'device_id', width: 240 },
                            { title: '禁用原因', dataIndex: 'reason', width: 220, render: (v: string) => v || '-' },
                            {
                              title: '禁用时间',
                              dataIndex: 'blocked_at',
                              width: 190,
                              render: (v: string) => v ? new Date(v).toLocaleString() : '-'
                            },
                            { title: '操作人', dataIndex: 'blocked_by', width: 160, render: (v: string) => v || '-' },
                            {
                              title: '操作',
                              key: 'actions',
                              width: 100,
                              fixed: isMobile ? undefined : 'right',
                              render: (_: any, record: BlockedDeviceItem) => (
                                <Button
                                  size="small"
                                  type="primary"
                                  ghost
                                  loading={unblockingId === record.id}
                                  disabled={batchUnblocking}
                                  onClick={() => unblockDevice(record)}
                                >
                                  恢复
                                </Button>
                              )
                            }
                          ]}
                        />
                      </>
                    )
                  }
                ]}
              />
            </Card>
          </Col>
        </Row>
      )}
    </div>
  )
}

import { Button, Card, Form, Grid, Input, Modal, Select, Space, Table, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select

export default function Devices() {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const [apps, setApps] = useState<any[]>([])
  const [devices, setDevices] = useState<any[]>([])
  const [countries, setCountries] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [tablePage, setTablePage] = useState(1)
  const [tablePageSize, setTablePageSize] = useState(10)
  const [form] = Form.useForm()

  const loadApps = async () => {
    try {
      const res = await api.get('/api/apps')
      setApps(res.data.items || [])
    } catch {
      message.error('加载应用失败')
    }
  }

  const loadDevices = async (params?: any, resetPagination = false) => {
    if (resetPagination) {
      setTablePage(1)
    }
    setLoading(true)
    try {
      const res = await api.get('/api/devices', { params })
      setDevices(res.data.items || [])
      setSelectedRowKeys([])
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载设备失败')
    } finally {
      setLoading(false)
    }
  }

  const loadCountries = async () => {
    try {
      const [regionsRes, devicesRes] = await Promise.all([
        api.get('/api/geo/regions'),
        api.get('/api/devices', { params: { limit: 500 } })
      ])
      const fromRegions = Array.isArray(regionsRes.data?.countries) ? regionsRes.data.countries : []
      const fromDevices = Array.isArray(devicesRes.data?.items)
        ? devicesRes.data.items.map((item: any) => item?.Country).filter((v: any) => typeof v === 'string' && v.trim() !== '')
        : []
      setCountries(Array.from(new Set([...fromRegions, ...fromDevices])))
    } catch {
      setCountries([])
    }
  }

  useEffect(() => {
    loadApps()
    loadCountries()
    loadDevices(undefined, true)
  }, [])

  const onSearch = async () => {
    const values = await form.validateFields()
    const query = {
      ...values,
      device_id: String(values.device_id || '').trim() || undefined,
      last_ip: String(values.last_ip || '').trim() || undefined
    }
    setSelectedRowKeys([])
    loadDevices(query, true)
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '确认删除选中设备历史记录？',
      content: '删除后不可恢复，仅删除设备历史记录，不影响应用和版本数据。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setDeleting(true)
        try {
          await api.post('/api/devices/batch-delete', { ids: selectedRowKeys })
          message.success('删除成功')
          setSelectedRowKeys([])
          const values = form.getFieldsValue()
          loadDevices(values, true)
          loadCountries()
        } catch (err: any) {
          message.error(err?.response?.data?.error || '删除失败')
        } finally {
          setDeleting(false)
        }
      }
    })
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>设备列表</Title>
        <Text type="secondary">查看客户端设备与版本分布</Text>
      </Space>

      <Card style={{ marginBottom: 16, borderRadius: isMobile ? 10 : 12 }}>
        <div
          style={{
            display: 'flex',
            flexDirection: isMobile ? 'column' : 'row',
            alignItems: isMobile ? 'stretch' : 'center',
            justifyContent: 'space-between',
            gap: 12
          }}
        >
          <Form layout={isMobile ? 'vertical' : 'inline'} form={form} style={isMobile ? { width: '100%' } : undefined}>
            <Form.Item name="app_id">
              <Select placeholder="选择应用" style={{ width: isMobile ? '100%' : 220 }} allowClear>
                {apps.map((app) => (
                  <Option key={app.ID} value={app.ID}>{app.Name}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="platform">
              <Select placeholder="平台" style={{ width: isMobile ? '100%' : 160 }} allowClear>
                <Option value="windows">Windows</Option>
                <Option value="mac">macOS</Option>
                <Option value="linux">Linux</Option>
                <Option value="android">Android</Option>
                <Option value="ios">iOS</Option>
              </Select>
            </Form.Item>
            <Form.Item name="country">
              <Select placeholder="国家" style={{ width: isMobile ? '100%' : 180 }} allowClear showSearch optionFilterProp="children">
                {countries.map((country) => (
                  <Option key={country} value={country}>{country}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="device_id">
              <Input
                placeholder="设备ID"
                allowClear
                style={{ width: isMobile ? '100%' : 220 }}
                onPressEnter={onSearch}
              />
            </Form.Item>
            <Form.Item name="last_ip">
              <Input
                placeholder="最后IP"
                allowClear
                style={{ width: isMobile ? '100%' : 180 }}
                onPressEnter={onSearch}
              />
            </Form.Item>
            <Form.Item>
              <Button type="primary" onClick={onSearch} style={isMobile ? { width: '100%' } : undefined}>查询</Button>
            </Form.Item>
          </Form>
          <Button
            danger
            disabled={selectedRowKeys.length === 0}
            loading={deleting}
            onClick={handleBatchDelete}
            style={isMobile ? { width: '100%' } : undefined}
          >
            批量删除
          </Button>
        </div>
      </Card>

      <Card style={{ borderRadius: isMobile ? 10 : 12 }}>
        <Table
          rowKey="ID"
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as string[])
          }}
          dataSource={devices}
          loading={loading}
          size={isMobile ? 'small' : 'middle'}
          pagination={{
            current: tablePage,
            pageSize: tablePageSize,
            size: isMobile ? 'small' : 'default',
            responsive: true,
            showSizeChanger: !isMobile,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (page, size) => {
              if (size && size !== tablePageSize) {
                setTablePageSize(size)
                setTablePage(1)
                return
              }
              setTablePage(page)
            },
            onShowSizeChange: (_, size) => {
              setTablePageSize(size)
              setTablePage(1)
            }
          }}
          scroll={isMobile ? { x: 1320 } : { x: 'max-content' }}
          columns={[
            { title: '设备ID', dataIndex: 'DeviceID', width: 360, ellipsis: true },
            { title: '平台', dataIndex: 'Platform', width: 120 },
            { title: '架构', dataIndex: 'Arch', width: 120 },
            { title: '版本', dataIndex: 'AppVersion', width: 120 },
            { title: '用户', dataIndex: 'UserID', width: 180, ellipsis: true },
            { title: '国家', dataIndex: 'Country', width: 120 },
            { title: '最后IP', dataIndex: 'LastIP', width: 180, ellipsis: true },
            { title: '最后活跃', dataIndex: 'LastSeenAt', width: 220, render: (d: string) => d ? new Date(d).toLocaleString() : '-' }
          ]}
        />
      </Card>
    </div>
  )
}

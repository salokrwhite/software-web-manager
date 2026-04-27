import {
  Alert,
  Avatar,
  Button,
  Checkbox,
  Col,
  DatePicker,
  Divider,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme
} from 'antd'
import {
  CopyOutlined,
  DownloadOutlined,
  DeleteOutlined,
  KeyOutlined,
  PlusOutlined
} from '@ant-design/icons'
import { useState } from 'react'
import api from '../../../api/client'

const { Text } = Typography
const { Option } = Select

type AppSecretsTabProps = {
  appId: string
  appSecrets: any[]
  isLocked: boolean
  reload: () => void
}

type CreatedSecret = {
  keyId: string
  appId: string
  appSecret: string
  scopes: string[]
  expiresAt?: string
  generatedAt?: string
}

function formatTime(value: string | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function formatExpireTime(value: string | undefined) {
  if (!value) return '永不过期'
  return formatTime(value)
}

function addDays(base: Date, days: number) {
  const date = new Date(base.getTime())
  date.setDate(date.getDate() + days)
  return date
}

function toCSVCell(value: string) {
  return `"${value.replace(/"/g, '""')}"`
}

export default function AppSecretsTab({
  appId,
  appSecrets,
  isLocked,
  reload
}: AppSecretsTabProps) {
  const { token } = theme.useToken()
  const [secretOpen, setSecretOpen] = useState(false)
  const [createdSecret, setCreatedSecret] = useState<CreatedSecret | null>(null)
  const [secretConfirmed, setSecretConfirmed] = useState(false)
  const [suggestExpire90d, setSuggestExpire90d] = useState(true)
  const [confirmSaving, setConfirmSaving] = useState(false)
  const [secretForm] = Form.useForm()

  const displayExpireTime = (() => {
    if (!createdSecret) return '永不过期'
    if (createdSecret.expiresAt) return formatExpireTime(createdSecret.expiresAt)
    if (!suggestExpire90d) return '永不过期'
    const base = createdSecret.generatedAt ? new Date(createdSecret.generatedAt) : new Date()
    if (Number.isNaN(base.getTime())) return '永不过期'
    return formatTime(addDays(base, 90).toISOString())
  })()

  const downloadSecretCSV = () => {
    if (!createdSecret) return
    const rows = [
      ['app_id', 'app_secret', 'scopes', 'expires_at'],
      [
        createdSecret.appId || '',
        createdSecret.appSecret || '',
        (createdSecret.scopes || []).join('|'),
        createdSecret.expiresAt || ''
      ]
    ]
    const csv = rows.map((row) => row.map((cell) => toCSVCell(String(cell))).join(',')).join('\n')
    const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `app_secret_${createdSecret.appId || 'credential'}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const createAppSecret = async () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await secretForm.validateFields()
      const secretName = String(values.name || '').trim() || 'app_secret'
      const payload = {
        name: secretName,
        scopes: values.scopes || [],
        expires_at: values.expires_at ? values.expires_at.format('YYYY-MM-DDTHH:mm:ssZ') : undefined
      }
      const res = await api.post(`/api/apps/${appId}/app-secrets`, payload)
      const itemExpiresAt = typeof res?.data?.item?.expires_at === 'string' ? res.data.item.expires_at : undefined
      setCreatedSecret({
        keyId: String(res?.data?.item?.id || '').trim(),
        appId: String(res?.data?.app_id || appId || '').trim(),
        appSecret: String(res?.data?.app_secret || '').trim(),
        scopes: Array.isArray(res?.data?.scopes) ? res.data.scopes : [],
        expiresAt: itemExpiresAt,
        generatedAt: new Date().toISOString()
      })
      setSecretConfirmed(false)
      setSuggestExpire90d(!itemExpiresAt)
      setSecretOpen(false)
      secretForm.resetFields()
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '生成失败')
    }
  }

  const revokeAppSecret = async (targetKeyId: string) => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    if (!targetKeyId) {
      message.error('缺少密钥ID')
      return
    }
    try {
      await api.delete(`/api/app-secrets/${targetKeyId}`)
      message.success('已撤销')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '撤销失败')
    }
  }

  const closeCreatedSecretModal = () => {
    setCreatedSecret(null)
    setSecretConfirmed(false)
    setSuggestExpire90d(true)
  }

  const copySecretOnly = () => {
    if (!createdSecret) return
    navigator.clipboard.writeText(createdSecret.appSecret || '')
    message.success('AccessKey Secret 已复制')
  }

  const handleConfirmCreatedSecret = async () => {
    if (!createdSecret) {
      closeCreatedSecretModal()
      return
    }
    if (!createdSecret.keyId) {
      message.error('缺少密钥ID')
      return
    }
    const targetExpiresAt = (() => {
      if (createdSecret.expiresAt) return createdSecret.expiresAt
      if (!suggestExpire90d) return null
      const base = createdSecret.generatedAt ? new Date(createdSecret.generatedAt) : new Date()
      if (Number.isNaN(base.getTime())) return null
      return addDays(base, 90).toISOString()
    })()

    try {
      setConfirmSaving(true)
      await api.patch(`/api/app-secrets/${createdSecret.keyId}/policy`, {
        expires_at: targetExpiresAt
      })
      await reload()
      closeCreatedSecretModal()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '保存密钥策略失败')
    } finally {
      setConfirmSaving(false)
    }
  }

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <Text type="secondary">管理客户端签名鉴权密钥（app_id + app_secret）</Text>
        </Col>
        <Col>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setSecretOpen(true)} disabled={isLocked}>
              生成密钥
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <Table
        rowKey="id"
        dataSource={appSecrets}
        pagination={false}
        columns={[
          {
            title: '名称',
            dataIndex: 'name',
            render: (name: string) => (
              <Space>
                <Avatar size="small" icon={<KeyOutlined />} style={{ background: token.colorWarning }} />
                <Text strong>{name || 'app_secret'}</Text>
              </Space>
            )
          },
          {
            title: 'Scopes',
            dataIndex: 'scopes',
            render: (scopes: any) => (
              <Space size={[4, 4]} wrap>
                {(Array.isArray(scopes) ? scopes : []).map((item: string, idx: number) => (
                  <Tag
                    key={`${item}-${idx}`}
                    color={item === 'update:check' ? 'processing' : item === 'event:write' ? 'success' : 'geekblue'}
                  >
                    {item}
                  </Tag>
                ))}
              </Space>
            )
          },
          {
            title: '过期时间',
            dataIndex: 'expires_at',
            render: (value: string) => formatExpireTime(value)
          },
          {
            title: '更新时间',
            dataIndex: 'updated_at',
            render: (value: string) => formatTime(value)
          },
          {
            title: '操作',
            render: (_: any, record: any) => (
              <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => revokeAppSecret(String(record?.id || ''))}
                  disabled={isLocked}
                >
                  撤销
                </Button>
              </Tooltip>
            )
          }
        ]}
      />

      <Modal
        title="生成应用密钥"
        open={secretOpen}
        onOk={createAppSecret}
        onCancel={() => { setSecretOpen(false); secretForm.resetFields() }}
        width={480}
      >
        <Form
          layout="vertical"
          form={secretForm}
          style={{ marginTop: 16 }}
          initialValues={{ name: 'app_secret', scopes: ['update:check', 'event:write'] }}
        >
          <Form.Item name="name" label="名称">
            <Input maxLength={128} placeholder="不填写则默认 app_secret" />
          </Form.Item>
          <Form.Item name="scopes" label="权限范围">
            <Select mode="multiple" placeholder="选择权限范围">
              <Option value="update:check">update:check</Option>
              <Option value="event:write">event:write</Option>
            </Select>
          </Form.Item>
          <Form.Item name="expires_at" label="过期时间（可选）">
            <DatePicker showTime style={{ width: '100%' }} placeholder="不设置则永不过期" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="创建 AccessKey"
        open={!!createdSecret}
        onCancel={closeCreatedSecretModal}
        closable={false}
        maskClosable={false}
        width={920}
        footer={
          <Row justify="space-between" align="middle" style={{ width: '100%', marginTop: 8 }}>
            <Col>
              <Checkbox checked={secretConfirmed} onChange={(e) => setSecretConfirmed(e.target.checked)}>
                我已保存好 AccessKey Secret
              </Checkbox>
            </Col>
            <Col>
              <Button type="primary" disabled={!secretConfirmed} loading={confirmSaving} onClick={handleConfirmCreatedSecret}>
                确定
              </Button>
            </Col>
          </Row>
        }
      >
        <Space direction="vertical" size={14} style={{ width: '100%' }}>
          <div style={{ border: '1px solid #e5e6eb', borderRadius: 8, padding: 16 }}>
            <Text strong style={{ fontSize: 18, display: 'block', marginBottom: 12 }}>保存 AccessKey</Text>
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 12 }}
              message="当前窗口关闭后，无法再次查询 Secret，请妥善保管。若丢失 AccessKey Secret，需创建新的 AccessKey。"
            />

            <div style={{ background: '#f0f2f5', borderRadius: 6, padding: '10px 14px', marginBottom: 10 }}>
              <Row style={{ marginBottom: 12 }}>
                <Col span={6}>
                  <Text>App ID</Text>
                </Col>
                <Col span={18}>
                  <Text style={{ wordBreak: 'break-all' }}>
                    {createdSecret?.appId || '-'}
                  </Text>
                </Col>
              </Row>
              <Row>
                <Col span={6}>
                  <Text>AccessKey Secret</Text>
                </Col>
                <Col span={18}>
                  <Text style={{ wordBreak: 'break-all' }}>
                    {createdSecret?.appSecret || '-'}
                  </Text>
                </Col>
              </Row>
            </div>

            <Space>
              <Button type="link" style={{ paddingLeft: 0 }} icon={<DownloadOutlined />} onClick={downloadSecretCSV}>
                下载 CSV 文件
              </Button>
              <Button type="link" icon={<CopyOutlined />} onClick={copySecretOnly}>
                复制
              </Button>
            </Space>
          </div>

          <div style={{ border: '1px solid #e5e6eb', borderRadius: 8, padding: 16 }}>
            <Text strong style={{ fontSize: 18, display: 'block', marginBottom: 12 }}>安全建议</Text>

            <Checkbox
              checked={suggestExpire90d}
              disabled={!!createdSecret?.expiresAt}
              onChange={(e) => setSuggestExpire90d(e.target.checked)}
            >
              <Text strong>设置 AccessKey 最大闲置时间为 90 天</Text>
            </Checkbox>
            <div>
              <Text type="secondary">
                为降低长期闲置 AccessKey 的泄露风险，建议设置 90 天未使用自动失效（可选）。
                {createdSecret?.expiresAt ? ' 已在创建时设置过期时间，此项不可选。' : ''}
              </Text>
            </div>

            <Divider style={{ margin: '12px 0' }} />

            <Row>
              <Col span={6}>
                <Text type="secondary">Scopes</Text>
              </Col>
              <Col span={18}>
                <Space size={[4, 4]} wrap>
                  {(createdSecret?.scopes || []).map((item) => (
                    <Tag key={item} color={item === 'update:check' ? 'processing' : 'success'}>
                      {item}
                    </Tag>
                  ))}
                </Space>
              </Col>
            </Row>
            <Row style={{ marginTop: 8 }}>
              <Col span={6}>
                <Text type="secondary">过期时间</Text>
              </Col>
              <Col span={18}>
                <Text>{displayExpireTime}</Text>
              </Col>
            </Row>
          </div>
        </Space>
      </Modal>
    </>
  )
}

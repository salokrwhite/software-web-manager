import {
  Alert,
  Avatar,
  Button,
  Col,
  Modal,
  Popconfirm,
  Row,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme
} from 'antd'
import {
  CheckCircleOutlined,
  DeleteOutlined,
  KeyOutlined,
  PlusOutlined,
  StopOutlined,
  SyncOutlined
} from '@ant-design/icons'
import { useState } from 'react'
import api from '../../../api/client'
import FeatureGuide, { GuideTag } from '../components/FeatureGuide'

const { Text, Paragraph } = Typography

type AuthzKeysTabProps = {
  appId: string
  authzKeys: any[]
  isLocked: boolean
  reload: () => void
}

type CreatedKey = {
  keyId: string
  publicKey: string
}

function formatTime(value: string | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const STATUS_META: Record<string, { color: string; label: string }> = {
  active: { color: 'success', label: '生效中' },
  pending: { color: 'processing', label: '待激活' },
  retired: { color: 'default', label: '已停用' }
}

function statusMeta(record: any) {
  if (record?.revoked_at) return { color: 'error', label: '已吊销' }
  return STATUS_META[record?.status] || { color: 'default', label: record?.status || '-' }
}

export default function AuthzKeysTab({ appId, authzKeys, isLocked, reload }: AuthzKeysTabProps) {
  const { token } = theme.useToken()
  const [createdKey, setCreatedKey] = useState<CreatedKey | null>(null)
  const [busyId, setBusyId] = useState<string>('')
  const [creating, setCreating] = useState(false)

  const guardLocked = () => {
    if (isLocked) {
      message.warning('应用待审核，暂不可操作')
      return true
    }
    return false
  }

  const handleCreated = (data: any) => {
    setCreatedKey({
      keyId: String(data?.key_id || '').trim(),
      publicKey: String(data?.public_key || '').trim()
    })
    reload()
  }

  const createKey = async () => {
    if (guardLocked()) return
    try {
      setCreating(true)
      const res = await api.post(`/api/apps/${appId}/authz-keys`, {})
      handleCreated(res?.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '生成失败')
    } finally {
      setCreating(false)
    }
  }

  const rotateKey = async (record: any) => {
    if (guardLocked()) return
    try {
      setBusyId(record.id)
      const res = await api.post(`/api/authz-keys/${record.id}/rotate`, {})
      handleCreated(res?.data)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '轮换失败')
    } finally {
      setBusyId('')
    }
  }

  const activateKey = async (record: any) => {
    if (guardLocked()) return
    try {
      setBusyId(record.id)
      await api.post(`/api/authz-keys/${record.id}/activate`, {})
      message.success('已激活，服务端将使用该密钥签名')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '激活失败')
    } finally {
      setBusyId('')
    }
  }

  const revokeKey = async (record: any) => {
    if (guardLocked()) return
    try {
      setBusyId(record.id)
      await api.delete(`/api/authz-keys/${record.id}`)
      message.success('已吊销')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '吊销失败')
    } finally {
      setBusyId('')
    }
  }

  const deleteKey = async (record: any) => {
    if (guardLocked()) return
    try {
      setBusyId(record.id)
      await api.delete(`/api/authz-keys/${record.id}/purge`)
      message.success('已删除')
      reload()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '删除失败')
    } finally {
      setBusyId('')
    }
  }

  return (
    <>
      <FeatureGuide
        storageKey="authz-keys"
        title="授权密钥"
        summary={
          <>
            授权密钥用于验证<Text strong>「这台设备是否被允许运行」</Text>：服务端用本应用的私钥给授权裁决签名，
            客户端用内置的<Text strong>公钥</Text>验签。<Text strong>每个应用一对独立密钥</Text>，
            <Text strong>私钥只存在服务端、永不外发</Text>，你只需把公钥和 Key ID 内置到客户端。
          </>
        }
        steps={[
          {
            title: '新建待激活密钥',
            description: <>点右上角<GuideTag>新建密钥</GuideTag>，生成一对新密钥（状态为<Text strong>待激活</Text>，此时服务端还不会用它签名）。</>
          },
          {
            title: '把公钥内置到客户端并发版',
            description: <>复制这把新密钥的<Text strong>公钥</Text>和 <Text strong>Key ID</Text>，发布一个内置了新公钥的客户端版本。</>
          },
          {
            title: '激活新密钥',
            description: <>等足够多客户端升级后点<GuideTag>激活</GuideTag>，服务端改用新密钥签名；老客户端（已含新公钥）照常验签。</>
          },
          {
            title: '吊销并清理旧密钥',
            description: <>观察期过后点<GuideTag>吊销</GuideTag>停用旧密钥；确认无客户端再依赖后可<GuideTag>删除</GuideTag>清理。</>
          }
        ]}
        tips={[
          <>和<Text strong>「应用密钥」</Text>的区别：应用密钥是<Text strong>客户端 → 服务端</Text>的「认证调用方」（对称、随客户端分发的弱标识）；授权密钥是<Text strong>服务端 → 客户端</Text>的「授权裁决」（非对称、私钥只在服务端，能防伪造/离线服务器）。两者配合使用。</>,
          <><Text strong>私钥永不外发</Text>：界面只展示公钥和 Key ID，私钥在服务端加密存储，任何接口都不会返回。</>,
          <>新建应用会<Text strong>自动开通</Text>一把生效中的密钥，开箱即用。</>,
          <>务必<Text strong>先发布带新公钥的客户端、再激活</Text>；顺序反了会把还没升级的老客户端挡在门外。</>,
          <><GuideTag>删除</GuideTag>仅对<Text strong>已停用 / 已吊销</Text>的密钥开放；正在生效的密钥需先吊销或轮换。</>
        ]}
      />
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <Text type="secondary">管理本应用的授权签名密钥（key_id + 公钥）</Text>
        </Col>
        <Col>
          <Tooltip title={isLocked ? '待审核，无法操作' : ''}>
            <Button type="primary" icon={<PlusOutlined />} loading={creating} onClick={createKey} disabled={isLocked}>
              新建密钥
            </Button>
          </Tooltip>
        </Col>
      </Row>
      <Table
        rowKey="id"
        dataSource={authzKeys}
        pagination={false}
        columns={[
          {
            title: 'Key ID',
            dataIndex: 'key_id',
            render: (keyId: string) => (
              <Space>
                <Avatar size="small" icon={<KeyOutlined />} style={{ background: token.colorPrimary }} />
                <Text strong copyable={{ text: keyId }}>{keyId || '-'}</Text>
              </Space>
            )
          },
          {
            title: '公钥',
            dataIndex: 'public_key',
            render: (pub: string) =>
              pub ? (
                <Text copyable={{ text: pub }}>{`${pub.slice(0, 12)}…${pub.slice(-6)}`}</Text>
              ) : (
                '-'
              )
          },
          {
            title: '状态',
            dataIndex: 'status',
            render: (_: any, record: any) => {
              const meta = statusMeta(record)
              return <Tag color={meta.color}>{meta.label}</Tag>
            }
          },
          {
            title: '激活时间',
            dataIndex: 'activated_at',
            render: (value: string) => formatTime(value)
          },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            render: (value: string) => formatTime(value)
          },
          {
            title: '操作',
            render: (_: any, record: any) => {
              const revoked = !!record.revoked_at
              const isActive = record.status === 'active' && !revoked
              const isPending = record.status === 'pending' && !revoked
              const isTerminal = record.status === 'retired' // 已停用 或 已吊销
              return (
                <Space>
                  {isPending && (
                    <Popconfirm
                      title="激活该密钥？"
                      description="激活后服务端立即改用此密钥签名，请确认带新公钥的客户端版本已发布。"
                      okText="激活"
                      cancelText="取消"
                      disabled={isLocked}
                      onConfirm={() => activateKey(record)}
                    >
                      <Button size="small" type="primary" icon={<CheckCircleOutlined />} loading={busyId === record.id} disabled={isLocked}>
                        激活
                      </Button>
                    </Popconfirm>
                  )}
                  {isActive && (
                    <Button size="small" icon={<SyncOutlined />} loading={busyId === record.id} disabled={isLocked} onClick={() => rotateKey(record)}>
                      轮换
                    </Button>
                  )}
                  {(isActive || isPending) && (
                    <Popconfirm
                      title="吊销该密钥？"
                      description={isActive ? '吊销当前生效密钥后，需有其他生效密钥或平台兜底，否则新客户端将无法获取授权。' : '吊销后该密钥不可恢复。'}
                      okText="吊销"
                      okButtonProps={{ danger: true }}
                      cancelText="取消"
                      disabled={isLocked}
                      onConfirm={() => revokeKey(record)}
                    >
                      <Button size="small" danger icon={<StopOutlined />} loading={busyId === record.id} disabled={isLocked}>
                        吊销
                      </Button>
                    </Popconfirm>
                  )}
                  {isTerminal && (
                    <Popconfirm
                      title="删除该密钥？"
                      description="将从列表中永久移除该密钥（已停用/已吊销），不可恢复。"
                      okText="删除"
                      okButtonProps={{ danger: true }}
                      cancelText="取消"
                      disabled={isLocked}
                      onConfirm={() => deleteKey(record)}
                    >
                      <Button size="small" danger icon={<DeleteOutlined />} loading={busyId === record.id} disabled={isLocked}>
                        删除
                      </Button>
                    </Popconfirm>
                  )}
                </Space>
              )
            }
          }
        ]}
      />

      <Modal
        title="密钥已创建（待激活）"
        open={!!createdKey}
        onCancel={() => setCreatedKey(null)}
        footer={<Button type="primary" onClick={() => setCreatedKey(null)}>我已复制公钥</Button>}
        width={720}
      >
        <Space direction="vertical" size={14} style={{ width: '100%' }}>
          <Alert
            type="warning"
            showIcon
            message="把下面的公钥与 Key ID 内置到客户端的 AuthzPublicKeys，并发布该客户端版本后，再回到列表点击“激活”。"
          />
          <div>
            <Text type="secondary">Key ID</Text>
            <Paragraph copyable={{ text: createdKey?.keyId }} style={{ marginBottom: 8 }}>
              <Text>{createdKey?.keyId}</Text>
            </Paragraph>
          </div>
          <div>
            <Text type="secondary">公钥 (hex)</Text>
            <Paragraph copyable={{ text: createdKey?.publicKey }} style={{ marginBottom: 0 }}>
              <Text style={{ wordBreak: 'break-all' }}>{createdKey?.publicKey}</Text>
            </Paragraph>
          </div>
        </Space>
      </Modal>
    </>
  )
}

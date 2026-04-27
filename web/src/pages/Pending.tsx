import { Button, Card, Typography, Tag, Space, Spin, Modal, Upload, message, Steps } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'
import { UploadOutlined } from '@ant-design/icons'
import api from '../api/client'

const { Title, Text } = Typography

const maxMaterialSize = 20 * 1024 * 1024

type EnterpriseStatus = {
  org_id: string
  org_name: string
  status: string
  rejection_reason?: string | null
  allow_resubmit?: boolean
  resubmit_token?: string | null
  reviewed_at?: string | null
  created_at?: string
}

export default function Pending() {
  const navigate = useNavigate()
  const location = useLocation()
  const [status, setStatus] = useState<EnterpriseStatus | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [resubmitOpen, setResubmitOpen] = useState(false)
  const [resubmitLoading, setResubmitLoading] = useState(false)
  const [fileList, setFileList] = useState<any[]>([])

  const orgId = useMemo(() => {
    const params = new URLSearchParams(location.search)
    return (params.get('id') || '').trim()
  }, [location.search])

  const loadStatus = async () => {
    if (!orgId) {
      setError('缺少申请ID，请从注册完成页进入')
      setStatus(null)
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await api.get(`/api/auth/enterprise-status/${orgId}`)
      setStatus(res.data)
    } catch (err: any) {
      setStatus(null)
      setError(err?.response?.data?.error || '加载状态失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadStatus()
  }, [orgId])

  const statusTag = (value?: string) => {
    const s = (value || '').toLowerCase()
    if (s === 'active') return <Tag color="green">已通过</Tag>
    if (s === 'pending') return <Tag color="orange">待审核</Tag>
    if (s === 'rejected') return <Tag color="red">已驳回</Tag>
    if (s === 'disabled') return <Tag color="red">已禁用</Tag>
    return <Tag>未知</Tag>
  }

  const submitResubmit = async () => {
    if (!orgId) {
      message.error('缺少申请ID')
      return
    }
    try {
      if (fileList.length === 0) {
        message.error('请上传企业材料')
        return
      }
      const resubmitToken = (status?.resubmit_token || '').trim()
      if (!resubmitToken) {
        message.error('缺少重提令牌，请刷新页面后重试')
        return
      }
      const oversized = fileList.find((file) => (file.size || file.originFileObj?.size || 0) > maxMaterialSize)
      if (oversized) {
        message.error('材料文件大小不能超过 20MB')
        return
      }
      const formData = new FormData()
      formData.append('org_id', orgId)
      formData.append('resubmit_token', resubmitToken)
      fileList.forEach((file) => {
        const raw = file.originFileObj || file
        formData.append('materials', raw)
      })
      setResubmitLoading(true)
      await api.post('/api/auth/enterprise-resubmit', formData)
      message.success('已重新提交审核')
      setResubmitOpen(false)
      setFileList([])
      loadStatus()
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || '提交失败')
    } finally {
      setResubmitLoading(false)
    }
  }

  const reviewedAt = status?.reviewed_at ? new Date(status.reviewed_at).toLocaleString() : ''
  const createdAt = status?.created_at ? new Date(status.created_at).toLocaleString() : ''
  const normalizedStatus = (status?.status || '').toLowerCase()
  const currentStep = normalizedStatus === 'pending' ? 1 : normalizedStatus ? 2 : 0
  const stepsStatus: 'wait' | 'process' | 'finish' | 'error' = normalizedStatus === ''
    ? 'wait'
    : normalizedStatus === 'rejected'
      ? 'error'
      : normalizedStatus === 'active'
        ? 'finish'
        : 'process'
  const stepItems = [
    {
      title: '提交申请',
      description: createdAt ? `提交于 ${createdAt}` : '填写企业信息并提交'
    },
    {
      title: '审核中',
      description: normalizedStatus === 'pending' ? '管理员审核中' : '等待审核完成'
    },
    {
      title: normalizedStatus === 'active' ? '审核通过' : normalizedStatus === 'rejected' ? '审核未通过' : '审核结果',
      description: normalizedStatus === 'active'
        ? (reviewedAt ? `通过于 ${reviewedAt}` : '可以登录使用系统')
        : normalizedStatus === 'rejected'
          ? '请查看驳回原因'
          : '审核完成后将通知'
    }
  ]

  return (
      <div style={{ minHeight: '100vh', background: '#f5f7fa', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '40px 24px' }}>
        <Card style={{ width: 'min(860px, 96vw)', borderRadius: 12 }} styles={{ body: { padding: '36px 44px' } }}>
        <Space direction="vertical" size={20} style={{ width: '100%' }}>
          <div>
            <Title level={3} style={{ marginBottom: 12 }}>账号待审核</Title>
            <Text type="secondary" style={{ lineHeight: 1.7 }}>
              您的企业注册申请已提交，系统管理员审核通过后即可登录使用。
            </Text>
          </div>

          <Steps
            current={currentStep}
            status={stepsStatus}
            items={stepItems}
            style={{ padding: '12px 0 16px' }}
          />

          {!orgId && (
            <Text type="warning">未检测到申请ID，无法查询实时状态。</Text>
          )}

          {orgId && (
            <>
              <Space align="center">
                {statusTag(status?.status)}
                {status?.org_name && <Text type="secondary">{status.org_name}</Text>}
              </Space>
              {createdAt && <Text type="secondary">提交时间：{createdAt}</Text>}
              {reviewedAt && <Text type="secondary">审核时间：{reviewedAt}</Text>}
              {status?.status === 'pending' && (
                <Text type="secondary">当前申请正在审核中，请稍后刷新查看结果。</Text>
              )}
              {status?.status === 'active' && (
                <Text type="secondary">审核已通过，可以登录使用系统。</Text>
              )}
              {status?.status === 'rejected' && (
                <Text type="danger">驳回理由：{status.rejection_reason || '未提供驳回理由'}</Text>
              )}
            </>
          )}

          {error && <Text type="danger">{error}</Text>}

          {loading && (
            <div style={{ display: 'flex', justifyContent: 'center', padding: '8px 0' }}>
              <Spin />
            </div>
          )}

          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <Space>
              <Button onClick={loadStatus} disabled={!orgId} loading={loading}>刷新状态</Button>
              {status?.status === 'active' && (
                <Button type="primary" onClick={() => navigate('/login')}>去登录</Button>
              )}
              {status?.status === 'rejected' && status?.allow_resubmit && (
                <Button type="primary" onClick={() => setResubmitOpen(true)}>重新提交材料</Button>
              )}
              <Button onClick={() => navigate('/login')}>返回登录</Button>
            </Space>
          </div>
        </Space>
      </Card>

      <Modal
        title="重新提交企业材料"
        open={resubmitOpen}
        onOk={submitResubmit}
        confirmLoading={resubmitLoading}
        onCancel={() => { setResubmitOpen(false); setFileList([]) }}
        okText="提交"
        cancelText="取消"
      >
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
            仅需上传企业材料即可重新提交审核。
          </Text>
          <Upload
            multiple
            beforeUpload={() => false}
            fileList={fileList}
            onChange={({ fileList: nextList }) => setFileList(nextList)}
          >
            <Button icon={<UploadOutlined />}>选择材料</Button>
          </Upload>
          <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
            必传材料，单个文件不超过 20MB
          </Text>
        </div>
      </Modal>
    </div>
  )
}

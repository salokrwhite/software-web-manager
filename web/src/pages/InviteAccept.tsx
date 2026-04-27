import { useEffect, useState } from 'react'
import { Button, Card, Form, Input, Typography, message, Spin } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'
import api, { storeTokens } from '../api/client'

const { Title, Text } = Typography

type InviteInfo = {
  org_id?: string
  org_name?: string
  org_type?: string
  role?: string
  expires_at?: string
}

export default function InviteAccept() {
  const location = useLocation()
  const token = location.pathname.split('/invite/')[1] || ''
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [inviteLoading, setInviteLoading] = useState(true)
  const [inviteInfo, setInviteInfo] = useState<InviteInfo | null>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    const loadInvite = async () => {
      if (!token) {
        message.error('邀请链接无效')
        setInviteLoading(false)
        return
      }
      try {
        const res = await api.get(`/api/org-invites/${token}`)
        setInviteInfo(res.data || {})
      } catch (err: any) {
        message.error(err?.response?.data?.error || '邀请链接无效')
      } finally {
        setInviteLoading(false)
      }
    }
    loadInvite()
  }, [token])

  const onSubmit = async () => {
    if (!token) {
      message.error('邀请链接无效')
      return
    }
    try {
      const values = await form.validateFields()
      setLoading(true)
      const payload: Record<string, any> = {}
      if (values.password && values.password.trim()) {
        payload.password = values.password.trim()
      }
      const res = await api.post(`/api/org-invites/${token}/accept`, payload)
      storeTokens(res.data.tokens)
      if (res.data.org_id) {
        sessionStorage.setItem('org_id', res.data.org_id)
      }
      if (res.data.role) {
        sessionStorage.setItem('role', res.data.role)
      }
      if (res.data.user?.email) {
        sessionStorage.setItem('user_email', res.data.user.email)
      } else {
        sessionStorage.removeItem('user_email')
      }
      if (res.data.org_type) {
        sessionStorage.setItem('org_type', res.data.org_type)
      } else {
        sessionStorage.removeItem('org_type')
      }
      sessionStorage.removeItem('impersonating')
      sessionStorage.removeItem('impersonation_org_id')
      sessionStorage.removeItem('system_backup_access_token')
      sessionStorage.removeItem('system_backup_refresh_token')
      sessionStorage.removeItem('system_backup_org_id')
      sessionStorage.removeItem('system_backup_role')
      if (res.data.system_role) {
        sessionStorage.setItem('system_role', res.data.system_role)
      } else {
        sessionStorage.removeItem('system_role')
      }
      message.success('加入组织成功')
      navigate('/dashboard')
    } catch (err: any) {
      message.error(err?.response?.data?.error || '接受邀请失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', background: '#f5f7fa', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Card style={{ width: 420 }}>
        {inviteLoading ? (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <Spin />
          </div>
        ) : (
          <>
            <Title level={4} style={{ marginBottom: 8 }}>
              {inviteInfo?.org_name ? `接受 ${inviteInfo.org_name} 的邀请` : '接受组织邀请'}
            </Title>
            <Text type="secondary">
              欢迎加入我们的团队！如您已有账号，可直接加入，无需设置密码。
            </Text>
            <Form form={form} layout="vertical" style={{ marginTop: 24 }}>
              <Form.Item
                name="password"
                label="设置密码（可选）"
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value) return Promise.resolve()
                      if (String(value).length < 6) {
                        return Promise.reject(new Error('密码至少 6 位'))
                      }
                      return Promise.resolve()
                    }
                  })
                ]}
              >
                <Input.Password placeholder="设置密码（已注册可留空）" />
              </Form.Item>
              <Button type="primary" block loading={loading} onClick={onSubmit}>
                加入组织
              </Button>
            </Form>
          </>
        )}
      </Card>
    </div>
  )
}

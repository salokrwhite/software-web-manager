import { Alert, Button, Card, Col, Empty, Row, Space, Spin, Tag, Typography } from 'antd'
import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import api, { storeTokens } from '../api/client'
import { getSafeRedirectPath } from '../utils/redirect'

const { Title, Text } = Typography

export default function OrgSelect() {
  const location = useLocation()
  const params = new URLSearchParams(location.search)
  const redirectParam = params.get('redirect')
  const safeRedirect = getSafeRedirectPath(redirectParam, '/dashboard')

  const [orgs, setOrgs] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadOrgs = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await api.get('/api/orgs')
      setOrgs(res.data.items || [])
    } catch (err: any) {
      setError(err?.response?.data?.error || '加载组织失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
  }, [])

  const handleEnter = async (org: any) => {
    const orgId = org.id || org.ID
    if (!orgId) return
    try {
      const res = await api.post(`/api/orgs/${orgId}/switch`)
      const tokens = res.data.tokens
      if (tokens) {
        storeTokens(tokens)
      }
      if (res.data.org_id) {
        sessionStorage.setItem('org_id', res.data.org_id)
      } else {
        sessionStorage.removeItem('org_id')
      }
      if (res.data.role) {
        sessionStorage.setItem('role', res.data.role)
      } else {
        sessionStorage.removeItem('role')
      }
      if (res.data.org_type) {
        sessionStorage.setItem('org_type', res.data.org_type)
      } else {
        sessionStorage.removeItem('org_type')
      }
      window.location.href = safeRedirect
    } catch (err: any) {
      setError(err?.response?.data?.error || '切换组织失败')
    }
  }

  const renderOrgType = (value: string) => {
    if (value === 'personal') return <Tag color="blue">个人空间</Tag>
    if (value === 'enterprise') return <Tag color="green">企业</Tag>
    return <Tag>组织</Tag>
  }

  return (
    <div style={{ padding: 32, maxWidth: 1100, margin: '0 auto' }}>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>选择进入的组织</Title>
        <Text type="secondary">请选择进入个人空间或企业组织</Text>
      </Space>

      {error && (
        <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />
      )}

      {loading ? (
        <div style={{ padding: 80, textAlign: 'center' }}>
          <Spin size="large" />
        </div>
      ) : orgs.length === 0 ? (
        <Empty description="暂无组织，请创建或加入企业">
          <Button onClick={() => (window.location.href = '/dashboard')}>返回仪表盘</Button>
        </Empty>
      ) : (
        <Row gutter={[16, 16]}>
          {orgs.map((org) => (
            <Col xs={24} sm={12} lg={8} key={org.id || org.ID}>
              <Card
                title={(
                  <Space>
                    <Text strong>{org.name || org.Name}</Text>
                    {renderOrgType((org.org_type || org.OrgType || '').toLowerCase())}
                  </Space>
                )}
                bordered
                style={{ borderRadius: 10 }}
                actions={[
                  <Button type="primary" onClick={() => handleEnter(org)} key="enter">进入</Button>
                ]}
              >
                <Space direction="vertical" size={6}>
                  <Text type="secondary">角色：{org.role || org.Role || '-'}</Text>
                  <Text type="secondary">成员数：{org.member_count ?? org.MemberCount ?? 0}</Text>
                  <Text type="secondary">应用数：{org.app_count ?? org.AppCount ?? 0}</Text>
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  )
}

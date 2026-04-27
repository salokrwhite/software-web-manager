import { Card, Descriptions, Typography, message } from 'antd'
import { useEffect, useMemo, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography

const formatPlanLabel = (plan: string) => {
  const value = (plan || '').toLowerCase()
  if (value === 'team') return 'Team'
  if (value === 'enterprise') return 'Enterprise'
  return 'Free'
}

export default function OrgAttributes() {
  const [loading, setLoading] = useState(false)
  const [org, setOrg] = useState<any>(null)
  const currentOrgId = sessionStorage.getItem('org_id') || ''

  const loadCurrentOrg = async () => {
    if (!currentOrgId) {
      setOrg(null)
      return
    }
    setLoading(true)
    try {
      const res = await api.get('/api/orgs')
      const items = res?.data?.items || []
      const current = items.find((item: any) => String(item.id || item.ID) === currentOrgId)
      setOrg(current || null)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载企业属性失败')
      setOrg(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCurrentOrg()
  }, [currentOrgId])

  const planLabel = useMemo(() => formatPlanLabel(String(org?.plan || org?.Plan || 'free')), [org])
  const orgName = String(org?.name || org?.Name || '-')

  return (
    <Card style={{ borderRadius: 12 }} loading={loading}>
      <Title level={4} style={{ marginTop: 0, marginBottom: 8 }}>
        企业属性
      </Title>
      <Text type="secondary">当前企业基础属性信息</Text>
      <Descriptions
        bordered
        column={1}
        size="middle"
        style={{ marginTop: 16 }}
        items={[
          { key: 'org_id', label: '当前企业ID', children: currentOrgId || '-' },
          { key: 'org_name', label: '企业名称', children: orgName },
          { key: 'org_plan', label: '企业套餐组', children: planLabel }
        ]}
      />
    </Card>
  )
}

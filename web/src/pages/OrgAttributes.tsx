import { Card, Descriptions, Result, Typography, message } from 'antd'
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
  const [forbidden, setForbidden] = useState(false)
  const currentOrgId = sessionStorage.getItem('org_id') || ''

  const loadCurrentOrg = async () => {
    if (!currentOrgId) {
      setOrg(null)
      return
    }
    setLoading(true)
    setForbidden(false)
    try {
      // Viewing the current org's attributes requires org_management.view —
      // enforced by the gated /public endpoint for the current org.
      const res = await api.get(`/api/orgs/${currentOrgId}/public`)
      setOrg(res?.data || null)
    } catch (err: any) {
      if (err?.response?.status === 403) {
        setForbidden(true)
        setOrg(null)
      } else {
        message.error(err?.response?.data?.error || '加载企业属性失败')
        setOrg(null)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCurrentOrg()
  }, [currentOrgId])

  const planLabel = useMemo(() => formatPlanLabel(String(org?.plan || org?.Plan || 'free')), [org])
  const orgName = String(org?.name || org?.Name || '-')

  if (forbidden) {
    return (
      <Card style={{ borderRadius: 12 }}>
        <Result
          status="403"
          title="无权查看企业信息"
          subTitle="当前角色未被授予「查看组织信息」权限，请联系企业管理员。"
        />
      </Card>
    )
  }

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

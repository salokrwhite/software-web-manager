import { useEffect, useState } from 'react'
import api, { storeTokens } from '../api/client'

type UseOrgSwitcherOptions = {
  canSwitchOrg?: boolean
}

type OrgItem = {
  id?: string
  ID?: string
  name?: string
  Name?: string
  org_type?: string
  OrgType?: string
}

export function useOrgSwitcher(options: UseOrgSwitcherOptions = {}) {
  const { canSwitchOrg = true } = options
  const [orgs, setOrgs] = useState<OrgItem[]>([])
  const [orgsLoading, setOrgsLoading] = useState(false)
  const [currentOrgId, setCurrentOrgId] = useState<string>(sessionStorage.getItem('org_id') || '')

  const getOrgId = (org: OrgItem) => (org.id || org.ID || '').toString()
  const getOrgType = (org: OrgItem) => (org.org_type || org.OrgType || '').toLowerCase()

  const loadOrgs = async () => {
    setOrgsLoading(true)
    try {
      const res = await api.get('/api/orgs')
      const items = res.data.items || []
      setOrgs(items)
      const storedOrgId = sessionStorage.getItem('org_id') || ''
      if (storedOrgId && storedOrgId !== currentOrgId) {
        setCurrentOrgId(storedOrgId)
        if (currentOrgId) {
          window.location.href = '/dashboard'
          return
        }
      }
      if (canSwitchOrg && storedOrgId) {
        const hasCurrent = items.some((item: OrgItem) => getOrgId(item) === storedOrgId)
        if (!hasCurrent) {
          const personalOrg = items.find((item: OrgItem) => getOrgType(item) === 'personal')
          const fallbackOrgId = getOrgId(personalOrg || items[0] || {})
          if (fallbackOrgId) {
            handleSwitchOrg(fallbackOrgId)
          }
        }
      }
    } catch {
      setOrgs([])
    } finally {
      setOrgsLoading(false)
    }
  }

  useEffect(() => {
    loadOrgs()
  }, [])

  const formatOrgLabel = (org: OrgItem) => {
    const orgType = getOrgType(org)
    const name = org.name || org.Name
    if (name === '个人空间') return name
    if (orgType === 'personal') return name
    const suffix = orgType === 'enterprise' ? '企业' : '组织'
    return `${name} · ${suffix}`
  }

  const handleSwitchOrg = async (orgId: string) => {
    if (!canSwitchOrg) return
    if (!orgId || orgId === currentOrgId) return
    try {
      const res = await api.post(`/api/orgs/${orgId}/switch`)
      if (res.data.tokens) {
        storeTokens(res.data.tokens)
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
      setCurrentOrgId(res.data.org_id || orgId)
      window.location.href = '/dashboard'
    } catch {
      // ignore switch errors
    }
  }

  const showOrgSwitcher = canSwitchOrg && orgs.length > 1

  return {
    orgs,
    orgsLoading,
    currentOrgId,
    showOrgSwitcher,
    formatOrgLabel,
    handleSwitchOrg,
    reloadOrgs: loadOrgs
  }
}

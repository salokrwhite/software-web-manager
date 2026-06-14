import { Spin } from 'antd'
import { useEffect, useState } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import api from './api/client'
import AdminLogin from './pages/AdminLogin'
import ApiDocsPage from './pages/ApiDocsPage'
import ChangelogPage from './pages/ChangelogPage'
import EnterpriseRegister from './pages/EnterpriseRegister'
import Install from './pages/Install'
import InviteAccept from './pages/InviteAccept'
import LandingPage from './pages/LandingPage'
import Login from './pages/Login'
import OrgSelect from './pages/OrgSelect'
import Pending from './pages/Pending'
import PricingPage from './pages/PricingPage'
import ProductIntro from './pages/ProductIntro'
import Privacy from './pages/Privacy'
import Register from './pages/Register'
import ServiceStatusPage from './pages/ServiceStatusPage'
import SsoCallback from './pages/SsoCallback'
import Terms from './pages/Terms'
import { AdminLayout } from './layouts/AdminLayout'
import OrglessLayout from './layouts/OrglessLayout'
import { PersonalLayout } from './layouts/PersonalLayout'
import { SystemAdminLayout } from './layouts/SystemAdminLayout'
import { RequireAuth } from './routes/RequireAuth'
import { useSiteName } from './utils/siteName'

export default function App() {
  const location = useLocation()
  const siteName = useSiteName()
  const [installStatus, setInstallStatus] = useState<boolean | null>(null)
  const [checking, setChecking] = useState(true)
  const hasToken = !!sessionStorage.getItem('access_token')

  useEffect(() => {
    const checkInstallStatus = async () => {
      try {
        const res = await api.get('/api/install/status')
        setInstallStatus(res.data.installed)
      } catch (err) {
        setInstallStatus(true)
      } finally {
        setChecking(false)
      }
    }
    checkInstallStatus()
  }, [location.pathname, hasToken])

  useEffect(() => {
    document.title = siteName
  }, [siteName])

  if (checking) {
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: '#f0f2f5'
      }}>
        <Spin size="large" tip="加载中..." fullscreen />
      </div>
    )
  }

  if (location.pathname === '/install') {
    if (installStatus === false) {
      return <Install />
    }
    return <Navigate to="/login" replace />
  }

  if (installStatus === false && location.pathname !== '/install') {
    return <Navigate to="/install" replace />
  }

  if (location.pathname === '/') {
    const token = sessionStorage.getItem('access_token')
    if (token) {
      return <Navigate to="/dashboard" replace />
    }
    return <LandingPage />
  }

  if (location.pathname === '/login') {
    return <Login />
  }

  if (location.pathname === '/register') {
    return <Register />
  }

  if (location.pathname === '/enterprise-register') {
    return <EnterpriseRegister />
  }

  if (location.pathname === '/terms') {
    return <Terms />
  }

  if (location.pathname === '/privacy') {
    return <Privacy />
  }

  if (location.pathname === '/product') {
    return <ProductIntro />
  }

  if (location.pathname === '/pricing') {
    return <PricingPage />
  }

  if (location.pathname === '/changelog') {
    return <ChangelogPage />
  }

  if (location.pathname === '/api-docs') {
    return <ApiDocsPage />
  }

  if (location.pathname === '/service-status') {
    return <ServiceStatusPage />
  }

  if (location.pathname === '/admin-login') {
    return <AdminLogin />
  }

  if (location.pathname === '/sso/callback') {
    return <SsoCallback />
  }

  if (location.pathname === '/pending') {
    return <Pending />
  }

  if (location.pathname.startsWith('/invite/')) {
    return <InviteAccept />
  }

  if (location.pathname === '/org-select') {
    return (
      <RequireAuth>
        <OrgSelect />
      </RequireAuth>
    )
  }

  const systemRole = (sessionStorage.getItem('system_role') || '').toLowerCase()
  const impersonating = sessionStorage.getItem('impersonating') === 'true'
  const orgId = sessionStorage.getItem('org_id') || ''
  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()

  return (
    <RequireAuth>
      {systemRole === 'system_admin' && !impersonating ? (
        <SystemAdminLayout />
      ) : orgType === 'personal' ? (
        <PersonalLayout />
      ) : orgId === '' ? (
        <OrglessLayout />
      ) : (
        <AdminLayout />
      )}
    </RequireAuth>
  )
}

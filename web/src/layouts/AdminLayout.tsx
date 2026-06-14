import { Alert, Avatar, Button, Dropdown, Layout, Menu, Select, Space, Typography } from 'antd'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  RocketOutlined,
  SettingOutlined,
  UserOutlined
} from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import api, { clearAuthSession, scheduleTokenRefresh } from '../api/client'
import AdminRoutes from '../routes/AdminRoutes'
import { useOrgSwitcher } from '../hooks/useOrgSwitcher'
import { buildAdminMenu, getAdminOpenKeys, getAdminSelectedKey } from './menu/adminMenu'
import { useSiteName } from '../utils/siteName'
import defaultAvatar from '../assets/default-avatar.svg'

const { Header, Content, Sider } = Layout
const { Text } = Typography

export function AdminLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const siteName = useSiteName()
  const role = (sessionStorage.getItem('role') || '').toLowerCase()
  const impersonating = sessionStorage.getItem('impersonating') === 'true'
  const systemRole = (sessionStorage.getItem('system_role') || '').toLowerCase()
  const canManageUsers = role === 'admin' || role === 'owner'
  const canSwitchOrg = systemRole !== 'org_admin'
  const userEmail = sessionStorage.getItem('user_email') || ''
  const displayName = userEmail || '管理员'
  const [collapsed, setCollapsed] = useState(false)
  const [openKeys, setOpenKeys] = useState<string[]>([])
  const [orgAvatarUrl, setOrgAvatarUrl] = useState<string>(localStorage.getItem('org_avatar_url') || '')

  const {
    orgs,
    orgsLoading,
    currentOrgId,
    showOrgSwitcher,
    formatOrgLabel,
    handleSwitchOrg
  } = useOrgSwitcher({ canSwitchOrg })

  const loadOrgProfile = async () => {
    try {
      const res = await api.get('/api/profile')
      const url = res.data?.avatar_url || ''
      setOrgAvatarUrl(url)
      if (url) {
        localStorage.setItem('org_avatar_url', url)
      } else {
        localStorage.removeItem('org_avatar_url')
      }
    } catch {
      // ignore profile loading errors in header
    }
  }

  useEffect(() => {
    loadOrgProfile()
    const handler = () => loadOrgProfile()
    window.addEventListener('org-profile-updated', handler)
    return () => window.removeEventListener('org-profile-updated', handler)
  }, [])

  useEffect(() => {
    if (collapsed) {
      setOpenKeys([])
      return
    }
    setOpenKeys(getAdminOpenKeys(location.pathname))
  }, [location.pathname, collapsed])

  const menuItems = buildAdminMenu({ canManageUsers, systemRole })

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人中心'
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '系统设置'
    },
    {
      type: 'divider' as const
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true
    }
  ]

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'profile') {
      navigate('/profile')
      return
    }
    if (key === 'logout') {
      clearAuthSession()
      navigate('/login')
    }
  }

  const exitImpersonation = () => {
    const access = sessionStorage.getItem('system_backup_access_token') || ''
    const refresh = sessionStorage.getItem('system_backup_refresh_token') || ''
    const expiresAt = sessionStorage.getItem('system_backup_access_token_expires_at') || ''
    const orgId = sessionStorage.getItem('system_backup_org_id') || ''
    const role = sessionStorage.getItem('system_backup_role') || ''
    if (access) {
      sessionStorage.setItem('access_token', access)
    } else {
      clearAuthSession()
    }
    if (refresh) {
      sessionStorage.setItem('refresh_token', refresh)
    } else {
      sessionStorage.removeItem('refresh_token')
    }
    if (expiresAt) {
      sessionStorage.setItem('access_token_expires_at', expiresAt)
    } else {
      sessionStorage.removeItem('access_token_expires_at')
    }
    if (orgId) {
      sessionStorage.setItem('org_id', orgId)
    } else {
      sessionStorage.removeItem('org_id')
    }
    if (role) {
      sessionStorage.setItem('role', role)
    } else {
      sessionStorage.removeItem('role')
    }
    scheduleTokenRefresh()
    sessionStorage.removeItem('org_type')
    sessionStorage.removeItem('impersonating')
    sessionStorage.removeItem('impersonation_org_id')
    sessionStorage.removeItem('system_backup_access_token')
    sessionStorage.removeItem('system_backup_refresh_token')
    sessionStorage.removeItem('system_backup_access_token_expires_at')
    sessionStorage.removeItem('system_backup_org_id')
    sessionStorage.removeItem('system_backup_role')
    if (systemRole === 'system_admin') {
      navigate('/system/orgs')
      return
    }
    navigate('/dashboard')
  }

  return (
    <Layout style={{ minHeight: '100vh', background: '#f5f7fa' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={240}
        collapsedWidth={80}
        style={{
          background: 'rgba(255, 255, 255, 0.72)',
          backdropFilter: 'blur(20px) saturate(180%)',
          WebkitBackdropFilter: 'blur(20px) saturate(180%)',
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.08)',
          borderRight: '1px solid rgba(255, 255, 255, 0.3)',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 1000,
          overflow: 'hidden'
        }}
      >
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          <div
            style={{
              height: 64,
              display: 'flex',
              alignItems: 'center',
              justifyContent: collapsed ? 'center' : 'flex-start',
              padding: collapsed ? '0' : '0 20px',
              borderBottom: '1px solid rgba(0, 0, 0, 0.06)'
            }}
          >
            <div
              style={{
                width: 36,
                height: 36,
                background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
                borderRadius: 8,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                marginRight: collapsed ? 0 : 12
              }}
            >
              <RocketOutlined style={{ color: '#fff', fontSize: 20 }} />
            </div>
            {!collapsed && (
              <div>
                <Text strong style={{ color: '#1a1a1a', fontSize: 16, display: 'block', lineHeight: 1.2 }}>
                  {siteName}
                </Text>
              </div>
            )}
          </div>
          <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
            <Menu
              theme="light"
              mode="inline"
              selectedKeys={[getAdminSelectedKey(location.pathname)]}
              openKeys={openKeys}
              onOpenChange={(keys) => setOpenKeys(keys as string[])}
              items={menuItems}
              inlineCollapsed={collapsed}
              style={{
                borderRight: 0,
                padding: '12px 0',
                background: 'transparent'
              }}
            />
          </div>
          {showOrgSwitcher && (
            <div style={{ padding: '12px 16px', borderTop: '1px solid rgba(0, 0, 0, 0.06)', marginTop: 'auto' }}>
              <Text type="secondary" style={{ fontSize: 12 }}>切换组织</Text>
              <Select
                size="small"
                value={currentOrgId || undefined}
                loading={orgsLoading}
                onChange={handleSwitchOrg}
                style={{ width: '100%', marginTop: 6 }}
                options={orgs.map((org) => ({
                  value: org.id || org.ID,
                  label: formatOrgLabel(org)
                }))}
              />
            </div>
          )}
        </div>
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 240, transition: 'margin-left 0.2s', minWidth: 0 }}>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
            position: 'fixed',
            top: 0,
            left: collapsed ? 80 : 240,
            right: 0,
            zIndex: 999,
            height: 64,
            transition: 'left 0.2s'
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
              style={{ marginRight: 16, fontSize: 16 }}
            />
            <Text type="secondary" style={{ fontSize: 14 }}>
              {location.pathname === '/dashboard' && '概览'}
              {location.pathname === '/apps' && '应用管理'}
              {location.pathname.startsWith('/apps/') && '应用详情'}
              {location.pathname === '/analytics' && '数据分析'}
              {location.pathname === '/advanced' && '在线设备'}
              {location.pathname === '/feedback' && '用户反馈'}
              {location.pathname === '/org-attributes' && '企业属性'}
              {location.pathname === '/orgs' && '组织管理'}
              {location.pathname === '/sub-users' && '子用户'}
              {location.pathname === '/role-management' && '角色管理'}
              {location.pathname === '/member-invites' && '邀请成员'}
              {location.pathname === '/devices' && '设备列表'}
              {location.pathname === '/audit-logs' && '审计日志'}
              {location.pathname.startsWith('/tickets/new') && '提交工单'}
              {location.pathname.startsWith('/tickets') && '工单详情'}
              {location.pathname === '/docs' && '开发文档'}
              {location.pathname.startsWith('/profile') && '个人中心'}
            </Text>
          </div>
          <Space size={24}>
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              placement="bottomRight"
            >
              <Space style={{ cursor: 'pointer' }}>
                <Avatar
                  size="small"
                  src={orgAvatarUrl || defaultAvatar}
                />
                <Text style={{ fontSize: 14 }}>{displayName}</Text>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ padding: 24, marginTop: 64, minHeight: 'calc(100vh - 64px)', minWidth: 0 }}>
          {impersonating && systemRole === 'system_admin' && (
            <Alert
              message="系统管理员冒充模式"
              description="您正在以企业管理员身份操作，点击退出可返回系统后台。"
              type="warning"
              showIcon
              action={<Button size="small" onClick={exitImpersonation}>退出冒充</Button>}
              style={{ marginBottom: 16 }}
            />
          )}
          <AdminRoutes systemRole={systemRole} />
        </Content>
      </Layout>
    </Layout>
  )
}

export default AdminLayout

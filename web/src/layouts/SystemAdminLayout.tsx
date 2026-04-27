import { Avatar, Button, Dropdown, Layout, Menu, Space, Typography, theme } from 'antd'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SafetyOutlined,
  SettingOutlined,
  UserOutlined
} from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import api, { clearAuthSession } from '../api/client'
import SystemAdminRoutes from '../routes/SystemAdminRoutes'
import { buildSystemMenu, getSystemOpenKeys, getSystemSelectedKey } from './menu/systemMenu'
import { useSiteName } from '../utils/siteName'

const { Header, Content, Sider } = Layout
const { Text } = Typography

export function SystemAdminLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const siteName = useSiteName()
  const { token } = theme.useToken()
  const [collapsed, setCollapsed] = useState(false)
  const [openKeys, setOpenKeys] = useState<string[]>([])
  const [systemAvatarUrl, setSystemAvatarUrl] = useState<string>(localStorage.getItem('system_avatar_url') || '')

  const loadSystemProfile = async () => {
    try {
      const res = await api.get('/api/system/profile')
      const url = res.data?.avatar_url || ''
      setSystemAvatarUrl(url)
      if (url) {
        localStorage.setItem('system_avatar_url', url)
      } else {
        localStorage.removeItem('system_avatar_url')
      }
    } catch {
      // ignore profile loading errors in header
    }
  }

  useEffect(() => {
    loadSystemProfile()
    const handler = () => loadSystemProfile()
    window.addEventListener('system-profile-updated', handler)
    return () => window.removeEventListener('system-profile-updated', handler)
  }, [])

  useEffect(() => {
    if (collapsed) {
      setOpenKeys([])
      return
    }
    setOpenKeys(getSystemOpenKeys(location.pathname))
  }, [location.pathname, collapsed])

  const menuItems = buildSystemMenu()

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'profile') {
      navigate('/system/profile')
      return
    }
    if (key === 'settings') {
      navigate('/system/settings')
      return
    }
    if (key === 'logout') {
      clearAuthSession()
      navigate('/login')
    }
  }

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
          overflow: 'auto'
        }}
      >
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
              background: 'linear-gradient(135deg, #1f1f1f 0%, #434343 100%)',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: collapsed ? 0 : 12
            }}
          >
            <SafetyOutlined style={{ color: '#fff', fontSize: 18 }} />
          </div>
          {!collapsed && (
            <div>
              <Text strong style={{ color: '#1a1a1a', fontSize: 16, display: 'block', lineHeight: 1.2 }}>
                {siteName}
              </Text>
            </div>
          )}
        </div>
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[getSystemSelectedKey(location.pathname)]}
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
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 240, transition: 'margin-left 0.2s' }}>
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
              {location.pathname.startsWith('/system/dashboard') && '系统仪表盘'}
              {location.pathname.startsWith('/system/orgs') && '企业列表'}
              {location.pathname.startsWith('/system/approvals/orgs') && '企业审核'}
              {location.pathname.startsWith('/system/approvals/apps') && '应用审核'}
              {location.pathname === '/system/approvals' && '审核中心'}
              {location.pathname.startsWith('/system/create') && '创建企业'}
              {location.pathname.startsWith('/system/apps') && '应用管理'}
              {location.pathname.startsWith('/system/users') && '用户管理'}
              {location.pathname.startsWith('/system/settings') && '系统设置'}
              {location.pathname.startsWith('/system/audit-logs') && '系统审计日志'}
              {location.pathname.startsWith('/system/tickets/new') && '提交工单'}
              {location.pathname.startsWith('/system/tickets') && '工单详情'}
              {location.pathname.startsWith('/system/profile') && '个人中心'}
            </Text>
          </div>
          <Space size={24} align="center" style={{ display: 'flex', alignItems: 'center' }}>
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              placement="bottomRight"
            >
              <Space align="center" style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
                <Avatar
                  size="small"
                  src={systemAvatarUrl || undefined}
                  icon={systemAvatarUrl ? undefined : <UserOutlined />}
                  style={{ backgroundColor: token.colorPrimary }}
                />
                <Text style={{ fontSize: 14 }}>系统管理员</Text>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ padding: 24, marginTop: 64, minHeight: 'calc(100vh - 64px)' }}>
          <SystemAdminRoutes />
        </Content>
      </Layout>
    </Layout>
  )
}

export default SystemAdminLayout

import { Avatar, Button, Dropdown, Layout, Menu, Space, Typography } from 'antd'
import { LogoutOutlined, MenuFoldOutlined, MenuUnfoldOutlined, RocketOutlined } from '@ant-design/icons'
import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { clearAuthSession } from '../api/client'
import OrglessRoutes from '../routes/OrglessRoutes'
import { buildOrglessMenu, getOrglessSelectedKey } from './menu/orglessMenu'
import { useSiteName } from '../utils/siteName'
import defaultAvatar from '../assets/default-avatar.svg'

const { Header, Content, Sider } = Layout
const { Text } = Typography

export default function OrglessLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const siteName = useSiteName()
  const [collapsed, setCollapsed] = useState(false)

  const menuItems = buildOrglessMenu()

  const userMenuItems = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true
    }
  ]

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      clearAuthSession()
      navigate('/login')
    }
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
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[getOrglessSelectedKey()]}
          items={menuItems}
          inlineCollapsed={collapsed}
          style={{
            borderRight: 0,
            padding: '12px 0',
            background: 'transparent'
          }}
        />
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
              {location.pathname.startsWith('/dashboard') && '仪表盘'}
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
                  src={localStorage.getItem('org_avatar_url') || defaultAvatar}
                />
                <Text style={{ fontSize: 14 }}>普通用户</Text>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ padding: 24, marginTop: 64, minHeight: 'calc(100vh - 64px)', minWidth: 0 }}>
          <OrglessRoutes />
        </Content>
      </Layout>
    </Layout>
  )
}

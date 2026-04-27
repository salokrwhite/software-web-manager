import type { MenuProps } from 'antd'
import {
  AppstoreOutlined,
  BarChartOutlined,
  CodeOutlined,
  CustomerServiceOutlined,
  DashboardOutlined,
  FormOutlined,
  MessageOutlined,
  MobileOutlined,
  ProfileOutlined,
  SafetyOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined
} from '@ant-design/icons'
import { Link } from 'react-router-dom'

type BuildAdminMenuOptions = {
  canManageUsers: boolean
  systemRole: string
}

export function buildAdminMenu(options: BuildAdminMenuOptions): MenuProps['items'] {
  const { canManageUsers, systemRole } = options
  return [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: <Link to="/dashboard">概览</Link>
    },
    {
      key: '/apps',
      icon: <AppstoreOutlined />,
      label: <Link to="/apps">应用管理</Link>
    },
    {
      key: '/analytics',
      icon: <BarChartOutlined />,
      label: <Link to="/analytics">数据分析</Link>
    },
    {
      key: '/advanced',
      icon: <SettingOutlined />,
      label: <Link to="/advanced">在线设备</Link>
    },
    {
      key: '/feedback',
      icon: <MessageOutlined />,
      label: <Link to="/feedback">用户反馈</Link>
    },
    ...((systemRole === 'org_admin' || canManageUsers)
      ? [
          {
            key: 'enterprise-center',
            icon: <TeamOutlined />,
            label: '企业管理',
            children: [
              ...(systemRole === 'org_admin'
                ? [
                    {
                      key: '/org-attributes',
                      icon: <TeamOutlined />,
                      label: <Link to="/org-attributes">企业属性</Link>
                    },
                    {
                      key: '/orgs',
                      icon: <TeamOutlined />,
                      label: <Link to="/orgs">组织管理</Link>
                    }
                  ]
                : []),
              ...(canManageUsers
                ? [
                    {
                      key: '/sub-users',
                      icon: <UserOutlined />,
                      label: <Link to="/sub-users">子用户</Link>
                    },
                    {
                      key: '/role-management',
                      icon: <SafetyOutlined />,
                      label: <Link to="/role-management">角色管理</Link>
                    },
                    {
                      key: '/member-invites',
                      icon: <UserOutlined />,
                      label: <Link to="/member-invites">邀请成员</Link>
                    }
                  ]
                : [])
            ]
          }
        ]
      : []),
    {
      key: '/devices',
      icon: <MobileOutlined />,
      label: <Link to="/devices">设备列表</Link>
    },
    {
      key: '/audit-logs',
      icon: <SafetyOutlined />,
      label: <Link to="/audit-logs">审计日志</Link>
    },
    {
      key: 'ticket-center',
      icon: <CustomerServiceOutlined />,
      label: '工单中心',
      children: [
        {
          key: '/tickets',
          icon: <ProfileOutlined />,
          label: <Link to="/tickets">工单详情</Link>
        },
        {
          key: '/tickets/new',
          icon: <FormOutlined />,
          label: <Link to="/tickets/new">提交工单</Link>
        }
      ]
    },
    {
      key: '/docs',
      icon: <CodeOutlined />,
      label: <Link to="/docs">开发文档</Link>
    }
  ]
}

export function getAdminSelectedKey(pathname: string) {
  if (pathname.startsWith('/apps')) return '/apps'
  if (pathname.startsWith('/analytics')) return '/analytics'
  if (pathname.startsWith('/advanced')) return '/advanced'
  if (pathname.startsWith('/feedback')) return '/feedback'
  if (pathname.startsWith('/org-attributes')) return '/org-attributes'
  if (pathname.startsWith('/orgs')) return '/orgs'
  if (pathname.startsWith('/sub-users')) return '/sub-users'
  if (pathname.startsWith('/role-management')) return '/role-management'
  if (pathname.startsWith('/member-invites')) return '/member-invites'
  if (pathname.startsWith('/devices')) return '/devices'
  if (pathname.startsWith('/audit-logs')) return '/audit-logs'
  if (pathname.startsWith('/tickets/new')) return '/tickets/new'
  if (pathname.startsWith('/tickets')) return '/tickets'
  if (pathname.startsWith('/docs')) return '/docs'
  if (pathname.startsWith('/dashboard')) return '/dashboard'
  return '/dashboard'
}

export function getAdminOpenKeys(pathname: string) {
  if (pathname.startsWith('/tickets')) return ['ticket-center']
  if (
    pathname.startsWith('/orgs') ||
    pathname.startsWith('/org-attributes') ||
    pathname.startsWith('/sub-users') ||
    pathname.startsWith('/role-management') ||
    pathname.startsWith('/member-invites')
  ) {
    return ['enterprise-center']
  }
  return []
}

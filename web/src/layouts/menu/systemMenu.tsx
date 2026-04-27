import type { MenuProps } from 'antd'
import {
  AppstoreOutlined,
  CustomerServiceOutlined,
  DashboardOutlined,
  FormOutlined,
  PlusOutlined,
  ProfileOutlined,
  SafetyOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined
} from '@ant-design/icons'
import { Link } from 'react-router-dom'

export function buildSystemMenu(): MenuProps['items'] {
  return [
    {
      key: '/system/dashboard',
      icon: <DashboardOutlined />,
      label: <Link to="/system/dashboard">系统仪表盘</Link>
    },
    {
      key: '/system/orgs',
      icon: <TeamOutlined />,
      label: <Link to="/system/orgs">企业列表</Link>
    },
    {
      key: 'system-approval-center',
      icon: <SafetyOutlined />,
      label: '审核中心',
      children: [
        {
          key: '/system/approvals/orgs',
          icon: <TeamOutlined />,
          label: <Link to="/system/approvals/orgs">企业审核</Link>
        },
        {
          key: '/system/approvals/apps',
          icon: <AppstoreOutlined />,
          label: <Link to="/system/approvals/apps">应用审核</Link>
        }
      ]
    },
    {
      key: '/system/create',
      icon: <PlusOutlined />,
      label: <Link to="/system/create">创建企业</Link>
    },
    {
      key: '/system/apps',
      icon: <AppstoreOutlined />,
      label: <Link to="/system/apps">应用管理</Link>
    },
    {
      key: '/system/users',
      icon: <UserOutlined />,
      label: <Link to="/system/users">用户管理</Link>
    },
    {
      key: '/system/audit-logs',
      icon: <SafetyOutlined />,
      label: <Link to="/system/audit-logs">系统审计日志</Link>
    },
    {
      key: 'system-ticket-center',
      icon: <CustomerServiceOutlined />,
      label: '工单中心',
      children: [
        {
          key: '/system/tickets',
          icon: <ProfileOutlined />,
          label: <Link to="/system/tickets">工单详情</Link>
        },
        {
          key: '/system/tickets/new',
          icon: <FormOutlined />,
          label: <Link to="/system/tickets/new">提交工单</Link>
        }
      ]
    },
    {
      key: '/system/settings',
      icon: <SettingOutlined />,
      label: <Link to="/system/settings">系统设置</Link>
    }
  ]
}

export function getSystemSelectedKey(pathname: string) {
  if (pathname.startsWith('/system/dashboard')) return '/system/dashboard'
  if (pathname.startsWith('/system/approvals/apps')) return '/system/approvals/apps'
  if (pathname.startsWith('/system/approvals/orgs')) return '/system/approvals/orgs'
  if (pathname.startsWith('/system/approvals')) return '/system/approvals/orgs'
  if (pathname.startsWith('/system/create')) return '/system/create'
  if (pathname.startsWith('/system/apps')) return '/system/apps'
  if (pathname.startsWith('/system/users')) return '/system/users'
  if (pathname.startsWith('/system/settings')) return '/system/settings'
  if (pathname.startsWith('/system/audit-logs')) return '/system/audit-logs'
  if (pathname.startsWith('/system/tickets/new')) return '/system/tickets/new'
  if (pathname.startsWith('/system/tickets')) return '/system/tickets'
  if (pathname.startsWith('/system/orgs')) return '/system/orgs'
  return '/system/dashboard'
}

export function getSystemOpenKeys(pathname: string) {
  if (pathname.startsWith('/system/approvals')) return ['system-approval-center']
  if (pathname.startsWith('/system/tickets')) return ['system-ticket-center']
  return []
}

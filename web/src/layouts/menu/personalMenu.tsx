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
  TeamOutlined
} from '@ant-design/icons'
import { Link } from 'react-router-dom'

export function buildPersonalMenu(): MenuProps['items'] {
  return [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: <Link to="/dashboard">仪表盘</Link>
    },
    {
      key: '/apps',
      icon: <AppstoreOutlined />,
      label: <Link to="/apps">应用管理</Link>
    },
    {
      key: '/devices',
      icon: <MobileOutlined />,
      label: <Link to="/devices">设备列表</Link>
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
    {
      key: '/audit-logs',
      icon: <SafetyOutlined />,
      label: <Link to="/audit-logs">审计日志</Link>
    },
    {
      key: '/join-org',
      icon: <TeamOutlined />,
      label: <Link to="/join-org">加入企业</Link>
    },
    {
      key: '/enterprise-upgrade',
      icon: <SafetyOutlined />,
      label: <Link to="/enterprise-upgrade">升级企业认证</Link>
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

export function getPersonalSelectedKey(pathname: string) {
  if (pathname.startsWith('/apps')) return '/apps'
  if (pathname.startsWith('/devices')) return '/devices'
  if (pathname.startsWith('/analytics')) return '/analytics'
  if (pathname.startsWith('/advanced')) return '/advanced'
  if (pathname.startsWith('/feedback')) return '/feedback'
  if (pathname.startsWith('/audit-logs')) return '/audit-logs'
  if (pathname.startsWith('/join-org')) return '/join-org'
  if (pathname.startsWith('/enterprise-upgrade')) return '/enterprise-upgrade'
  if (pathname.startsWith('/tickets/new')) return '/tickets/new'
  if (pathname.startsWith('/tickets')) return '/tickets'
  if (pathname.startsWith('/docs')) return '/docs'
  if (pathname.startsWith('/dashboard')) return '/dashboard'
  return '/dashboard'
}

export function getPersonalOpenKeys(pathname: string) {
  if (pathname.startsWith('/tickets')) return ['ticket-center']
  return []
}

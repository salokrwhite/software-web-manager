import type { MenuProps } from 'antd'
import { DashboardOutlined } from '@ant-design/icons'
import { Link } from 'react-router-dom'

export function buildOrglessMenu(): MenuProps['items'] {
  return [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: <Link to="/dashboard">仪表盘</Link>
    }
  ]
}

export function getOrglessSelectedKey() {
  return '/dashboard'
}

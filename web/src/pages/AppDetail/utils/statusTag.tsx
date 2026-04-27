import { Tag } from 'antd'
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  PauseCircleOutlined,
  SafetyOutlined,
  SyncOutlined
} from '@ant-design/icons'

export const getStatusTag = (status: string) => {
  switch (status) {
    case 'published':
      return <Tag icon={<CheckCircleOutlined />} color="success">已发布</Tag>
    case 'approved':
      return <Tag icon={<SafetyOutlined />} color="processing">已审批</Tag>
    case 'in_review':
      return <Tag icon={<SyncOutlined spin />} color="processing">审核中</Tag>
    case 'rejected':
      return <Tag color="error">已拒绝</Tag>
    case 'revoked':
      return <Tag color="warning">已撤销</Tag>
    case 'draft':
      return <Tag icon={<ClockCircleOutlined />} color="default">草稿</Tag>
    default:
      return <Tag icon={<PauseCircleOutlined />}>{status}</Tag>
  }
}

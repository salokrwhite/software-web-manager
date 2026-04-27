import { Card, Space, Table, Tag, Typography } from 'antd'
import dayjs from 'dayjs'
import { parseJSONValue, parseRegionRules, parseWhitelist } from '../utils/parse'
import { hasRegionRules } from '../utils/region'
import { summarizeTargetingRules } from '../utils/targeting'

const { Text } = Typography

type ReleasePolicySummaryCardProps = {
  releaseChannels: any[]
  title?: string
}

const formatWindow = (start?: string, end?: string) => {
  if (!start || !end) return '长期有效'
  return `${dayjs(start).format('YYYY-MM-DD HH:mm')} ~ ${dayjs(end).format('YYYY-MM-DD HH:mm')}`
}

export default function ReleasePolicySummaryCard({
  releaseChannels,
  title = '发布策略摘要（统一视图）'
}: ReleasePolicySummaryCardProps) {
  const rows = (releaseChannels || []).map((item: any) => {
    const targetingRules = parseJSONValue(item.targeting_rules)
    const regionRules = parseRegionRules(item.region_rules)
    const whitelist = parseWhitelist(item.whitelist)
    const targetingSummary = summarizeTargetingRules(targetingRules)
    const regionSummary = hasRegionRules(regionRules) ? '通道自定义区域' : '继承应用级区域'
    const isScheduled = String(item.status || '').toLowerCase() === 'scheduled'
    const scheduledAt = item.published_at ? dayjs(item.published_at).format('YYYY-MM-DD HH:mm') : ''
    const timeSummary = isScheduled
      ? `预约激活 ${scheduledAt || '-'}；窗口 ${formatWindow(item.rollout_start_at, item.rollout_end_at)}`
      : formatWindow(item.rollout_start_at, item.rollout_end_at)

    return {
      ...item,
      _policy_time: timeSummary,
      _policy_audience: {
        regionSummary,
        targetingSummary
      },
      _policy_traffic: {
        rollout: Number(item.rollout_percent || 100),
        whitelistCount: whitelist.length,
        paused: !!item.paused
      }
    }
  })

  return (
    <Card
      size="small"
      title={title}
      style={{ borderRadius: 12, marginBottom: 16 }}
    >
      <Table
        rowKey="id"
        size="small"
        pagination={{ pageSize: 5 }}
        dataSource={rows}
        columns={[
          { title: '渠道', dataIndex: 'channel_code', width: 110 },
          { title: '版本', dataIndex: 'release_version', width: 140 },
          {
            title: '时间策略',
            dataIndex: '_policy_time',
            render: (v: string) => <Text>{v}</Text>
          },
          {
            title: '人群策略',
            dataIndex: '_policy_audience',
            render: (v: any) => (
              <Space wrap>
                <Tag color="blue">{v.regionSummary}</Tag>
                {v.targetingSummary.length > 0
                  ? v.targetingSummary.map((item: string) => <Tag key={item}>{item}</Tag>)
                  : <Tag>目标规则不限</Tag>}
              </Space>
            )
          },
          {
            title: '流量策略',
            dataIndex: '_policy_traffic',
            render: (v: any) => (
              <Space wrap>
                <Tag color="green">灰度 {v.rollout}%</Tag>
                <Tag>白名单 {v.whitelistCount}</Tag>
                {v.paused ? <Tag color="orange">已暂停</Tag> : <Tag color="success">运行中</Tag>}
              </Space>
            )
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (v: string) => v === 'active' ? <Tag color="success">active</Tag> : <Tag>{v || '-'}</Tag>
          }
        ]}
      />
    </Card>
  )
}

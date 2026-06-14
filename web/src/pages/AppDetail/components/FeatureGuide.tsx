import { Alert, Card, Collapse, Space, Steps, Tag, Typography } from 'antd'
import { BulbOutlined, ReadOutlined } from '@ant-design/icons'
import { useState, type ReactNode } from 'react'

const { Text, Paragraph } = Typography

export type GuideStep = {
  title: string
  description: ReactNode
}

type FeatureGuideProps = {
  /** 唯一标识，用于记住用户的折叠选择 */
  storageKey: string
  /** 功能标题，例如「灰度策略」 */
  title: string
  /** 一句话说明这个功能是做什么的 */
  summary: ReactNode
  /** 典型操作流程 */
  steps: GuideStep[]
  /** 小贴士（可选） */
  tips?: ReactNode[]
}

const STORAGE_PREFIX = 'swm.guide.collapsed.'

export default function FeatureGuide({ storageKey, title, summary, steps, tips }: FeatureGuideProps) {
  const key = STORAGE_PREFIX + storageKey
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    try {
      return localStorage.getItem(key) === '1'
    } catch {
      return false
    }
  })

  const persist = (next: boolean) => {
    setCollapsed(next)
    try {
      localStorage.setItem(key, next ? '1' : '0')
    } catch {
      /* ignore */
    }
  }

  return (
    <Collapse
      activeKey={collapsed ? [] : ['guide']}
      onChange={(keys) => persist((keys as string[]).length === 0)}
      style={{ marginBottom: 16, background: '#f0f7ff', borderColor: '#bae0ff' }}
      items={[
        {
          key: 'guide',
          label: (
            <Space>
              <ReadOutlined style={{ color: '#1677ff' }} />
              <Text strong>新手引导 · {title}</Text>
              <Text type="secondary" style={{ fontWeight: 400 }}>
                {collapsed ? '点击展开使用说明' : '点击收起'}
              </Text>
            </Space>
          ),
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Paragraph style={{ margin: 0 }}>{summary}</Paragraph>

              <Card size="small" title="操作流程" style={{ borderRadius: 8 }}>
                <Steps
                  direction="vertical"
                  size="small"
                  current={-1}
                  items={steps.map((s) => ({
                    title: s.title,
                    description: s.description,
                    status: 'process' as const
                  }))}
                />
              </Card>

              {tips && tips.length > 0 && (
                <Alert
                  type="success"
                  showIcon
                  icon={<BulbOutlined />}
                  message="小贴士"
                  description={
                    <ul style={{ margin: 0, paddingLeft: 18 }}>
                      {tips.map((tip, i) => (
                        <li key={i}>{tip}</li>
                      ))}
                    </ul>
                  }
                />
              )}
            </Space>
          )
        }
      ]}
    />
  )
}

/** 引导卡片里用于强调操作名的小标签 */
export function GuideTag({ children }: { children: ReactNode }) {
  return <Tag color="blue" style={{ marginInline: 2 }}>{children}</Tag>
}

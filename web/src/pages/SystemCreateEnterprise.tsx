import { useEffect, useState } from 'react'
import { Button, Card, Form, Input, Select, Space, Typography, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Title, Text } = Typography
const { Option } = Select
const DEFAULT_PLAN_TYPES = ['free', 'team', 'enterprise']

const formatPlanLabel = (plan: string) => {
  const value = (plan || '').toLowerCase()
  if (value === 'team') return 'Team'
  if (value === 'enterprise') return 'Enterprise'
  return 'Free'
}

export default function SystemCreateEnterprise() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [planTypes, setPlanTypes] = useState<string[]>(DEFAULT_PLAN_TYPES)
  const [form] = Form.useForm()

  const loadPlanTypes = async () => {
    try {
      const res = await api.get('/api/system/settings')
      const items = Array.isArray(res?.data?.org_plan_types) ? res.data.org_plan_types : []
      const normalized = items
        .map((item: string) => (item || '').toLowerCase().trim())
        .filter((item: string) => item === 'free' || item === 'team' || item === 'enterprise')
      setPlanTypes(normalized.length > 0 ? normalized : DEFAULT_PLAN_TYPES)
    } catch {
      setPlanTypes(DEFAULT_PLAN_TYPES)
    }
  }

  useEffect(() => {
    loadPlanTypes()
  }, [])

  const onSubmit = async () => {
    try {
      const values = await form.validateFields()
      setLoading(true)
      await api.post('/api/system/orgs', values)
      message.success('企业已创建')
      navigate('/system/orgs')
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>创建企业</Title>
        <Text type="secondary">创建企业组织与企业管理员账号</Text>
      </Space>

      <Card style={{ width: '100%' }}>
        <Form layout="vertical" form={form}>
          <Form.Item name="org_name" label="企业名称" rules={[{ required: true, message: '请输入企业名称' }]}>
            <Input placeholder="企业名称" />
          </Form.Item>
          <Form.Item name="owner_email" label="企业管理员邮箱" rules={[{ required: true, message: '请输入管理员邮箱' }, { type: 'email', message: '邮箱格式错误' }]}>
            <Input placeholder="admin@company.com" />
          </Form.Item>
          <Form.Item name="password" label="管理员初始密码" rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '至少 6 位' }]}>
            <Input.Password placeholder="初始密码" />
          </Form.Item>
          <Form.Item name="plan" label="计划(可选)">
            <Select allowClear placeholder="不选择则默认 Free">
              {planTypes.map((plan) => (
                <Option key={plan} value={plan}>
                  {formatPlanLabel(plan)}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={onSubmit} loading={loading}>创建</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

import { useState } from 'react'
import { Button, Card, Form, Input, message, Typography, Steps, Result, Alert, Divider } from 'antd'
import { useNavigate } from 'react-router-dom'
import {
  LockOutlined,
  MailOutlined,
} from '@ant-design/icons'
import api, { storeTokens } from '../api/client'

const { Title, Text, Paragraph } = Typography
const { Step } = Steps

interface DatabaseFormValues {
  db_host: string
  db_port: string
  db_name: string
  db_user: string
  db_password: string
}

interface AdminFormValues {
  email: string
  password: string
  confirmPassword: string
}

export default function Install() {
  const navigate = useNavigate()
  const [dbForm] = Form.useForm()
  const [adminForm] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [installSuccess, setInstallSuccess] = useState(false)
  const [dbConfig, setDbConfig] = useState<DatabaseFormValues | null>(null)

  const features = [
    { title: '安全可靠', desc: '企业级数据加密保护' },
    { title: '全球分发', desc: 'CDN 加速全球访问' },
    { title: '团队协作', desc: '多角色权限管理' },
    { title: '版本管理', desc: '完整的版本生命周期' }
  ]

  const steps = [
    { title: '欢迎' },
    { title: '数据库配置' },
    { title: '管理员配置' },
    { title: '完成' }
  ]

  const handleDbConfig = async (values: DatabaseFormValues) => {
    setLoading(true)
    try {
      // 测试数据库连接
      const res = await api.post('/api/install/test-db', values)
      if (res.data.success) {
        setDbConfig(values)
        setCurrentStep(2)
        message.success('数据库连接成功')
      } else {
        message.error(res.data.error || '数据库连接失败')
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '数据库连接失败，请检查配置')
    } finally {
      setLoading(false)
    }
  }

  const handleInstall = async (values: AdminFormValues) => {
    if (values.password !== values.confirmPassword) {
      message.error('两次输入的密码不一致')
      return
    }

    if (!dbConfig) {
      message.error('请先配置数据库')
      return
    }

    setLoading(true)
    try {
      const res = await api.post('/api/install', {
        ...dbConfig,
        email: values.email,
        password: values.password
      })

      storeTokens(res.data.tokens)
      sessionStorage.removeItem('org_id')
      sessionStorage.removeItem('role')
      sessionStorage.removeItem('org_type')
      sessionStorage.removeItem('impersonating')
      sessionStorage.removeItem('impersonation_org_id')
      sessionStorage.removeItem('system_backup_access_token')
      sessionStorage.removeItem('system_backup_refresh_token')
      sessionStorage.removeItem('system_backup_org_id')
      sessionStorage.removeItem('system_backup_role')
      if (res.data.user?.email) {
        sessionStorage.setItem('user_email', res.data.user.email)
      } else {
        sessionStorage.removeItem('user_email')
      }
      if (res.data.system_role) {
        sessionStorage.setItem('system_role', res.data.system_role)
      } else {
        sessionStorage.removeItem('system_role')
      }

      setInstallSuccess(true)
      setCurrentStep(3)
      message.success('系统安装成功！')
    } catch (err: any) {
      message.error(err?.response?.data?.error || '安装失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const goToDashboard = () => {
    navigate('/dashboard')
  }

  const renderWelcomeStep = () => (
    <div style={{ textAlign: 'center', padding: '40px 0' }}>
      <Title level={3} style={{ marginBottom: 16 }}>欢迎使用 SWM</Title>
      <Paragraph style={{ fontSize: 16, color: '#666', maxWidth: 480, margin: '0 auto 40px' }}>
        Software Web Manager 是企业级应用版本管理平台。在开始之前，我们需要进行一些基本配置。
      </Paragraph>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24, textAlign: 'left', maxWidth: 560, margin: '0 auto 40px' }}>
        {features.map((item, index) => (
          <div key={index} style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
            <div
              style={{
                width: 32,
                height: 32,
                background: '#1890ff',
                borderRadius: '50%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 14,
                color: '#fff',
                fontWeight: 600,
                flexShrink: 0
              }}
            >
              {index + 1}
            </div>
            <div>
              <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>{item.title}</div>
              <div style={{ fontSize: 13, color: '#888' }}>{item.desc}</div>
            </div>
          </div>
        ))}
      </div>

      <Button
        type="primary"
        size="large"
        onClick={() => setCurrentStep(1)}
        style={{ height: 48, padding: '0 48px', fontSize: 16 }}
      >
        开始安装
      </Button>
    </div>
  )

  const renderDbConfigStep = () => (
    <div style={{ maxWidth: 480, margin: '0 auto', padding: '20px 0' }}>
      <div style={{ marginBottom: 32 }}>
        <Title level={4} style={{ marginBottom: 8 }}>数据库配置</Title>
      </div>

      <Form
        form={dbForm}
        layout="vertical"
        onFinish={handleDbConfig}
        size="large"
        initialValues={{
          db_host: 'localhost',
          db_port: '3306',
          db_name: 'swmanager'
        }}
      >
        <Form.Item
          name="db_host"
          label="数据库主机"
          rules={[{ required: true, message: '请输入数据库主机' }]}
        >
          <Input placeholder="localhost" />
        </Form.Item>
        <Form.Item
          name="db_port"
          label="端口"
          rules={[{ required: true, message: '请输入端口' }]}
        >
          <Input placeholder="3306" />
        </Form.Item>
        <Form.Item
          name="db_name"
          label="数据库名称"
          rules={[{ required: true, message: '请输入数据库名称' }]}
        >
          <Input placeholder="swmanager" />
        </Form.Item>
        <Form.Item
          name="db_user"
          label="用户名"
          rules={[{ required: true, message: '请输入用户名' }]}
        >
          <Input placeholder="root" />
        </Form.Item>
        <Form.Item
          name="db_password"
          label="密码"
          rules={[{ required: true, message: '请输入密码' }]}
        >
          <Input.Password placeholder="数据库密码" />
        </Form.Item>

        <div style={{ display: 'flex', gap: 12, marginTop: 24 }}>
          <Button onClick={() => setCurrentStep(0)} style={{ flex: 1 }}>
            上一步
          </Button>
          <Button
            type="primary"
            htmlType="submit"
            loading={loading}
            style={{ flex: 1 }}
          >
            测试连接
          </Button>
        </div>
      </Form>
    </div>
  )

  const renderAdminConfigStep = () => (
    <div style={{ maxWidth: 480, margin: '0 auto', padding: '20px 0' }}>
      <div style={{ marginBottom: 32 }}>
        <Title level={4} style={{ marginBottom: 8 }}>管理员配置</Title>
        <Text type="secondary">创建系统管理员账户</Text>
      </div>

      <Form
        form={adminForm}
        layout="vertical"
        onFinish={handleInstall}
        size="large"
      >
        <Form.Item
          name="email"
          label="管理员邮箱"
          rules={[
            { required: true, message: '请输入邮箱' },
            { type: 'email', message: '请输入有效的邮箱地址' }
          ]}
        >
          <Input
            prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
            placeholder="admin@example.com"
          />
        </Form.Item>
        <Form.Item
          name="password"
          label="密码"
          rules={[
            { required: true, message: '请输入密码' },
            { min: 8, message: '密码至少8位' }
          ]}
        >
          <Input.Password
            prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
            placeholder="设置密码（至少8位）"
          />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label="确认密码"
          rules={[
            { required: true, message: '请确认密码' },
            { min: 8, message: '密码至少8位' }
          ]}
        >
          <Input.Password
            prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
            placeholder="再次输入密码"
          />
        </Form.Item>

        <div style={{ display: 'flex', gap: 12, marginTop: 24 }}>
          <Button onClick={() => setCurrentStep(1)} style={{ flex: 1 }}>
            上一步
          </Button>
          <Button
            type="primary"
            htmlType="submit"
            loading={loading}
            style={{ flex: 1 }}
          >
            完成安装
          </Button>
        </div>
      </Form>
    </div>
  )

  const renderSuccessStep = () => (
    <div style={{ textAlign: 'center', padding: '60px 0' }}>
      <Result
        status="success"
        title="安装成功！"
        subTitle="系统已初始化完成，管理员账户已创建"
        extra={[
          <Button type="primary" key="dashboard" size="large" onClick={goToDashboard}>
            进入管理后台
          </Button>
        ]}
      />
    </div>
  )

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return renderWelcomeStep()
      case 1:
        return renderDbConfigStep()
      case 2:
        return renderAdminConfigStep()
      case 3:
        return renderSuccessStep()
      default:
        return renderWelcomeStep()
    }
  }

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: '#f0f2f5' }}>
      <div
        style={{
          flex: 1,
          background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
          padding: '60px',
          color: '#fff',
          position: 'relative',
          overflow: 'hidden'
        }}
      >
        <div
          style={{
            position: 'absolute',
            top: -100,
            right: -100,
            width: 400,
            height: 400,
            background: 'rgba(255,255,255,0.1)',
            borderRadius: '50%'
          }}
        />
        <div
          style={{
            position: 'absolute',
            bottom: -150,
            left: -150,
            width: 500,
            height: 500,
            background: 'rgba(255,255,255,0.08)',
            borderRadius: '50%'
          }}
        />

        <div style={{ textAlign: 'center', zIndex: 1, maxWidth: 480 }}>
          <Title level={2} style={{ color: '#fff', marginBottom: 16, fontSize: 36 }}>
            SWM 软件版本管理平台
          </Title>
          <Paragraph style={{ color: 'rgba(255,255,255,0.9)', fontSize: 16, marginBottom: 48 }}>
            企业级应用版本管理解决方案，助力您的软件交付流程
          </Paragraph>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24, textAlign: 'left' }}>
            {features.map((item, index) => (
              <div key={index} style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
                <div
                  style={{
                    width: 28,
                    height: 28,
                    background: 'rgba(255,255,255,0.2)',
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 14,
                    color: '#fff',
                    fontWeight: 600,
                    flexShrink: 0
                  }}
                >
                  {index + 1}
                </div>
                <div>
                  <div style={{ fontSize: 15, fontWeight: 600, marginBottom: 4 }}>{item.title}</div>
                  <div style={{ fontSize: 13, color: 'rgba(255,255,255,0.7)' }}>{item.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div style={{ position: 'absolute', bottom: 40, color: 'rgba(255,255,255,0.6)', fontSize: 13 }}>
          © 2024 SWM Software Web Manager. All rights reserved.
        </div>
      </div>

      <div
        style={{
          width: 600,
          background: '#fff',
          display: 'flex',
          flexDirection: 'column',
          padding: '40px 32px',
          boxShadow: '-4px 0 20px rgba(0,0,0,0.05)'
        }}
      >
        <Steps
          current={currentStep}
          style={{ marginBottom: 32 }}
          labelPlacement="horizontal"
          responsive={false}
        >
          {steps.map((step, index) => (
            <Step key={index} title={step.title} />
          ))}
        </Steps>

        <div style={{ flex: 1, overflow: 'auto' }}>
          {renderStepContent()}
        </div>
      </div>
    </div>
  )
}

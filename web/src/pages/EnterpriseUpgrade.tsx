import { useEffect, useState } from 'react'
import { Alert, Button, Card, Checkbox, Form, Input, Modal, Steps, Upload, message, Typography, Space } from 'antd'
import { BankOutlined, MailOutlined, UploadOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import api from '../api/client'

const { Title, Text } = Typography

const maxMaterialSize = 20 * 1024 * 1024

export default function EnterpriseUpgrade() {
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [fileList, setFileList] = useState<any[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [approvedApps, setApprovedApps] = useState<any[]>([])
  const [otherOrgs, setOtherOrgs] = useState<any[]>([])
  const orgId = sessionStorage.getItem('org_id') || ''
  const userEmail = sessionStorage.getItem('user_email') || ''

  const stepFields: string[][] = [
    ['org_name'],
    [],
    ['agreement'],
    []
  ]

  useEffect(() => {
    const loadMigrationData = async () => {
      try {
        const [appsRes, orgsRes] = await Promise.all([
          api.get('/api/apps'),
          api.get('/api/orgs')
        ])
        const apps = Array.isArray(appsRes?.data?.items) ? appsRes.data.items : []
        const activeApps = apps.filter((item: any) => {
          const status = String(item?.Status || item?.status || '').toLowerCase()
          return status === 'active'
        })
        setApprovedApps(activeApps)

        const orgs = Array.isArray(orgsRes?.data?.items) ? orgsRes.data.items : []
        const joinedOthers = orgs.filter((item: any) => {
          const itemOrgId = String(item?.id || item?.ID || '')
          const orgType = String(item?.org_type || item?.OrgType || '').toLowerCase()
          return itemOrgId !== orgId && orgType !== 'personal'
        })
        setOtherOrgs(joinedOthers)
      } catch {
        // ignore data migration precheck failures
      }
    }
    void loadMigrationData()
  }, [orgId])

  const goNext = async () => {
    try {
      await form.validateFields(stepFields[currentStep])
      setCurrentStep((prev) => prev + 1)
    } catch {
      return
    }
  }

  const goPrev = () => {
    setCurrentStep((prev) => Math.max(0, prev - 1))
  }

  const confirmRiskIfNeeded = () =>
    new Promise<boolean>((resolve) => {
      Modal.confirm({
        title: '确认不保留已通过应用？',
        content: '您当前未勾选“保留现有已通过应用”。继续提交后，个人空间下当前创建的所有应用都将被删除（无论是否审核通过）。请确认是否继续提交。',
        okText: '确认提交',
        cancelText: '返回检查',
        onOk: () => resolve(true),
        onCancel: () => resolve(false)
      })
    })

  const handleSubmit = async () => {
    try {
      await form.validateFields()
      if (fileList.length === 0) {
        message.error('请上传企业材料')
        return
      }
      const oversized = fileList.find((file) => (file.size || file.originFileObj?.size || 0) > maxMaterialSize)
      if (oversized) {
        message.error('材料文件大小不能超过 20MB')
        return
      }
      const values = form.getFieldsValue(true)
      const orgNameValue = (values.org_name || '').trim()
      const keepApprovedApps = values.keep_approved_apps === true

      if (otherOrgs.length > 0) {
        message.error('请先退出其他组织后再升级企业认证')
        setCurrentStep(3)
        return
      }
      if (approvedApps.length > 0 && !keepApprovedApps) {
        const confirmed = await confirmRiskIfNeeded()
        if (!confirmed) {
          setCurrentStep(3)
          return
        }
      }

      const formData = new FormData()
      if (orgNameValue) {
        formData.append('org_name', orgNameValue)
      }
      formData.append('keep_approved_apps', String(keepApprovedApps))
      fileList.forEach((file) => {
        const raw = file.originFileObj || file
        formData.append('materials', raw)
      })
      setSubmitting(true)
      const res = await api.post('/api/orgs/upgrade', formData)
      message.success('企业认证申请已提交')
      const nextOrgId = res?.data?.org?.id || orgId
      navigate(`/pending?id=${nextOrgId}`)
    } catch (err: any) {
      if (err?.errorFields) {
        return
      }
      message.error(err?.response?.data?.error || '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div style={{ width: '100%' }}>
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        <div>
          <Title level={3} style={{ marginBottom: 4 }}>升级企业认证</Title>
          <Text type="secondary">提交企业信息与材料，审核通过后账号升级为企业管理员。</Text>
        </div>

        <Card>
          <Steps
            current={currentStep}
            items={[
              { title: '企业信息' },
              { title: '管理员账号' },
              { title: '材料与协议' },
              { title: '数据迁移' }
            ]}
            style={{ marginBottom: 24 }}
            responsive={false}
          />
          <Form form={form} layout="vertical" size="large">
            {currentStep === 0 && (
              <Form.Item
                name="org_name"
                label="企业名称"
                rules={[{ required: true, message: '请输入企业名称' }]}
              >
                <Input
                  prefix={<BankOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder="企业/组织名称"
                />
              </Form.Item>
            )}

            {currentStep === 1 && (
              <Form.Item label="企业管理员邮箱">
                <Input
                  value={userEmail}
                  prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                  disabled
                />
                <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                  当前登录账号将作为企业管理员
                </Text>
              </Form.Item>
            )}

            {currentStep === 2 && (
              <>
                <Form.Item label="企业材料">
                  <Upload
                    multiple
                    beforeUpload={() => false}
                    fileList={fileList}
                    onChange={({ fileList: nextList }) => setFileList(nextList)}
                  >
                    <Button icon={<UploadOutlined />}>选择材料</Button>
                  </Upload>
                  <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                    必传材料，单个文件不超过 20MB
                  </Text>
                </Form.Item>
                <Form.Item
                  name="agreement"
                  valuePropName="checked"
                  rules={[
                    { validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('请阅读并同意服务条款与隐私政策')) }
                  ]}
                >
                  <Checkbox>我已阅读并同意服务条款与隐私政策</Checkbox>
                </Form.Item>
              </>
            )}

            {currentStep === 3 && (
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Alert
                  type="info"
                  showIcon
                  message="数据迁移"
                  description="升级企业认证后，个人空间下的数据将按企业组织规则管理。"
                />

                {approvedApps.length > 0 ? (
                  <>
                    <Alert
                      type="warning"
                      showIcon
                      message={`检测到 ${approvedApps.length} 个已通过应用`}
                      description="请确认升级后是否保留这些已通过应用。"
                    />
                    <Form.Item
                      name="keep_approved_apps"
                      valuePropName="checked"
                    >
                      <Checkbox>我确认升级后保留现有已通过应用</Checkbox>
                    </Form.Item>
                  </>
                ) : (
                  <Alert
                    type="success"
                    showIcon
                    message="未检测到已通过应用"
                    description="升级后无需处理应用保留。"
                  />
                )}

                {otherOrgs.length > 0 && (
                  <Alert
                    type="error"
                    showIcon
                    message="检测到您已加入其他组织"
                    description="请先退出其他组织后才可以升级企业认证。"
                  />
                )}
              </Space>
            )}

            <Space style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8 }}>
              <Button onClick={goPrev} disabled={currentStep === 0}>
                上一步
              </Button>
              {currentStep < 3 ? (
                <Button type="primary" onClick={goNext}>
                  下一步
                </Button>
              ) : (
                <Button type="primary" onClick={handleSubmit} loading={submitting} disabled={otherOrgs.length > 0}>
                  提交认证申请
                </Button>
              )}
            </Space>
          </Form>
        </Card>
      </Space>
    </div>
  )
}

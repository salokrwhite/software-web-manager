import { Button, Card, Form, Input, message, Alert, Radio, Select } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeftOutlined } from '@ant-design/icons'
import api from '../api/client'

export default function ReleaseCreate() {
  const { id } = useParams()
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const [locked, setLocked] = useState(false)
  const [lockReason, setLockReason] = useState('')
  const [releaseTemplates, setReleaseTemplates] = useState<any[]>([])
  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()
  const isPersonal = orgType === 'personal'

  const goBack = () => {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      navigate(-1)
      return
    }
    navigate(`/apps/${id}`)
  }

  useEffect(() => {
    const loadApp = async () => {
      if (!id || !isPersonal) return
      try {
        const res = await api.get(`/api/apps/${id}`)
        const status = (res.data?.app?.Status || res.data?.app?.status || '').toLowerCase()
        const reason = res.data?.app?.RejectionReason || res.data?.app?.rejection_reason || ''
        if (status && status !== 'active') {
          setLocked(true)
          setLockReason(status === 'rejected' ? (reason || '未提供驳回理由') : '')
        }
      } catch {
        // ignore
      }
    }
    loadApp()
  }, [id, isPersonal])

  useEffect(() => {
    const loadTemplates = async () => {
      try {
        const res = await api.get('/api/release-templates')
        setReleaseTemplates((res.data.items || []).map((t: any) => ({
          id: t.ID || t.id,
          name: t.Name || t.name
        })))
      } catch {
        // ignore
      }
    }
    loadTemplates()
  }, [])

  const onSubmit = async () => {
    if (locked) {
      message.warning('应用待审核，暂不可操作')
      return
    }
    try {
      const values = await form.validateFields()
      const payload: any = {
        version: values.version
      }
      if (values.version_code !== undefined && values.version_code !== null && String(values.version_code).trim() !== '') {
        payload.version_code = Number(values.version_code)
      }
      if (values.release_template_id) {
        payload.release_template_id = values.release_template_id
      }
      payload.notes = values.notes
      if (values.package_mode === 'external_link') {
        payload.external_download_url = values.external_download_url
      } else {
        payload.external_download_url = ''
      }
      await api.post(`/api/apps/${id}/releases`, payload)
      message.success('创建成功')
      navigate(`/apps/${id}`)
    } catch (err: any) {
      message.error(err?.response?.data?.error || '创建失败')
    }
  }

  return (
    <Card
      title="新建版本"
      extra={(
        <Button icon={<ArrowLeftOutlined />} onClick={goBack}>
          返回上一页
        </Button>
      )}
    >
      {locked && (
        <Alert
          style={{ marginBottom: 16 }}
          type={lockReason ? 'error' : 'warning'}
          showIcon
          message="该应用正在审核中，暂不可新建版本。"
          description={lockReason || undefined}
        />
      )}
      <Form layout="vertical" form={form} initialValues={{ package_mode: 'upload' }}>
        <Form.Item name="version" label="版本号" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item
          name="version_code"
          label="内部版本号"
          rules={[
            {
              validator: (_, value) => {
                if (value === undefined || value === null || String(value).trim() === '') {
                  return Promise.resolve()
                }
                if (/^\d+$/.test(String(value).trim())) {
                  return Promise.resolve()
                }
                return Promise.reject(new Error('请输入数字'))
              }
            }
          ]}
        >
          <Input inputMode="numeric" placeholder="例如：1001" />
        </Form.Item>
        <Form.Item name="release_template_id" label="发布模板">
          <Select
            allowClear
            placeholder="选择发布模板"
            options={releaseTemplates.map((t) => ({ label: t.name, value: t.id }))}
          />
        </Form.Item>
        <Form.Item name="package_mode" label="软件包来源">
          <Radio.Group
            onChange={(e) => {
              const mode = e.target.value
              if (mode === 'upload') {
                form.setFieldsValue({ external_download_url: '' })
              }
            }}
          >
            <Radio value="upload">需要上传软件包</Radio>
            <Radio value="external_link">不上传，使用外部下载链接</Radio>
          </Radio.Group>
        </Form.Item>
        <Form.Item shouldUpdate noStyle>
          {({ getFieldValue }) => (
            getFieldValue('package_mode') === 'external_link' ? (
              <Form.Item
                name="external_download_url"
                label="外部下载链接"
                extra="客户端确认更新后将打开该链接进行下载。"
                rules={[
                  { required: true, message: '请输入外部下载链接' },
                  { type: 'url', message: '请输入有效的 URL' }
                ]}
              >
                <Input placeholder="https://example.com/download/latest" />
              </Form.Item>
            ) : null
          )}
        </Form.Item>
        <Form.Item
          name="notes"
          label="更新说明"
          rules={[{ required: true, message: '请输入更新说明' }]}
        >
          <Input.TextArea rows={4} />
        </Form.Item>
        <Button type="primary" onClick={onSubmit} disabled={locked}>创建</Button>
      </Form>
    </Card>
  )
}

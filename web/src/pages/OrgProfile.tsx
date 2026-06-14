import { Avatar, Button, Card, Form, Input, Space, Typography, Upload, message } from 'antd'
import type { UploadProps } from 'antd'
import { LockOutlined, UploadOutlined, UserOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import api, { getErrorMessage } from '../api/client'
import { useSSOConfig } from '../utils/ssoConfig'

const { Title, Text } = Typography

type ProfileResponse = {
  id: string
  email: string
  avatar_url?: string
  sso_bound?: boolean
}

const MAX_AVATAR_SIZE = 2 * 1024 * 1024

const cropImageToSquare = (file: File, size: number) =>
  new Promise<Blob>((resolve, reject) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      const cropSize = Math.min(img.width, img.height)
      const sx = (img.width - cropSize) / 2
      const sy = (img.height - cropSize) / 2
      const canvas = document.createElement('canvas')
      canvas.width = size
      canvas.height = size
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        URL.revokeObjectURL(url)
        reject(new Error('无法处理图片'))
        return
      }
      ctx.drawImage(img, sx, sy, cropSize, cropSize, 0, 0, size, size)
      canvas.toBlob((blob) => {
        URL.revokeObjectURL(url)
        if (!blob) {
          reject(new Error('无法生成头像'))
          return
        }
        resolve(blob)
      }, 'image/png')
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('无法读取图片'))
    }
    img.src = url
  })

export default function OrgProfile() {
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [avatarUrl, setAvatarUrl] = useState<string>(localStorage.getItem('org_avatar_url') || '')
  const [loading, setLoading] = useState(false)
  const [avatarUploading, setAvatarUploading] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [ssoBinding, setSsoBinding] = useState(false)
  const [ssoUnbinding, setSsoUnbinding] = useState(false)
  const [passwordForm] = Form.useForm()
  const sso = useSSOConfig()
  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()
  const systemRole = (sessionStorage.getItem('system_role') || '').toLowerCase()
  const profileHint = systemRole === 'org_admin'
    ? '管理企业管理员的账号信息'
    : orgType === 'personal'
      ? '管理用户的账号信息'
      : '管理账号信息'

  const loadProfile = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/profile')
      const data = res.data as ProfileResponse
      setProfile(data)
      const url = data.avatar_url || ''
      setAvatarUrl(url)
      if (url) {
        localStorage.setItem('org_avatar_url', url)
      } else {
        localStorage.removeItem('org_avatar_url')
      }
    } catch (err: any) {
      message.error(getErrorMessage(err, '加载个人信息失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadProfile()
  }, [])

  const uploadAvatar = async (file: File) => {
    setAvatarUploading(true)
    try {
      const blob = await cropImageToSquare(file, 256)
      const formData = new FormData()
      formData.append('avatar', blob, 'avatar.png')
      const res = await api.post('/api/profile/avatar', formData)
      const url = res.data?.avatar_url || ''
      setAvatarUrl(url)
      if (url) {
        localStorage.setItem('org_avatar_url', url)
      } else {
        localStorage.removeItem('org_avatar_url')
      }
      window.dispatchEvent(new Event('org-profile-updated'))
      message.success('头像已更新')
    } catch (err: any) {
      message.error(getErrorMessage(err, err?.message || '头像上传失败'))
    } finally {
      setAvatarUploading(false)
    }
  }

  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    if (!['image/jpeg', 'image/png'].includes(file.type)) {
      message.error('仅支持 JPG/PNG 格式')
      return Upload.LIST_IGNORE
    }
    if (file.size > MAX_AVATAR_SIZE) {
      message.error('头像不能超过 2MB')
      return Upload.LIST_IGNORE
    }
    void uploadAvatar(file)
    return false
  }

  const handleUpdatePassword = async () => {
    const values = await passwordForm.validateFields()
    if (values.new_password !== values.confirm_password) {
      message.error('两次输入的密码不一致')
      return
    }
    setPasswordLoading(true)
    try {
      await api.post('/api/profile/password', {
        current_password: values.current_password,
        new_password: values.new_password
      })
      message.success('密码已更新')
      passwordForm.resetFields()
    } catch (err: any) {
      message.error(getErrorMessage(err, '修改密码失败'))
    } finally {
      setPasswordLoading(false)
    }
  }

  const bindSSO = async () => {
    setSsoBinding(true)
    try {
      const res = await api.get('/api/profile/sso/bind', { params: { redirect: window.location.pathname } })
      const url = res.data?.authorize_url
      if (!url) {
        throw new Error('missing authorize_url')
      }
      window.location.href = url
    } catch (err: any) {
      message.error(getErrorMessage(err, '无法发起 SSO 绑定'))
      setSsoBinding(false)
    }
  }

  const unbindSSO = async () => {
    setSsoUnbinding(true)
    try {
      await api.post('/api/profile/sso/unbind')
      message.success('已解绑 SSO')
      await loadProfile()
    } catch (err: any) {
      message.error(getErrorMessage(err, '解绑失败'))
    } finally {
      setSsoUnbinding(false)
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>个人中心</Title>
        <Text type="secondary">{profileHint}</Text>
      </Space>

      <Card loading={loading} style={{ marginBottom: 16 }}>
        <Space size={24} align="center" wrap>
          <Avatar
            size={72}
            src={avatarUrl || undefined}
            icon={avatarUrl ? undefined : <UserOutlined />}
          />
          <Space direction="vertical" size={6}>
            <Text>邮箱：{profile?.email || '-'}</Text>
            <Text>用户ID：{profile?.id || '-'}</Text>
            <Upload
              accept="image/png,image/jpeg"
              showUploadList={false}
              beforeUpload={beforeUpload}
            >
              <Button icon={<UploadOutlined />} loading={avatarUploading}>更换头像</Button>
            </Upload>
            <Text type="secondary">支持 JPG/PNG，自动转换为 WebP，裁剪为 256x256，最大 2MB</Text>
          </Space>
        </Space>
      </Card>

      {sso.enabled && (
        <Card title="SSO 单点登录" loading={loading} style={{ marginBottom: 16 }}>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Text type="secondary">
              {profile?.sso_bound ? '已绑定 SSO 账号，可使用单点登录方式登录。' : '尚未绑定 SSO 账号，绑定后可使用单点登录方式登录。'}
            </Text>
            {profile?.sso_bound ? (
              <Button danger loading={ssoUnbinding} onClick={unbindSSO}>解绑 SSO</Button>
            ) : (
              <Button type="primary" loading={ssoBinding} onClick={bindSSO}>绑定 {sso.displayName}</Button>
            )}
          </Space>
        </Card>
      )}

      <Card title="修改密码">
        <Form form={passwordForm} layout="vertical" onFinish={handleUpdatePassword}>
          <Form.Item
            name="current_password"
            label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="请输入当前密码" />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码至少 6 位' }
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label="确认新密码"
            rules={[{ required: true, message: '请确认新密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="再次输入新密码" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={passwordLoading}>
            更新密码
          </Button>
        </Form>
      </Card>
    </div>
  )
}

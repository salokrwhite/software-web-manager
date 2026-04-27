import { Avatar, Button, Card, Form, Input, Space, Typography, Upload, message } from 'antd'
import type { UploadProps } from 'antd'
import { LockOutlined, UploadOutlined, UserOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import api from '../api/client'

const { Title, Text } = Typography

type ProfileResponse = {
  id: string
  email: string
  avatar_url?: string
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

export default function SystemProfile() {
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [avatarUrl, setAvatarUrl] = useState<string>(localStorage.getItem('system_avatar_url') || '')
  const [loading, setLoading] = useState(false)
  const [avatarUploading, setAvatarUploading] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [passwordForm] = Form.useForm()

  const loadProfile = async () => {
    setLoading(true)
    try {
      const res = await api.get('/api/system/profile')
      const data = res.data as ProfileResponse
      setProfile(data)
      const url = data.avatar_url || ''
      setAvatarUrl(url)
      if (url) {
        localStorage.setItem('system_avatar_url', url)
      } else {
        localStorage.removeItem('system_avatar_url')
      }
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载个人信息失败')
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
      const res = await api.post('/api/system/profile/avatar', formData)
      const url = res.data?.avatar_url || ''
      setAvatarUrl(url)
      if (url) {
        localStorage.setItem('system_avatar_url', url)
      } else {
        localStorage.removeItem('system_avatar_url')
      }
      window.dispatchEvent(new Event('system-profile-updated'))
      message.success('头像已更新')
    } catch (err: any) {
      message.error(err?.response?.data?.error || err?.message || '头像上传失败')
    } finally {
      setAvatarUploading(false)
    }
  }

  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      message.error('仅支持 JPG/PNG/WebP 格式')
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
      await api.post('/api/system/profile/password', {
        current_password: values.current_password,
        new_password: values.new_password
      })
      message.success('密码已更新')
      passwordForm.resetFields()
    } catch (err: any) {
      message.error(err?.response?.data?.error || '修改密码失败')
    } finally {
      setPasswordLoading(false)
    }
  }

  return (
    <div>
      <Space direction="vertical" size={4} style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>个人中心</Title>
        <Text type="secondary">管理系统管理员的账号信息</Text>
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
              accept="image/*"
              showUploadList={false}
              beforeUpload={beforeUpload}
            >
              <Button icon={<UploadOutlined />} loading={avatarUploading}>更换头像</Button>
            </Upload>
            <Text type="secondary">支持 JPG/PNG/WebP，自动裁剪为 256x256，最大 2MB</Text>
          </Space>
        </Space>
      </Card>

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

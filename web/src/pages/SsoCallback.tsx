import { Spin, message } from 'antd'
import { useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import api, { storeTokens } from '../api/client'
import { getSafeRedirectPath } from '../utils/redirect'

const ERROR_MESSAGES: Record<string, string> = {
  sso_account_not_provisioned: '该账号尚未在本系统注册，请联系管理员开通后再使用 SSO 登录',
  sso_disabled: 'SSO 单点登录未开启',
  sso_not_configured: 'SSO 配置不完整，请联系管理员',
  sso_store_unavailable: 'SSO 服务暂时不可用，请稍后再试',
  sso_state_expired: '登录会话已过期，请重新发起 SSO 登录',
  sso_state_invalid: '登录状态校验失败，请重新登录',
  sso_invalid_request: 'SSO 回调参数缺失',
  sso_token_exchange_failed: 'SSO 令牌交换失败，请重试',
  sso_id_token_invalid: 'SSO 身份令牌校验失败',
  sso_missing_sub: '未能从 SSO 获取用户标识',
  access_denied: '授权被拒绝',
  user_disabled: '账号已停用，请联系系统管理员',
  org_disabled: '组织已停用，请联系系统管理员',
  user_no_org: '当前账号未归属任何组织，请联系管理员',
  sso_already_bound: '该 SSO 账号已被其他用户绑定',
  sso_error: 'SSO 登录失败，请重试'
}

export default function SsoCallback() {
  const navigate = useNavigate()
  const handledRef = useRef(false)

  useEffect(() => {
    if (handledRef.current) return
    handledRef.current = true

    const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash
    const params = new URLSearchParams(hash)
    // Clear sensitive tokens from the address bar immediately.
    window.history.replaceState(null, '', window.location.pathname + window.location.search)

    const hasSession = !!sessionStorage.getItem('access_token')

    const error = params.get('error')
    if (error) {
      if (error === 'user_pending' || error === 'org_pending') {
        message.info('账号待审核，请联系系统管理员')
        navigate('/pending', { replace: true })
        return
      }
      message.error(ERROR_MESSAGES[error] || 'SSO 登录失败，请重试')
      if (hasSession) {
        navigate(getSafeRedirectPath(params.get('redirect'), '/dashboard'), { replace: true })
      } else {
        navigate('/login', { replace: true })
      }
      return
    }

    if (params.get('sso_bound') === '1') {
      message.success('已绑定 SSO 账号')
      navigate(getSafeRedirectPath(params.get('redirect'), '/dashboard'), { replace: true })
      return
    }

    const accessToken = params.get('access_token')
    const refreshToken = params.get('refresh_token')
    if (!accessToken || !refreshToken) {
      message.error('SSO 登录失败，请重试')
      navigate('/login', { replace: true })
      return
    }

    storeTokens({
      access_token: accessToken,
      refresh_token: refreshToken,
      expires_in: Number(params.get('expires_in')) || undefined
    })

    const setOrRemove = (key: string, value: string | null) => {
      if (value) {
        sessionStorage.setItem(key, value)
      } else {
        sessionStorage.removeItem(key)
      }
    }
    setOrRemove('org_id', params.get('org_id'))
    setOrRemove('role', params.get('role'))
    setOrRemove('user_email', params.get('email'))
    setOrRemove('org_type', params.get('org_type'))
    setOrRemove('system_role', (params.get('system_role') || '').toLowerCase() || null)
    sessionStorage.removeItem('impersonating')
    sessionStorage.removeItem('impersonation_org_id')
    sessionStorage.removeItem('system_backup_access_token')
    sessionStorage.removeItem('system_backup_refresh_token')
    sessionStorage.removeItem('system_backup_org_id')
    sessionStorage.removeItem('system_backup_role')

    const safeRedirect = getSafeRedirectPath(params.get('redirect'), '/dashboard')

    const finish = async () => {
      try {
        const orgRes = await api.get('/api/orgs')
        const items = orgRes?.data?.items || []
        if (items.length > 1) {
          navigate(`/org-select?redirect=${encodeURIComponent(safeRedirect)}`, { replace: true })
          return
        }
      } catch {
        // ignore org list errors and fall back to default redirect
      }
      navigate(safeRedirect, { replace: true })
    }
    void finish()
  }, [navigate])

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <Spin size="large" tip="正在完成 SSO 登录..." fullscreen />
    </div>
  )
}

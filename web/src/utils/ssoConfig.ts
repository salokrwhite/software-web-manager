import { useEffect, useState } from 'react'
import api from '../api/client'

export type SSOPublicConfig = {
  enabled: boolean
  displayName: string
}

const DEFAULT_DISPLAY_NAME = 'SSO 单点登录'

// useSSOConfig reads the public SSO toggle/label from /api/public/settings.
// Only sso_enabled and sso_display_name are exposed publicly (no secrets).
export const useSSOConfig = (): SSOPublicConfig => {
  const [config, setConfig] = useState<SSOPublicConfig>({ enabled: false, displayName: DEFAULT_DISPLAY_NAME })

  useEffect(() => {
    let active = true
    api
      .get('/api/public/settings')
      .then((res) => {
        if (!active) return
        setConfig({
          enabled: res?.data?.sso_enabled === true,
          displayName: String(res?.data?.sso_display_name || '').trim() || DEFAULT_DISPLAY_NAME
        })
      })
      .catch(() => {
        // keep defaults (disabled) on failure
      })
    return () => {
      active = false
    }
  }, [])

  return config
}

// startSSOLogin asks the backend for an authorization URL (with state/nonce/PKCE
// prepared server-side) and navigates the browser to the IdP.
export const startSSOLogin = async (redirect: string) => {
  const res = await api.get('/api/auth/sso/login', { params: { redirect } })
  const url = res?.data?.authorize_url
  if (!url) {
    throw new Error('missing authorize_url')
  }
  window.location.href = url
}

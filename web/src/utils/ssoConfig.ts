import { useEffect, useState } from 'react'
import api, { clearAuthSession, getSSOIdToken } from '../api/client'

export type SSOPublicConfig = {
  enabled: boolean
  displayName: string
}

const DEFAULT_DISPLAY_NAME = 'SSO 单点登录'
const SSO_CONFIG_STORAGE_KEY = 'sso_config'

let ssoConfigRequest: Promise<SSOPublicConfig> | null = null

const readCachedSSOConfig = (): SSOPublicConfig | null => {
  try {
    const raw = localStorage.getItem(SSO_CONFIG_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    return {
      enabled: parsed?.enabled === true,
      displayName: String(parsed?.displayName || '').trim() || DEFAULT_DISPLAY_NAME
    }
  } catch {
    return null
  }
}

// fetchSSOConfig reads the public SSO toggle/label from /api/public/settings,
// dedupes concurrent callers, and caches the result so the UI can render the
// last-known value immediately instead of flashing the disabled state.
// Only sso_enabled and sso_display_name are exposed publicly (no secrets).
export const fetchSSOConfig = async (): Promise<SSOPublicConfig> => {
  if (ssoConfigRequest) {
    return ssoConfigRequest
  }
  ssoConfigRequest = (async () => {
    try {
      const res = await api.get('/api/public/settings')
      const config: SSOPublicConfig = {
        enabled: res?.data?.sso_enabled === true,
        displayName: String(res?.data?.sso_display_name || '').trim() || DEFAULT_DISPLAY_NAME
      }
      localStorage.setItem(SSO_CONFIG_STORAGE_KEY, JSON.stringify(config))
      return config
    } catch {
      // keep last-known config on failure instead of forcing disabled
      return readCachedSSOConfig() || { enabled: false, displayName: DEFAULT_DISPLAY_NAME }
    } finally {
      ssoConfigRequest = null
    }
  })()
  return ssoConfigRequest
}

export const useSSOConfig = (): SSOPublicConfig => {
  const [config, setConfig] = useState<SSOPublicConfig>(
    () => readCachedSSOConfig() || { enabled: false, displayName: DEFAULT_DISPLAY_NAME }
  )

  useEffect(() => {
    let active = true
    fetchSSOConfig().then((next) => {
      if (active) {
        setConfig(next)
      }
    })
    return () => {
      active = false
    }
  }, [])

  return config
}

// logoutWithSSO ends the session. When the current session was established via
// SSO (an IdP id_token is kept), it clears the local session and redirects the
// browser to the IdP's OIDC logout endpoint so the SSO session is terminated too
// — otherwise the IdP would silently log the user straight back in.
// Returns true when the browser is being redirected to the IdP (the caller
// should not navigate itself); false means only a local logout happened.
export const logoutWithSSO = async (): Promise<boolean> => {
  const idToken = getSSOIdToken()
  if (idToken) {
    try {
      const res = await api.get('/api/auth/sso/logout', { params: { id_token_hint: idToken } })
      const url = res?.data?.logout_url
      if (url) {
        clearAuthSession()
        window.location.href = url
        return true
      }
    } catch {
      // fall back to a local-only logout below
    }
  }
  clearAuthSession()
  return false
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

import axios from 'axios'

const STORAGE = sessionStorage
const AUTH_KEYS = {
  access: 'access_token',
  refresh: 'refresh_token',
  expiresAt: 'access_token_expires_at',
  orgId: 'org_id',
  role: 'role',
  userEmail: 'user_email',
  orgType: 'org_type',
  systemRole: 'system_role',
  impersonating: 'impersonating',
  impersonationOrgId: 'impersonation_org_id',
  systemBackupAccess: 'system_backup_access_token',
  systemBackupRefresh: 'system_backup_refresh_token',
  systemBackupExpiresAt: 'system_backup_access_token_expires_at',
  systemBackupOrgId: 'system_backup_org_id',
  systemBackupRole: 'system_backup_role'
} as const

const AUTH_MIGRATION_KEYS = [
  AUTH_KEYS.access,
  AUTH_KEYS.refresh,
  AUTH_KEYS.expiresAt,
  AUTH_KEYS.orgId,
  AUTH_KEYS.role,
  AUTH_KEYS.userEmail,
  AUTH_KEYS.orgType,
  AUTH_KEYS.systemRole,
  AUTH_KEYS.impersonating,
  AUTH_KEYS.impersonationOrgId,
  AUTH_KEYS.systemBackupAccess,
  AUTH_KEYS.systemBackupRefresh,
  AUTH_KEYS.systemBackupExpiresAt,
  AUTH_KEYS.systemBackupOrgId,
  AUTH_KEYS.systemBackupRole
]

const SIGN_VERSION = 'v1'

const textEncoder = new TextEncoder()

const toBaseUrl = () => import.meta.env.VITE_API_BASE || 'http://localhost:8080'

const rfc3986Encode = (value: string) =>
  encodeURIComponent(value).replace(/[!'()*]/g, (ch) => `%${ch.charCodeAt(0).toString(16).toUpperCase()}`)

const toHex = (buffer: ArrayBuffer) => Array.from(new Uint8Array(buffer)).map((b) => b.toString(16).padStart(2, '0')).join('')

const toArrayBuffer = (bytes: Uint8Array) => {
  const buffer = new ArrayBuffer(bytes.byteLength)
  new Uint8Array(buffer).set(bytes)
  return buffer
}

const sha256Hex = async (bytes: Uint8Array) => {
  const digest = await crypto.subtle.digest('SHA-256', toArrayBuffer(bytes))
  return toHex(digest)
}

const hmacSha256Hex = async (keyText: string, dataText: string) => {
  const key = await crypto.subtle.importKey(
    'raw',
    toArrayBuffer(textEncoder.encode(keyText)),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  const sig = await crypto.subtle.sign('HMAC', key, toArrayBuffer(textEncoder.encode(dataText)))
  return toHex(sig)
}

const decodeBase64Url = (input: string) => {
  const normalized = input.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new TextDecoder().decode(bytes)
}

const extractJwtSub = (token: string) => {
  const parts = token.split('.')
  if (parts.length < 2) return ''
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1] || ''))
    return String(payload?.sub || payload?.uid || '').trim()
  } catch {
    return ''
  }
}

const isAnonymousApiPath = (path: string) => {
  if (path === '/api/install' || path.startsWith('/api/install/')) return true
  if (path.startsWith('/api/auth/')) return true
  if (path === '/api/public/settings') return true
  if (/^\/api\/org-invites\/[^/]+(?:\/accept)?$/.test(path)) return true
  return false
}

const shouldSignJwtRequest = (path: string) => {
  if (!path.startsWith('/api')) return false
  if (path.startsWith('/api/system')) return true
  return !isAnonymousApiPath(path)
}

const appendParamValues = (params: URLSearchParams, key: string, value: any) => {
  if (value === undefined || value === null) return
  if (Array.isArray(value)) {
    for (const item of value) appendParamValues(params, key, item)
    return
  }
  params.append(key, String(value))
}

const buildCanonicalQuery = (urlText: string, params: any, baseURL: string) => {
  const url = new URL(urlText, baseURL)
  const merged = new URLSearchParams(url.search)
  if (params) {
    if (params instanceof URLSearchParams) {
      params.forEach((value, key) => merged.append(key, value))
    } else if (typeof params === 'object') {
      Object.entries(params).forEach(([key, value]) => appendParamValues(merged, key, value))
    }
  }
  const entries: Array<[string, string]> = []
  merged.forEach((value, key) => entries.push([key, value]))
  entries.sort((a, b) => {
    if (a[0] < b[0]) return -1
    if (a[0] > b[0]) return 1
    if (a[1] < b[1]) return -1
    if (a[1] > b[1]) return 1
    return 0
  })
  return entries.map(([k, v]) => `${rfc3986Encode(k)}=${rfc3986Encode(v)}`).join('&')
}

const concatUint8Arrays = (parts: Uint8Array[]) => {
  const total = parts.reduce((sum, arr) => sum + arr.length, 0)
  const out = new Uint8Array(total)
  let offset = 0
  for (const arr of parts) {
    out.set(arr, offset)
    offset += arr.length
  }
  return out
}

const serializeFormData = async (form: FormData) => {
  const boundary = `----swm-${crypto.randomUUID().replace(/-/g, '')}`
  const chunks: Uint8Array[] = []
  for (const [name, value] of form.entries()) {
    chunks.push(textEncoder.encode(`--${boundary}\r\n`))
    if (typeof value === 'string') {
      chunks.push(textEncoder.encode(`Content-Disposition: form-data; name="${name}"\r\n\r\n`))
      chunks.push(textEncoder.encode(value))
      chunks.push(textEncoder.encode('\r\n'))
      continue
    }
    const fileName = (value as File).name || 'blob'
    const contentType = value.type || 'application/octet-stream'
    chunks.push(textEncoder.encode(`Content-Disposition: form-data; name="${name}"; filename="${fileName}"\r\n`))
    chunks.push(textEncoder.encode(`Content-Type: ${contentType}\r\n\r\n`))
    const fileBytes = new Uint8Array(await value.arrayBuffer())
    chunks.push(fileBytes)
    chunks.push(textEncoder.encode('\r\n'))
  }
  chunks.push(textEncoder.encode(`--${boundary}--\r\n`))
  const body = concatUint8Arrays(chunks)
  return {
    body,
    contentType: `multipart/form-data; boundary=${boundary}`
  }
}

const normalizeBody = async (config: any) => {
  const data = config.data
  if (data === undefined || data === null) {
    return { bytes: new Uint8Array(0) }
  }
  if (typeof data === 'string') {
    return { bytes: textEncoder.encode(data), data }
  }
  if (data instanceof URLSearchParams) {
    const serialized = data.toString()
    return { bytes: textEncoder.encode(serialized), data: serialized, contentType: 'application/x-www-form-urlencoded;charset=utf-8' }
  }
  if (data instanceof FormData) {
    const serialized = await serializeFormData(data)
    return { bytes: serialized.body, data: new Blob([serialized.body]), contentType: serialized.contentType }
  }
  if (data instanceof Blob) {
    const bytes = new Uint8Array(await data.arrayBuffer())
    return { bytes, data }
  }
  if (data instanceof ArrayBuffer) {
    const bytes = new Uint8Array(data)
    return { bytes, data }
  }
  if (ArrayBuffer.isView(data)) {
    const view = data as ArrayBufferView
    const bytes = new Uint8Array(view.buffer, view.byteOffset, view.byteLength)
    return { bytes, data }
  }
  const serialized = JSON.stringify(data)
  return { bytes: textEncoder.encode(serialized), data: serialized, contentType: 'application/json' }
}

export const migrateLegacyAuth = () => {
  const hasSession = !!STORAGE.getItem(AUTH_KEYS.access)
  if (hasSession) {
    return
  }
  let migrated = false
  for (const key of AUTH_MIGRATION_KEYS) {
    const value = localStorage.getItem(key)
    if (value !== null) {
      STORAGE.setItem(key, value)
      localStorage.removeItem(key)
      migrated = true
    }
  }
  if (migrated) {
    const expires = STORAGE.getItem(AUTH_KEYS.expiresAt)
    if (!expires) {
      STORAGE.removeItem(AUTH_KEYS.expiresAt)
    }
  }
}

export const getAccessToken = () => STORAGE.getItem(AUTH_KEYS.access) || ''
export const getRefreshToken = () => STORAGE.getItem(AUTH_KEYS.refresh) || ''
export const getAccessTokenExpiresAt = () => {
  const raw = STORAGE.getItem(AUTH_KEYS.expiresAt)
  if (!raw) return 0
  const parsed = Number(raw)
  return Number.isFinite(parsed) ? parsed : 0
}

export const storeTokens = (tokens: { access_token: string; refresh_token: string; expires_in?: number }) => {
  STORAGE.setItem(AUTH_KEYS.access, tokens.access_token)
  STORAGE.setItem(AUTH_KEYS.refresh, tokens.refresh_token)
  if (typeof tokens.expires_in === 'number' && tokens.expires_in > 0) {
    const expiresAt = Date.now() + tokens.expires_in * 1000
    STORAGE.setItem(AUTH_KEYS.expiresAt, String(expiresAt))
  } else {
    STORAGE.removeItem(AUTH_KEYS.expiresAt)
  }
  scheduleTokenRefresh()
}

export const clearAuthSession = () => {
  for (const key of AUTH_MIGRATION_KEYS) {
    STORAGE.removeItem(key)
  }
}

export const api = axios.create({
  baseURL: toBaseUrl()
})

api.interceptors.request.use(async (config) => {
  const token = getAccessToken()
  config.headers = config.headers || {}
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
    const baseURL = config.baseURL || toBaseUrl()
    const requestURI = api.getUri(config)
    const absolute = new URL(requestURI, baseURL)
    if (!shouldSignJwtRequest(absolute.pathname)) {
      return config
    }

    const sub = extractJwtSub(token)
    if (!sub) {
      return config
    }

    const normalized = await normalizeBody(config)
    if (normalized.data !== undefined) {
      config.data = normalized.data
    }
    if (normalized.contentType && !config.headers['Content-Type']) {
      config.headers['Content-Type'] = normalized.contentType
    }
    const bodyHash = await sha256Hex(normalized.bytes)
    const ts = Math.floor(Date.now() / 1000)
    const nonce = crypto.randomUUID()
    const canonicalQuery = buildCanonicalQuery(absolute.toString(), null, baseURL)
    const canonical = [
      (config.method || 'GET').toUpperCase(),
      absolute.pathname,
      canonicalQuery,
      bodyHash,
      String(ts),
      nonce,
      sub
    ].join('\n')
    const signature = await hmacSha256Hex(token, canonical)

    config.headers['X-Timestamp'] = String(ts)
    config.headers['X-Nonce'] = nonce
    config.headers['X-Signature'] = signature
    config.headers['X-Sign-Version'] = SIGN_VERSION
  }
  return config
})

const refreshClient = axios.create({
  baseURL: toBaseUrl()
})

let refreshPromise: Promise<string | null> | null = null
let refreshTimer: number | null = null
const REFRESH_BUFFER_MS = 60_000

const refreshAccessToken = async () => {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return null
  if (refreshPromise) return refreshPromise
  refreshPromise = refreshClient
    .post('/api/auth/refresh', { refresh_token: refreshToken })
    .then((res) => {
      const tokens = res?.data?.tokens
      const data = res?.data || {}
      if (tokens?.access_token && tokens?.refresh_token) {
        storeTokens(tokens)
      }
      if (Object.prototype.hasOwnProperty.call(data, 'org_id')) {
        if (data.org_id) {
          STORAGE.setItem(AUTH_KEYS.orgId, data.org_id)
        } else {
          STORAGE.removeItem(AUTH_KEYS.orgId)
        }
      }
      if (Object.prototype.hasOwnProperty.call(data, 'role')) {
        if (data.role) {
          STORAGE.setItem(AUTH_KEYS.role, data.role)
        } else {
          STORAGE.removeItem(AUTH_KEYS.role)
        }
      }
      if (Object.prototype.hasOwnProperty.call(data, 'org_type')) {
        if (data.org_type) {
          STORAGE.setItem(AUTH_KEYS.orgType, data.org_type)
        } else {
          STORAGE.removeItem(AUTH_KEYS.orgType)
        }
      }
      if (Object.prototype.hasOwnProperty.call(data, 'system_role')) {
        if (data.system_role) {
          STORAGE.setItem(AUTH_KEYS.systemRole, data.system_role)
        } else {
          STORAGE.removeItem(AUTH_KEYS.systemRole)
        }
      }
      if (tokens?.access_token) {
        return tokens.access_token as string
      }
      return null
    })
    .catch(() => null)
    .finally(() => {
      refreshPromise = null
    })
  return refreshPromise
}

export const scheduleTokenRefresh = () => {
  if (refreshTimer) {
    window.clearTimeout(refreshTimer)
    refreshTimer = null
  }
  const expiresAt = getAccessTokenExpiresAt()
  if (!expiresAt) return
  const delay = Math.max(0, expiresAt - Date.now() - REFRESH_BUFFER_MS)
  refreshTimer = window.setTimeout(async () => {
    const token = await refreshAccessToken()
    if (!token) {
      clearAuthSession()
    }
  }, delay)
}

api.interceptors.response.use(
  (resp) => resp,
  async (error) => {
    const status = error?.response?.status
    const original = error?.config
    if (status === 401 && original && !original._retry) {
      original._retry = true
      const token = await refreshAccessToken()
      if (token) {
        original.headers = original.headers || {}
        original.headers.Authorization = `Bearer ${token}`
        return api(original)
      }
      clearAuthSession()
    }
    return Promise.reject(error)
  }
)

migrateLegacyAuth()
scheduleTokenRefresh()

export default api

const ERROR_I18N_MAP: Record<string, string> = {
  'invalid credentials': '账号或密码错误',
  'user not active': '账号未激活',
  'user has no org': '未加入组织，请联系管理员加入组织',
  'org not active': '组织未激活或已停用',
  'failed to issue token': '登录失败，请稍后重试',
  'admin login required': '请使用管理员入口登录',
  'email already registered': '邮箱已注册',
  'email_code_required': '请输入邮箱验证码',
  'email_code_invalid': '邮箱验证码错误',
  'email_code_expired': '邮箱验证码已过期，请重新获取',
  'email_code_send_too_frequent': '验证码发送过于频繁，请稍后再试',
  'register_email_not_configured': '系统未配置注册邮件发送，请联系管理员',
  'register_email_send_failed': '验证码发送失败，请稍后再试',
  'failed to query user': '用户查询失败，请稍后重试',
  'failed to initialize email verification table': '验证码服务初始化失败，请稍后重试',
  'failed to create email verification code': '验证码生成失败，请稍后重试',
  'failed to generate email code': '验证码生成失败，请稍后重试',
  'failed to hash password': '密码处理失败，请稍后重试',
  'failed to register': '注册失败，请稍后重试',
  'insufficient role': '权限不足',
  'otp required': '请输入 2FA 验证码',
  'invalid otp': '2FA 验证码错误',
  'otp already enabled': '2FA 已绑定',
  'otp not enabled': '2FA 未绑定',
  'otp not setup': '请先绑定 2FA',
  'personal_app_limit_reached': '已达到个人应用创建上限',
  'app_pending_review': '应用待审核，暂不可操作',
  'app_rejected': '应用已被驳回，暂不可操作',
  'site_name required': '请输入站点名称',
  'site_name too long': '站点名称过长',
  'home_page_announcement_content too long': '首页公告内容过长',
  'apps_page_announcement_content too long': '应用管理页面公告内容过长',
  'page_announcement_content too long': '页面公告内容过长',
  'register_email_code_template too long': '注册邮件模板内容过长',
  'service_status_overall_message too long': '服务状态总览说明过长',
  'service_status_announcement too long': '服务状态公告过长',
  'service_status_components invalid': '服务组件配置无效，请检查名称和内容长度',
  'service_status_incidents invalid': '事件历史配置无效，请检查标题和内容长度',
  'failed to encode service_status_components': '服务组件配置保存失败',
  'failed to encode service_status_incidents': '事件历史配置保存失败',
  'no settings to update': '没有可更新的设置项',
  'failed to load system settings': '加载系统设置失败',
  'failed to save system settings': '保存系统设置失败',
  'failed to initialize system settings table': '系统设置表初始化失败，请检查数据库权限',
  'user_register_disabled': '当前系统已关闭新用户注册',
  'enterprise_register_disabled': '当前系统已关闭企业用户注册',
  'leave_other_orgs_required': '请先退出其他组织后再升级企业认证',
  'failed to load smtp password': '加载 SMTP 密码失败',
  'smtp_host required': '请输入 SMTP 服务器',
  'smtp_port invalid': 'SMTP 端口无效',
  'smtp_conn_ttl_seconds invalid': 'SMTP 连接有效期无效',
  'smtp_sender_email required': '请输入发件人邮箱',
  'smtp_sender_email invalid': '发件人邮箱格式错误',
  'smtp_username required': '请输入 SMTP 用户名',
  'smtp_password required': '请先配置 SMTP 密码',
  'role_name required': '请输入权限类型',
  'role_name too long': '权限类型长度超限',
  'role_name reserved': '该权限类型为系统保留，不能使用',
  'role already exists': '角色已存在',
  'role in use': '该角色正在被成员使用，无法删除',
  'invalid permission_code': '权限点无效',
  'failed to save role permissions': '保存权限配置失败',
  'failed to load role permissions': '加载角色权限失败',
  'builtin role cannot be disabled': '系统内置权限类型不能禁用',
  'failed to list org roles': '加载角色失败',
  'failed to update org role': '更新角色失败',
  'failed to delete org role': '删除角色失败',
  'failed to check role usage': '检查角色使用情况失败',
  'analytics_refresh_in_progress': '当前应用正在刷新分析数据，请稍后重试',
  'failed to refresh analytics': '刷新分析数据失败'
}

export const getErrorMessage = (err: any, fallback: string) => {
  const raw = err?.response?.data?.error
  if (typeof raw === 'string' && ERROR_I18N_MAP[raw]) {
    return ERROR_I18N_MAP[raw]
  }
  const detail = err?.response?.data?.detail
  if (typeof detail === 'string' && detail.trim()) {
    return detail
  }
  return raw || fallback
}

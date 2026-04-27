export const buildAbsoluteRedirect = (pathname: string, search?: string, hash?: string) => {
  const path = pathname || '/'
  const query = search || ''
  const fragment = hash || ''
  return `${window.location.origin}${path}${query}${fragment}`
}

export const getSafeRedirectPath = (value: string | null | undefined, fallback: string) => {
  const raw = (value || '').trim()
  if (!raw) return fallback

  const normalizePath = (path: string) => {
    if (!path.startsWith('/')) return fallback
    if (path.startsWith('//')) return fallback
    if (path.startsWith('/login') || path.startsWith('/admin-login')) return fallback
    return path
  }

  if (raw.startsWith('/')) {
    return normalizePath(raw)
  }

  try {
    const parsed = new URL(raw)
    if (parsed.origin !== window.location.origin) return fallback
    return normalizePath(`${parsed.pathname}${parsed.search}${parsed.hash}`)
  } catch {
    return fallback
  }
}
